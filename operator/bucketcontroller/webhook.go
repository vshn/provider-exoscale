package bucketcontroller

import (
	"context"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/go-logr/logr"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/webhook"
	"k8s.io/apimachinery/pkg/runtime"
)

// BucketValidator validates admission requests.
type BucketValidator struct {
	log logr.Logger
}

// ValidateCreate implements admission.CustomValidator.
func (v *BucketValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	bucket := obj.(*exoscalev1.Bucket)
	v.log.V(1).Info("Validate create", "name", bucket.Name)

	providerConfigRef := bucket.Spec.ProviderConfigReference
	if providerConfigRef == nil || providerConfigRef.Name == "" {
		return nil, fmt.Errorf(".spec.providerConfigRef.name is required")
	}

	// Validate zone exists
	err := webhook.ValidateZoneExists(ctx, string(bucket.Spec.ForProvider.Zone))
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// ValidateUpdate implements admission.CustomValidator.
func (v *BucketValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	newBucket := newObj.(*exoscalev1.Bucket)
	oldBucket := oldObj.(*exoscalev1.Bucket)
	v.log.V(1).Info("Validate update")

	if oldBucket.Status.AtProvider.BucketName != "" {
		if newBucket.GetBucketName() != oldBucket.Status.AtProvider.BucketName {
			return nil, fmt.Errorf("a bucket named %q has been created already, you cannot rename it",
				oldBucket.Status.AtProvider.BucketName)
		}
		if newBucket.Spec.ForProvider.Zone != oldBucket.Spec.ForProvider.Zone {
			return nil, fmt.Errorf("a bucket named %q has been created already, you cannot change the zone",
				oldBucket.Status.AtProvider.BucketName)
		}
	}
	providerConfigRef := newBucket.Spec.ProviderConfigReference
	if providerConfigRef == nil || providerConfigRef.Name == "" {
		return nil, fmt.Errorf(".spec.providerConfigRef.name is required")
	}
	return nil, nil
}

// ValidateDelete implements admission.CustomValidator.
func (v *BucketValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	res := obj.(*exoscalev1.Bucket)
	v.log.V(1).Info("Validate delete (noop)", "name", res.Name)
	return nil, nil
}
