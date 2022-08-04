package iamkeycontroller

import (
	"context"
	"fmt"
	pipeline "github.com/ccremer/go-command-pipeline"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/steps"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/url"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"strings"
)

// Observe implements managed.ExternalClient.
func (p *IAMKeyPipeline) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("Observing resource")

	iamKey := fromManaged(mg)
	if iamKey.Status.AtProvider.KeyID == "" {
		// get the data generated by Create() via annotations, since in Create() we're not allowed to update the status.
		if KeyId, exists := iamKey.Annotations[KeyIDAnnotationKey]; exists {
			iamKey.Status.AtProvider.KeyID = KeyId
			delete(iamKey.Annotations, KeyIDAnnotationKey)
			if iamKey.Annotations[BucketsAnnotationKey] != "" {
				iamKey.Status.AtProvider.Services = exoscalev1.Services{
					SOS: exoscalev1.SOS{
						Buckets: strings.Split(iamKey.Annotations[BucketsAnnotationKey], ","),
					},
				}
			}
		} else {
			// New resource, create user first
			return managed.ExternalObservation{}, nil
		}
	}

	err := p.getIAMKeyFn(iamKey)(ctx)
	if err != nil {
		return managed.ExternalObservation{}, resource.Ignore(isNotFound, err)
	}

	result := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).WithSteps(
		pipeline.If(hasSecretRef(iamKey),
			pipeline.NewPipeline().WithNestedSteps("observe credentials secret",
				pipeline.NewStepFromFunc("fetch credentials secret", p.fetchCredentialsSecretFn(iamKey)),
				pipeline.NewStepFromFunc("compare credentials", p.checkKey),
			).WithErrorHandler(p.observeCredentialsHandler),
		),
	).RunWithContext(ctx)
	if result.IsFailed() {
		return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ConnectionDetails: toConnectionDetails(p.exoscaleIAMKey)}, nil
	}

	iamKey.SetConditions(xpv1.Available())
	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ConnectionDetails: toConnectionDetails(p.exoscaleIAMKey)}, nil
}

// getIAMKeyFn fetches an existing IAM key from the project associated with the API Key and Secret.
func (p *IAMKeyPipeline) getIAMKeyFn(iamKey *exoscalev1.IAMKey) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		log := controllerruntime.LoggerFrom(ctx)

		exoscaleIAMKey, err := p.exoscaleClient.GetIAMAccessKey(ctx, iamKey.GetZone(), iamKey.Status.AtProvider.KeyID)
		if err != nil {
			return err
		}
		p.exoscaleIAMKey = exoscaleIAMKey
		log.V(1).Info("Fetched IAM key in exoscale", "iamID", exoscaleIAMKey.Key, "keyName", exoscaleIAMKey.Name)
		return nil
	}
}

func (p *IAMKeyPipeline) fetchCredentialsSecretFn(iamKey *exoscalev1.IAMKey) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		secretRef := iamKey.Spec.WriteConnectionSecretToReference
		p.credentialsSecret = &corev1.Secret{}

		err := p.kube.Get(ctx, types.NamespacedName{Namespace: secretRef.Namespace, Name: secretRef.Name}, p.credentialsSecret)
		return err
	}
}

func (p *IAMKeyPipeline) checkKey(_ context.Context) error {
	data := p.credentialsSecret.Data

	if len(data) == 0 {
		return fmt.Errorf("secret %q does not have any data", fmt.Sprintf("%s/%s", p.credentialsSecret.Namespace, p.credentialsSecret.Name))
	}

	key := exoscalev1.AccessKeyIDName
	desired := *p.exoscaleIAMKey.Key
	if observed, exists := p.credentialsSecret.Data[key]; !exists || string(observed) != desired {
		return fmt.Errorf("secret %q is missing key ID: %s", fmt.Sprintf("%s/%s", p.credentialsSecret.Namespace, p.credentialsSecret.Name), key)
	}

	// Populate secret key from the secret credentials as exoscale IAM get operation does not return the secret key
	secret := string(data[exoscalev1.SecretAccessKeyName])
	p.exoscaleIAMKey.Secret = &secret
	return nil
}

func (p *IAMKeyPipeline) observeCredentialsHandler(ctx context.Context, err error) error {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Error(err, "Credentials Secret needs reconciling")
	return nil
}

func isNotFound(err error) bool {
	var errResp *url.Error
	if errors.As(err, &errResp) {
		return err.(*url.Error).Err.Error() == "resource not found"
	}
	return false
}