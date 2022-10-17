package postgresqlcontroller

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

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
	newInstance := newObj.(*exoscalev1.PostgreSQL)
	//	oldInstance := oldObj.(*exoscalev1.PostgreSQL)
	v.log.V(1).Info("Validate update")

	return v.validateSpec(newInstance)
}

// ValidateDelete implements admission.CustomValidator.
func (v *Validator) ValidateDelete(_ context.Context, obj runtime.Object) error {
	//	instance := obj.(*exoscalev1.PostgreSQL)
	v.log.V(1).Info("Validate delete (noop)")
	return nil
}

func (v *Validator) validateSpec(obj *exoscalev1.PostgreSQL) error {
	for _, validatorFn := range []func(exoscalev1.PostgreSQLParameters) error{
		v.validateIpFilter,
		v.validateMaintenanceSchedule,
	} {
		if err := validatorFn(obj.Spec.ForProvider); err != nil {
			return err
		}
	}
	return nil
}

func (v *Validator) validateIpFilter(obj exoscalev1.PostgreSQLParameters) error {
	if len(obj.IPFilter) == 0 {
		return fmt.Errorf("IP filter cannot be empty")
	}
	return nil
}

func (v *Validator) validateMaintenanceSchedule(obj exoscalev1.PostgreSQLParameters) error {
	if _, _, _, err := obj.Maintenance.TimeOfDay.Parse(); err != nil {
		return fmt.Errorf("invalid time format (hh:mm): %w", err)
	}
	return nil
}
