package iamkeycontroller

import (
	"context"
	"fmt"

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
	return nil
}

// ValidateUpdate implements admission.CustomValidator.
func (v *IAMKeyValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) error {
	newIAMKey := newObj.(*exoscalev1.IAMKey)
	oldIAMKey := oldObj.(*exoscalev1.IAMKey)
	v.log.V(1).Info("Validate update")

	if oldIAMKey.Status.AtProvider.KeyName != "" {
		if newIAMKey.GetKeyName() != oldIAMKey.Status.AtProvider.KeyName {
			return fmt.Errorf("an IAMKey named %q has been created already, you cannot rename it",
				oldIAMKey.Status.AtProvider.KeyName)
		}
		if newIAMKey.Spec.ForProvider.Zone != oldIAMKey.Spec.ForProvider.Zone {
			return fmt.Errorf("an IAMKey named %q has been created already, you cannot change the zone",
				oldIAMKey.Status.AtProvider.KeyName)
		}
		if stringArrayEquals(newIAMKey.Spec.ForProvider.Services.SOS.Buckets, oldIAMKey.Status.AtProvider.Services.SOS.Buckets) {
			return fmt.Errorf("a IAMKey named %q has been created already, you cannot change the bucket list",
				oldIAMKey.Status.AtProvider.KeyName)
		}
		if newIAMKey.Spec.WriteConnectionSecretToReference.Name != oldIAMKey.Spec.WriteConnectionSecretToReference.Name {
			return fmt.Errorf("an IAMKey named %q has been created already, you cannot change the connection secret name",
				oldIAMKey.Status.AtProvider.KeyName)
		}
		if newIAMKey.Spec.WriteConnectionSecretToReference.Namespace != oldIAMKey.Spec.WriteConnectionSecretToReference.Namespace {
			return fmt.Errorf("an IAMKey named %q has been created already, you cannot change the connection secret namespace",
				oldIAMKey.Status.AtProvider.KeyName)
		}
	}
	return nil
}

func stringArrayEquals(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// ValidateDelete implements admission.CustomValidator.
func (v *IAMKeyValidator) ValidateDelete(_ context.Context, obj runtime.Object) error {
	res := obj.(*exoscalev1.IAMKey)
	v.log.V(1).Info("Validate delete (noop)", "name", res.Name)
	return nil
}
