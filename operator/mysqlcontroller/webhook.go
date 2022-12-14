package mysqlcontroller

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/webhook"
	"k8s.io/apimachinery/pkg/runtime"
)

var admittedVersions = []string{"8"}

// Validator validates admission requests.
type Validator struct {
	log logr.Logger
}

// ValidateCreate implements admission.CustomValidator.
func (v *Validator) ValidateCreate(_ context.Context, obj runtime.Object) error {
	mySQLInstance := obj.(*exoscalev1.MySQL)
	v.log.V(1).Info("validate create")

	return validateSpec(mySQLInstance)
}

// ValidateUpdate implements admission.CustomValidator.
func (v *Validator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) error {
	newInstance, ok := newObj.(*exoscalev1.MySQL)
	if !ok {
		return fmt.Errorf("invalid managed resource type %T for mysql webhook", newObj)
	}
	oldInstance, ok := oldObj.(*exoscalev1.MySQL)
	if !ok {
		return fmt.Errorf("invalid managed resource type %T for mysql webhook", oldObj)
	}
	v.log.V(1).Info("validate update")

	err := validateSpec(newInstance)
	if err != nil {
		return err
	}
	return validateImmutable(*oldInstance, *newInstance)
}

// ValidateDelete implements admission.CustomValidator.
func (v *Validator) ValidateDelete(_ context.Context, obj runtime.Object) error {
	v.log.V(1).Info("validate delete (noop)")
	return nil
}

func validateSpec(obj *exoscalev1.MySQL) error {
	for _, validatorFn := range []func(exoscalev1.MySQLParameters) error{
		validateVersion,
		validateIpFilter,
		validateMaintenanceSchedule,
		validateSettings,
	} {
		if err := validatorFn(obj.Spec.ForProvider); err != nil {
			return err
		}
	}
	return nil
}

func validateVersion(obj exoscalev1.MySQLParameters) error {
	return webhook.ValidateVersions(obj.Version, admittedVersions)
}

func validateIpFilter(obj exoscalev1.MySQLParameters) error {
	if len(obj.IPFilter) == 0 {
		return fmt.Errorf("IP filter cannot be empty")
	}
	return nil
}

func validateMaintenanceSchedule(obj exoscalev1.MySQLParameters) error {
	if _, _, _, err := obj.Maintenance.TimeOfDay.Parse(); err != nil {
		return err
	}
	return nil
}

func validateSettings(obj exoscalev1.MySQLParameters) error {
	return webhook.ValidateRawExtension(obj.MySQLSettings)
}

func validateImmutable(oldInst, newInst exoscalev1.MySQL) error {
	err := compareZone(oldInst.Spec.ForProvider, newInst.Spec.ForProvider)
	if err != nil {
		return err
	}
	return compareVersion(oldInst, newInst)
}

func compareZone(oldParams, newParams exoscalev1.MySQLParameters) error {
	if oldParams.Zone != newParams.Zone {
		return fmt.Errorf("field is immutable: %s (old), %s (changed)", oldParams.Zone, newParams.Zone)
	}
	return nil
}

func compareVersion(oldInst, newInst exoscalev1.MySQL) error {
	if oldInst.Spec.ForProvider.Version == newInst.Spec.ForProvider.Version {
		return nil
	}
	if newInst.Spec.ForProvider.Version == "" {
		// Setting version to empty string should always be fine
		return nil
	}
	if oldInst.Spec.ForProvider.Version == "" {
		// Fall back to reported version if no version was set before
		oldInst.Spec.ForProvider.Version = oldInst.Status.AtProvider.Version
	}
	return webhook.ValidateUpdateVersion(oldInst.Status.AtProvider.Version, oldInst.Spec.ForProvider.Version, newInst.Spec.ForProvider.Version)
}
