package iamkeycontroller

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/equality"

	"github.com/go-logr/logr"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// IAMKeyValidator validates admission requests.
type IAMKeyValidator struct {
	log logr.Logger
}

// ValidateCreate implements admission.CustomValidator.
func (v *IAMKeyValidator) ValidateCreate(_ context.Context, obj runtime.Object) error {
	iamKey := obj.(*exoscalev1.IAMKey)
	v.log.V(1).Info("Validate create", "name", iamKey.Name)
	if len(iamKey.Spec.ForProvider.Services.SOS.Buckets) == 0 {
		return fmt.Errorf("an IAMKey named %q should have at least 1 allowed bucket",
			iamKey.Name)
	}
	secretRef := iamKey.Spec.WriteConnectionSecretToReference
	if secretRef == nil || secretRef.Name == "" || secretRef.Namespace == "" {
		return fmt.Errorf("an IAMKey named %q requires a connection secret reference with name and namespace",
			iamKey.Name)
	}
	return nil
}

// ValidateUpdate implements admission.CustomValidator.
func (v *IAMKeyValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) error {
	newIAMKey := newObj.(*exoscalev1.IAMKey)
	oldIAMKey := oldObj.(*exoscalev1.IAMKey)
	v.log.V(1).Info("Validate update")

	if oldIAMKey.Status.AtProvider.KeyID != "" {
		if !equality.Semantic.DeepEqual(newIAMKey.Spec.ForProvider, oldIAMKey.Spec.ForProvider) {
			return fmt.Errorf("an IAMKey named %q has been created already, you cannot update it",
				oldIAMKey.Name)
		}
		if !equality.Semantic.DeepEqual(newIAMKey.Spec.WriteConnectionSecretToReference, oldIAMKey.Spec.WriteConnectionSecretToReference) {
			return fmt.Errorf("an IAMKey named %q has been created already, you cannot update the connection secret reference",
				oldIAMKey.Name)
		}
	}
	return nil
}

// ValidateDelete implements admission.CustomValidator.
func (v *IAMKeyValidator) ValidateDelete(_ context.Context, obj runtime.Object) error {
	res := obj.(*exoscalev1.IAMKey)
	v.log.V(1).Info("Validate delete (noop)", "name", res.Name)
	return nil
}
