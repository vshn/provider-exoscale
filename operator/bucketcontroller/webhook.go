package bucketcontroller

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// BucketValidator validates admission requests.
type BucketValidator struct {
	log logr.Logger
}

// ValidateCreate implements admission.CustomValidator.
func (v *BucketValidator) ValidateCreate(_ context.Context, obj runtime.Object) error {
	bucket := obj.(*exoscalev1.Bucket)
	v.log.V(1).Info("Validate create", "name", bucket.Name)

	providerConfigRef := bucket.Spec.ProviderConfigReference
	if providerConfigRef == nil || providerConfigRef.Name == "" {
		return fmt.Errorf(".spec.providerConfigRef.name is required")
	}
	return nil
}

// ValidateUpdate implements admission.CustomValidator.
func (v *BucketValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) error {
	newBucket := newObj.(*exoscalev1.Bucket)
	oldBucket := oldObj.(*exoscalev1.Bucket)
	v.log.V(1).Info("Validate update")

	if oldBucket.Status.AtProvider.BucketName != "" {
		if newBucket.GetBucketName() != oldBucket.Status.AtProvider.BucketName {
			return fmt.Errorf("a bucket named %q has been created already, you cannot rename it",
				oldBucket.Status.AtProvider.BucketName)
		}
		if newBucket.Spec.ForProvider.Zone != oldBucket.Spec.ForProvider.Zone {
			return fmt.Errorf("a bucket named %q has been created already, you cannot change the zone",
				oldBucket.Status.AtProvider.BucketName)
		}
	}
	providerConfigRef := newBucket.Spec.ProviderConfigReference
	if providerConfigRef == nil || providerConfigRef.Name == "" {
		return fmt.Errorf(".spec.providerConfigRef.name is required")
	}
	return nil
}

// ValidateDelete implements admission.CustomValidator.
func (v *BucketValidator) ValidateDelete(_ context.Context, obj runtime.Object) error {
	res := obj.(*exoscalev1.Bucket)
	v.log.V(1).Info("Validate delete (noop)", "name", res.Name)
	return nil
}
