package iamkeycontroller

import (
	"context"
	"fmt"
	"reflect"

	pipeline "github.com/ccremer/go-command-pipeline"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalesdk "github.com/exoscale/egoscale/v3"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/pipelineutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Observe implements managed.ExternalClient.
func (p *IAMKeyPipeline) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("Observing resource")

	iamKey := fromManaged(mg)
	// to manage state of new and old keys I need other variable, this is why this annotation is set
	// otherwise observation fails for one of key types

	if iamKey.Status.AtProvider.RoleID == "" {
		// get the data generated by Create() via annotations, since in Create() we're not allowed to update the status.
		if RoleID, err := exoscalesdk.ParseUUID(iamKey.Annotations[RoleIDAnnotationKey]); err == nil {
			iamKey.Status.AtProvider.RoleID = RoleID
			delete(iamKey.Annotations, RoleIDAnnotationKey)
			log.V(1).Info("Deleting annotation", "key", RoleIDAnnotationKey)
		} else {
			// New resource, create user first
			log.V(1).Info("IAM Role not found, returning")
			return managed.ExternalObservation{}, nil
		}
	}

	if iamKey.Status.AtProvider.KeyID == "" {
		// get the data generated by Create() via annotations, since in Create() we're not allowed to update the status.
		if KeyId, exists := iamKey.Annotations[KeyIDAnnotationKey]; exists {
			iamKey.Status.AtProvider.KeyID = KeyId
			delete(iamKey.Annotations, KeyIDAnnotationKey)
			log.V(1).Info("Deleting annotation", "key", KeyIDAnnotationKey)
		} else {
			// New resource, create user first
			log.V(1).Info("IAM key not found, returning")
			return managed.ExternalObservation{}, nil
		}
	}

	pctx := &pipelineContext{Context: ctx, iamKey: iamKey}

	apiKey, err := p.exoscaleClient.GetAPIKey(ctx, iamKey.Status.AtProvider.KeyID)
	if err != nil {
		if iamKey.GetDeletionTimestamp() != nil {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{ResourceExists: false}, err
	}
	pctx.iamExoscaleKey = &exoscalesdk.AccessKey{
		Key:  apiKey.Key,
		Name: apiKey.Name,
	}

	pipe := pipeline.NewPipeline[*pipelineContext]()
	err = pipe.WithBeforeHooks(pipelineutil.DebugLogger(pctx)).
		WithSteps(
			pipe.NewStep("fetch credentials secret", p.fetchCredentialsSecret),
			pipe.NewStep("check credentials", p.checkSecret),
			pipe.NewStep("check if role is up to date", p.isRoleUptodate),
		).RunWithContext(pctx)

	if err != nil {
		log.V(2).Info("pipeline that fetches and checks credentials secret returned", "error", err)
		return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false}, nil
	}

	iamKey.Status.AtProvider.KeyName = pctx.iamExoscaleKey.Name

	if iamKey.Status.AtProvider.RoleID == "" {
		iamKey.Status.AtProvider.ServicesSpec.SOS.Buckets = getBuckets(pctx.iamExoscaleKey.Resources)
	} else {
		iamKey.Status.AtProvider.ServicesSpec.SOS.Buckets = iamKey.Spec.ForProvider.Services.SOS.Buckets
	}

	connDetails, err := toConnectionDetails(pctx.iamExoscaleKey)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("cannot parse connection details: %w", err)
	}
	log.Info("Observation successfull", "keyName", iamKey.Status.AtProvider.KeyName)
	iamKey.SetConditions(xpv1.Available())
	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ConnectionDetails: connDetails}, nil
}

func (p *IAMKeyPipeline) fetchCredentialsSecret(ctx *pipelineContext) error {
	log := controllerruntime.LoggerFrom(ctx)
	secretRef := ctx.iamKey.Spec.WriteConnectionSecretToReference
	ctx.credentialsSecret = &corev1.Secret{}
	log.Info("Fetching credentials secret during iamkey observation", secretRef.Namespace, secretRef.Name)
	err := p.kube.Get(ctx, types.NamespacedName{Namespace: secretRef.Namespace, Name: secretRef.Name}, ctx.credentialsSecret)
	if err != nil {
		log.Error(err, "Cannot fetch credentials secret")
		return err
	}
	return nil

}

func (p *IAMKeyPipeline) checkSecret(ctx *pipelineContext) error {
	data := ctx.credentialsSecret.Data
	if len(data) == 0 {
		return fmt.Errorf("secret %q does not have any data", fmt.Sprintf("%s/%s", ctx.credentialsSecret.Namespace, ctx.credentialsSecret.Name))
	}
	// Populate secret key from the secret credentials as exoscale IAM get operation does not return the secret key
	secret := string(data[exoscalev1.SecretAccessKeyName])
	ctx.iamExoscaleKey.Secret = secret
	return nil
}

func getBuckets(iamResources []exoscalesdk.AccessKeyResource) []string {
	buckets := make([]string, 0, len(iamResources))
	if len(iamResources) == 0 {
		return buckets
	}
	for _, iamResource := range iamResources {
		if iamResource.Domain == SOSResourceDomain {
			buckets = append(buckets, iamResource.ResourceName)
		}
	}
	return buckets
}

func (p *IAMKeyPipeline) isRoleUptodate(ctx *pipelineContext) error {

	// Only new keys support the roles. We don't handle legacy keys.
	if _, exists := ctx.iamKey.Annotations["newKeyType"]; !exists {
		return nil
	}

	errNotUpToDate := fmt.Errorf("roles are not equal, IAM Key is not up to date")

	obsRole, err := p.observeRole(ctx)
	if err != nil {
		return errNotUpToDate
	}

	desiredRole := createRole(ctx.iamKey.Spec.ForProvider.KeyName, ctx.iamKey.Spec.ForProvider.Services.SOS.Buckets)

	// We're only interested in the policy as most fields in the role can't be
	// changed anyway after creation.
	if !reflect.DeepEqual(obsRole.Policy, desiredRole.Policy) {
		return errNotUpToDate
	}

	return nil
}

func (p *IAMKeyPipeline) observeRole(ctx *pipelineContext) (*exoscalesdk.IAMRole, error) {

	respRole, err := p.exoscaleClient.GetIAMRole(ctx, ctx.iamKey.Status.AtProvider.RoleID)
	if err != nil || respRole.ID == "" {
		return nil, err
	}

	return respRole, nil
}
