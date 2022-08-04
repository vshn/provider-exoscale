package iamkeycontroller

import (
	"context"
	"fmt"
	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	v2 "github.com/exoscale/egoscale/v2"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/steps"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
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

	pipe := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).
		WithSteps(
			pipeline.NewStepFromFunc("create IAM key", p.createIAMKeyFn(iam)),
			pipeline.If(hasSecretRef(iam),
				pipeline.NewStepFromFunc("create credentials secret", p.createCredentialsSecretFn(iam)),
			),
			pipeline.NewStepFromFunc("emit event", p.emitCreationEventFn(iam)),
		)
	result := pipe.RunWithContext(ctx)
	if result.IsFailed() {
		return managed.ExternalCreation{}, errors.Wrap(result.Err(), "cannot create IAM Key")
	}

	return managed.ExternalCreation{ConnectionDetails: toConnectionDetails(p.exoscaleIAMKey)}, nil
}

// createIAMKeyFn creates a new IAMKey in the project associated with the API Key and Secret.
func (p *IAMKeyPipeline) createIAMKeyFn(iamKey *exoscalev1.IAMKey) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		exoscaleClient := p.exoscaleClient
		log := controllerruntime.LoggerFrom(ctx)

		var keyResources []v2.IAMAccessKeyResource
		for _, bucket := range iamKey.Spec.ForProvider.SOS.Buckets {
			keyResource := v2.IAMAccessKeyResource{
				Domain:       SOSResourceDomain,
				ResourceName: bucket,
				ResourceType: BucketResourceType,
			}
			keyResources = append(keyResources, keyResource)
		}

		exoscaleIAM, err := exoscaleClient.CreateIAMAccessKey(ctx, iamKey.GetZone(), iamKey.GetKeyName(), v2.CreateIAMAccessKeyWithResources(keyResources))
		if err != nil {
			return err
		}

		if err != nil {
			return err
		}
		// Limitation by crossplane: The interface managed.ExternalClient doesn't allow updating the resource during creation except annotations.
		// But we need to somehow store the key ID returned by the creation operation, because exoscale API allows multiple IAM Keys with the same display name.
		// So we store it in an annotation since that is the only allowed place to update our resource.
		// However, once we observe the spec again, we will copy the key ID from the annotation to the status field,
		//  and that will become the authoritative source of truth for future reconciliations.
		metav1.SetMetaDataAnnotation(&iamKey.ObjectMeta, KeyIDAnnotationKey, *exoscaleIAM.Key)
		metav1.SetMetaDataAnnotation(&iamKey.ObjectMeta, BucketsAnnotationKey, getBuckets(*exoscaleIAM.Resources))

		log.V(1).Info("Created IAM Key in exoscale", "keyID", *exoscaleIAM.Key, "displayName", *exoscaleIAM.Name, "tags", exoscaleIAM.Tags)
		p.exoscaleIAMKey = exoscaleIAM
		return nil
	}
}

func getBuckets(iamResources []v2.IAMAccessKeyResource) string {
	var buckets strings.Builder
	for _, iamResource := range iamResources {
		buckets.WriteString(",")
		if iamResource.Domain == SOSResourceDomain {
			buckets.WriteString(iamResource.ResourceName)
		}
	}
	return buckets.String()[1:buckets.Len()]
}

func (p *IAMKeyPipeline) emitCreationEventFn(obj runtime.Object) func(ctx context.Context) error {
	return func(_ context.Context) error {
		p.recorder.Event(obj, event.Event{
			Type:    event.TypeNormal,
			Reason:  "Created",
			Message: "IAMKey successfully created",
		})
		return nil
	}
}

// createCredentialsSecretFn creates the secret with AIMKey's S3 credentials.
func (p *IAMKeyPipeline) createCredentialsSecretFn(iamKey *exoscalev1.IAMKey) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		kube := p.kube
		exoscaleIAMKey := p.exoscaleIAMKey
		log := controllerruntime.LoggerFrom(ctx)

		secretRef := iamKey.Spec.WriteConnectionSecretToReference

		p.credentialsSecret = &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretRef.Name, Namespace: secretRef.Namespace}}
		_, err := controllerruntime.CreateOrUpdate(ctx, kube, p.credentialsSecret, func() error {
			secret := p.credentialsSecret
			secret.Labels = labels.Merge(secret.Labels, getCommonLabels(iamKey.Name))
			if secret.Data == nil {
				secret.Data = map[string][]byte{}
			}
			for k, v := range toConnectionDetails(exoscaleIAMKey) {
				secret.Data[k] = v
			}
			return controllerutil.SetOwnerReference(iamKey, secret, kube.Scheme())
		})
		if err != nil {
			return err
		}
		log.V(1).Info("Created credential secret", "secretName", fmt.Sprintf("%s/%s", secretRef.Namespace, secretRef.Name))
		return nil
	}
}

func getCommonLabels(instanceName string) labels.Set {
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
	return labels.Set{
		"app.kubernetes.io/instance":   instanceName,
		"app.kubernetes.io/managed-by": exoscalev1.Group,
		"app.kubernetes.io/created-by": fmt.Sprintf("controller-%s", strings.ToLower(exoscalev1.IAMKeyKind)),
	}
}
