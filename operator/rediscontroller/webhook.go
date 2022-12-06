package rediscontroller

import (
	"context"
	"fmt"

	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/mapper"
	"github.com/vshn/provider-exoscale/operator/webhook"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
)

// Validator validates admission requests.
type Validator struct {
	log logr.Logger
}

// ValidateCreate implements admission.CustomValidator.
func (v *Validator) ValidateCreate(_ context.Context, obj runtime.Object) error {
	instance := obj.(*exoscalev1.Redis)
	v.log.V(1).Info("validate create")

	return v.validateSpec(instance)
}

// ValidateUpdate implements admission.CustomValidator.
func (v *Validator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) error {
	newInstance := newObj.(*exoscalev1.Redis)
	oldInstance := oldObj.(*exoscalev1.Redis)
	v.log.V(1).Info("validate update")

	if newInstance.DeletionTimestamp != nil {
		v.log.V(2).Info("instance deleted, not validating updates")
		return nil
	}
	err := v.validateSpec(newInstance)
	if err != nil {
		return err
	}
	return v.compare(oldInstance, newInstance)
}

// ValidateDelete implements admission.CustomValidator.
func (v *Validator) ValidateDelete(_ context.Context, obj runtime.Object) error {
	//	instance := obj.(*exoscalev1.Redis)
	v.log.V(1).Info("validate delete (noop)")
	return nil
}

func (v *Validator) validateSpec(obj *exoscalev1.Redis) error {
	for _, validatorFn := range []func(exoscalev1.RedisParameters) error{
		v.validateIpFilter,
		v.validateMaintenanceSchedule,
		v.validateRedisSettings,
	} {
		if err := validatorFn(obj.Spec.ForProvider); err != nil {
			return err
		}
	}
	return nil
}

func (v *Validator) validateIpFilter(obj exoscalev1.RedisParameters) error {
	if len(obj.IPFilter) == 0 {
		return fmt.Errorf("IP filter cannot be empty")
	}
	return nil
}

func (v *Validator) validateMaintenanceSchedule(obj exoscalev1.RedisParameters) error {
	if _, _, _, err := obj.Maintenance.TimeOfDay.Parse(); err != nil {
		return err
	}
	return nil
}

func (v *Validator) validateRedisSettings(obj exoscalev1.RedisParameters) error {
	return webhook.ValidateRawExtension(obj.RedisSettings)
}

func (v *Validator) compare(old, new *exoscalev1.Redis) error {
	if !v.isCreated(old) {
		// comparing immutable fields is only necessary after creation.
		return nil
	}
	for _, compareFn := range []func(_, _ *exoscalev1.Redis) error{
		v.compareZone,
	} {
		if err := compareFn(old, new); err != nil {
			return err
		}
	}
	return nil
}

func (v *Validator) compareZone(old, new *exoscalev1.Redis) error {
	if old.Spec.ForProvider.Zone != new.Spec.ForProvider.Zone {
		return fmt.Errorf("field is immutable after creation: %s (old), %s (changed)", old.Spec.ForProvider.Zone, new.Spec.ForProvider.Zone)
	}
	return nil
}

func (v *Validator) isCreated(obj *exoscalev1.Redis) bool {
	cond := mapper.FindStatusCondition(obj.Status.Conditions, xpv1.Available().Type)
	return cond != nil
}
