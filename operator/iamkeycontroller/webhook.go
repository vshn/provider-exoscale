package iamkeycontroller

import (
	"context"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

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
func (v *IAMKeyValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	iamKey := obj.(*exoscalev1.IAMKey)
	v.log.V(1).Info("Validate create", "name", iamKey.Name)
	if len(iamKey.Spec.ForProvider.Services.SOS.Buckets) == 0 {
		return nil, fmt.Errorf("an IAMKey named %q should have at least 1 allowed bucket",
			iamKey.Name)
	}
	secretRef := iamKey.Spec.WriteConnectionSecretToReference
	if secretRef == nil || secretRef.Name == "" || secretRef.Namespace == "" {
		return nil, fmt.Errorf(".spec.writeConnectionSecretToRef.name and .spec.writeConnectionSecretToRef.namespace are required")
	}

	providerConfigRef := iamKey.Spec.ProviderConfigReference
	if providerConfigRef == nil || providerConfigRef.Name == "" {
		return nil, fmt.Errorf(".spec.providerConfigRef.name is required")
	}
	return nil, nil
}

// ValidateUpdate implements admission.CustomValidator.
func (v *IAMKeyValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	newIAMKey := newObj.(*exoscalev1.IAMKey)
	oldIAMKey := oldObj.(*exoscalev1.IAMKey)
	v.log.V(1).Info("Validate update")

	if oldIAMKey.Status.AtProvider.KeyID != "" {
		if !equality.Semantic.DeepEqual(newIAMKey.Spec.ForProvider, oldIAMKey.Spec.ForProvider) {
			return nil, fmt.Errorf("an IAMKey named %q has been created already, you cannot update it",
				oldIAMKey.Name)
		}
		if !equality.Semantic.DeepEqual(newIAMKey.Spec.WriteConnectionSecretToReference, oldIAMKey.Spec.WriteConnectionSecretToReference) {
			return nil, fmt.Errorf("an IAMKey named %q has been created already, you cannot update the connection secret reference",
				oldIAMKey.Name)
		}
	}
	providerConfigRef := newIAMKey.Spec.ProviderConfigReference
	if providerConfigRef == nil || providerConfigRef.Name == "" {
		return nil, fmt.Errorf(".spec.providerConfigRef.name is required")
	}
	return nil, nil
}

// ValidateDelete implements admission.CustomValidator.
func (v *IAMKeyValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	res := obj.(*exoscalev1.IAMKey)
	v.log.V(1).Info("Validate delete (noop)", "name", res.Name)
	return nil, nil
}
