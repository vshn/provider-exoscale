package iamkeycontroller

import (
	"context"
	"fmt"
	"strings"

	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalesdk "github.com/exoscale/egoscale/v2"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/pipelineutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/pointer"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Create implements managed.ExternalClient.
func (p *IAMKeyPipeline) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("Creating resource")

	iam := fromManaged(mg)
	if iam.Status.AtProvider.KeyID != "" {
		// IAMKey already exists
		return managed.ExternalCreation{}, nil
	}

	pctx := &pipelineContext{Context: ctx, iamKey: iam}
	pipe := pipeline.NewPipeline[*pipelineContext]()
	pipe.WithBeforeHooks(pipelineutil.DebugLogger(pctx)).
		WithSteps(
			pipe.NewStep("create IAM key", p.createIAMKey),
			pipe.NewStep("create credentials secret", p.createCredentialsSecret),
			pipe.NewStep("emit event", p.emitCreationEvent),
		)
	err := pipe.RunWithContext(pctx)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create IAM Key")
	}
	connDetails, err := toConnectionDetails(pctx.iamExoscaleKey)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("cannot parse connection details: %w", err)
	}

	return managed.ExternalCreation{ConnectionDetails: connDetails}, nil
}

// createIAMKey creates a new IAMKey in the project associated with the API Key and Secret.
func (p *IAMKeyPipeline) createIAMKey(ctx *pipelineContext) error {
	iamKey := ctx.iamKey
	log := controllerruntime.LoggerFrom(ctx)

	var keyResources []exoscalesdk.IAMAccessKeyResource
	for _, bucket := range iamKey.Spec.ForProvider.Services.SOS.Buckets {
		keyResource := exoscalesdk.IAMAccessKeyResource{
			Domain:       SOSResourceDomain,
			ResourceName: bucket,
			ResourceType: BucketResourceType,
		}
		keyResources = append(keyResources, keyResource)
	}

	zone := iamKey.Spec.ForProvider.Zone
	iamKeyName := iamKey.GetKeyName()
	iamKeyOpts := []exoscalesdk.CreateIAMAccessKeyOpt{
		// Allowed object storage operations on the new IAM key
		// Some permissions are excluded such as create and list all sos buckets.
		exoscalesdk.CreateIAMAccessKeyWithOperations(IAMKeyAllowedOperations),
		// The tag and resource are used to limit the permissions to object storage
		exoscalesdk.CreateIAMAccessKeyWithTags([]string{SOSResourceDomain}),
		exoscalesdk.CreateIAMAccessKeyWithResources(keyResources),
	}

	exoscaleIAM, err := p.exoscaleClient.CreateIAMAccessKey(ctx, zone, iamKeyName, iamKeyOpts...)
	if err != nil {
		return err
	}

	// Limitation by crossplane: The interface managed.ExternalClient doesn't allow updating the resource during creation except annotations.
	// But we need to somehow store the key ID returned by the creation operation, because exoscale API allows multiple IAM Keys with the same display name.
	// So we store it in an annotation since that is the only allowed place to update our resource.
	// However, once we observe the spec again, we will copy the key ID from the annotation to the status field,
	//  and that will become the authoritative source of truth for future reconciliations.
	metav1.SetMetaDataAnnotation(&ctx.iamKey.ObjectMeta, KeyIDAnnotationKey, *exoscaleIAM.Key)

	log.V(1).Info("Created IAM Key in exoscale", "keyID", *exoscaleIAM.Key, "displayName", *exoscaleIAM.Name, "tags", exoscaleIAM.Tags)
	ctx.iamExoscaleKey = exoscaleIAM
	return nil

}

func (p *IAMKeyPipeline) emitCreationEvent(ctx *pipelineContext) error {
	p.recorder.Event(ctx.iamKey, event.Event{
		Type:    event.TypeNormal,
		Reason:  "Created",
		Message: "IAMKey successfully created",
	})
	return nil

}

// createCredentialsSecret creates the secret with AIMKey's S3 credentials.
func (p *IAMKeyPipeline) createCredentialsSecret(ctx *pipelineContext) error {
	log := controllerruntime.LoggerFrom(ctx)
	secretRef := ctx.iamKey.Spec.WriteConnectionSecretToReference

	ctx.credentialsSecret = &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretRef.Name, Namespace: secretRef.Namespace}}
	_, err := controllerruntime.CreateOrUpdate(ctx, p.kube, ctx.credentialsSecret, func() error {
		secret := ctx.credentialsSecret
		secret.Labels = labels.Merge(secret.Labels, getCommonLabels(ctx.iamKey.Name))
		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}
		connDetails, err := toConnectionDetails(ctx.iamExoscaleKey)
		if err != nil {
			return fmt.Errorf("cannot parse connection details: %w", err)
		}
		for k, v := range connDetails {
			secret.Data[k] = v
		}
		secret.Immutable = pointer.Bool(true)
		return controllerutil.SetOwnerReference(ctx.iamKey, secret, p.kube.Scheme())
	})
	if err != nil {
		return err
	}
	log.V(1).Info("Created credential secret", "secretName", fmt.Sprintf("%s/%s", secretRef.Namespace, secretRef.Name))
	return nil

}

func getCommonLabels(instanceName string) labels.Set {
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
	return labels.Set{
		"app.kubernetes.io/instance":   instanceName,
		"app.kubernetes.io/managed-by": exoscalev1.Group,
		"app.kubernetes.io/created-by": fmt.Sprintf("controller-%s", strings.ToLower(exoscalev1.IAMKeyKind)),
	}
}
