package postgresqlcontroller

import (
	"context"
	"fmt"

	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/webhook"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
)

var admittedVersions = []string{"14"}

// Validator validates admission requests.
type Validator struct {
	log logr.Logger
}

// ValidateCreate implements admission.CustomValidator.
func (v *Validator) ValidateCreate(_ context.Context, obj runtime.Object) error {
	instance := obj.(*exoscalev1.PostgreSQL)
	v.log.V(1).Info("Validate create")

	return v.validateSpec(instance)
}

// ValidateUpdate implements admission.CustomValidator.
func (v *Validator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) error {
	newInstance, ok := newObj.(*exoscalev1.PostgreSQL)
	if !ok {
		return fmt.Errorf("invalid managed resource type %T for postgres webhook", newObj)
	}
	oldInstance, ok := oldObj.(*exoscalev1.PostgreSQL)
	if !ok {
		return fmt.Errorf("invalid managed resource type %T for postgres webhook", oldObj)
	}
	v.log.V(1).Info("Validate update")

	err := v.validateSpec(newInstance)
	if err != nil {
		return err
	}
	return validateImmutable(*oldInstance, *newInstance)
}

// ValidateDelete implements admission.CustomValidator.
func (v *Validator) ValidateDelete(_ context.Context, obj runtime.Object) error {
	v.log.V(1).Info("Validate delete (noop)")
	return nil
}

func (v *Validator) validateSpec(obj *exoscalev1.PostgreSQL) error {
	for _, validatorFn := range []func(exoscalev1.PostgreSQLParameters) error{
		validateVersion,
		validateIpFilter,
		validateMaintenanceSchedule,
		validatePGSettings,
	} {
		if err := validatorFn(obj.Spec.ForProvider); err != nil {
			return err
		}
	}
	return nil
}

func validateVersion(obj exoscalev1.PostgreSQLParameters) error {
	return webhook.ValidateVersions(obj.Version, admittedVersions)
}

func validateIpFilter(obj exoscalev1.PostgreSQLParameters) error {
	if len(obj.IPFilter) == 0 {
		return fmt.Errorf("IP filter cannot be empty")
	}
	return nil
}

func validateMaintenanceSchedule(obj exoscalev1.PostgreSQLParameters) error {
	if _, _, _, err := obj.Maintenance.TimeOfDay.Parse(); err != nil {
		return err
	}
	return nil
}

func validatePGSettings(obj exoscalev1.PostgreSQLParameters) error {
	return webhook.ValidateRawExtension(obj.PGSettings)
}

func validateImmutable(oldInst, newInst exoscalev1.PostgreSQL) error {
	err := compareZone(oldInst.Spec.ForProvider, newInst.Spec.ForProvider)
	if err != nil {
		return err
	}
	return compareVersion(oldInst, newInst)
}

func compareZone(oldParams, newParams exoscalev1.PostgreSQLParameters) error {
	if oldParams.Zone != newParams.Zone {
		return fmt.Errorf("field is immutable: %s (old), %s (changed)", oldParams.Zone, newParams.Zone)
	}
	return nil
}

func compareVersion(oldInst, newInst exoscalev1.PostgreSQL) error {
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
