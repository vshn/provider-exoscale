package mysqlcontroller

import (
	"context"
	"fmt"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/go-logr/logr"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/mapper"
	"github.com/vshn/provider-exoscale/operator/webhook"
	"k8s.io/apimachinery/pkg/runtime"
)

// Validator validates admission requests.
type Validator struct {
	log logr.Logger
}

// ValidateCreate implements admission.CustomValidator.
func (v *Validator) ValidateCreate(_ context.Context, obj runtime.Object) error {
	mySQLInstance := obj.(*exoscalev1.MySQL)
	v.log.V(1).Info("validate create")

	return v.validateSpec(mySQLInstance)
}

// ValidateUpdate implements admission.CustomValidator.
func (v *Validator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) error {
	newInstance := newObj.(*exoscalev1.MySQL)
	oldInstance := oldObj.(*exoscalev1.MySQL)
	v.log.V(1).Info("validate update")

	err := v.validateSpec(newInstance)
	if err != nil {
		return err
	}
	return v.compare(oldInstance, newInstance)
}

// ValidateDelete implements admission.CustomValidator.
func (v *Validator) ValidateDelete(_ context.Context, obj runtime.Object) error {
	//	mySQLInstance := obj.(*exoscalev1.MySQL)
	v.log.V(1).Info("validate delete (noop)")
	return nil
}

func (v *Validator) validateSpec(obj *exoscalev1.MySQL) error {
	for _, validatorFn := range []func(exoscalev1.MySQLParameters) error{
		v.validateIpFilter,
		v.validateMaintenanceSchedule,
		v.validateSettings,
	} {
		if err := validatorFn(obj.Spec.ForProvider); err != nil {
			return err
		}
	}
	return nil
}

func (v *Validator) validateIpFilter(obj exoscalev1.MySQLParameters) error {
	if len(obj.IPFilter) == 0 {
		return fmt.Errorf("IP filter cannot be empty")
	}
	return nil
}

func (v *Validator) validateMaintenanceSchedule(obj exoscalev1.MySQLParameters) error {
	if _, _, _, err := obj.Maintenance.TimeOfDay.Parse(); err != nil {
		return err
	}
	return nil
}

func (v *Validator) validateSettings(obj exoscalev1.MySQLParameters) error {
	return webhook.ValidateRawExtension(obj.MySQLSettings)
}

func (v *Validator) compare(old, new *exoscalev1.MySQL) error {
	if !v.isCreated(old) {
		// comparing immutable fields is only necessary after creation.
		return nil
	}
	for _, compareFn := range []func(_, _ *exoscalev1.MySQL) error{
		v.compareZone,
		v.compareVersion,
	} {
		if err := compareFn(old, new); err != nil {
			return err
		}
	}
	return nil
}

func (v *Validator) compareZone(old, new *exoscalev1.MySQL) error {
	if old.Spec.ForProvider.Zone != new.Spec.ForProvider.Zone {
		return fmt.Errorf("field is immutable after creation: %s (old), %s (changed)", old.Spec.ForProvider.Zone, new.Spec.ForProvider.Zone)
	}
	return nil
}

func (v *Validator) compareVersion(old, new *exoscalev1.MySQL) error {
	return webhook.ValidateVersion(old.Status.AtProvider.Version, old.Spec.ForProvider.Version, new.Spec.ForProvider.Version)
}

func (v *Validator) isCreated(obj *exoscalev1.MySQL) bool {
	cond := mapper.FindStatusCondition(obj.Status.Conditions, xpv1.Available().Type)
	return cond != nil
}
