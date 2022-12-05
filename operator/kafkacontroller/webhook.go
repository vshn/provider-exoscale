package kafkacontroller

import (
	"context"
	"fmt"
	"strings"

	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/webhook"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
)

// SetupWebhook adds a webhook for kafka resources.
func SetupWebhook(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&exoscalev1.Kafka{}).
		WithValidator(&Validator{
			log: mgr.GetLogger().WithName("webhook").WithName(strings.ToLower(exoscalev1.KafkaKind)),
		}).
		Complete()
}

// Validator validates kafka admission requests.
type Validator struct {
	log logr.Logger
}

// ValidateCreate validates the spec of a created kafka resource.
func (v *Validator) ValidateCreate(_ context.Context, obj runtime.Object) error {
	instance, ok := obj.(*exoscalev1.Kafka)
	if !ok {
		return fmt.Errorf("invalid managed resource type %T for kafka webhook", obj)
	}
	v.log.V(2).WithValues("instance", instance).Info("validate create")

	return validateSpec(instance.Spec.ForProvider)
}

// ValidateUpdate validates the spec of an updated kafka resource and checks that no immutable field has been modified.
func (v *Validator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) error {
	newInstance, ok := newObj.(*exoscalev1.Kafka)
	if !ok {
		return fmt.Errorf("invalid managed resource type %T for kafka webhook", newObj)
	}
	oldInstance, ok := oldObj.(*exoscalev1.Kafka)
	if !ok {
		return fmt.Errorf("invalid managed resource type %T for kafka webhook", newObj)
	}
	v.log.V(2).WithValues("old", oldInstance, "new", newInstance).Info("VALIDATE update")

	err := validateSpec(newInstance.Spec.ForProvider)
	if err != nil {
		return err
	}
	return validateImmutable(*oldInstance, *newInstance)
}

// ValidateDelete validates a delete. Currently does not validate anything.
func (v *Validator) ValidateDelete(_ context.Context, obj runtime.Object) error {
	v.log.V(2).Info("validate delete (noop)")
	return nil
}

func validateSpec(params exoscalev1.KafkaParameters) error {
	err := validateIpFilter(params)
	if err != nil {
		return err
	}
	err = validateMaintenanceSchedule(params)
	if err != nil {
		return err
	}
	return validateKafkaSettings(params)
}

func validateIpFilter(params exoscalev1.KafkaParameters) error {
	if len(params.IPFilter) == 0 {
		return fmt.Errorf("IP filter cannot be empty")
	}
	return nil
}

func validateMaintenanceSchedule(params exoscalev1.KafkaParameters) error {
	_, _, _, err := params.Maintenance.TimeOfDay.Parse()
	return err
}

func validateKafkaSettings(obj exoscalev1.KafkaParameters) error {
	return webhook.ValidateRawExtension(obj.KafkaSettings)
}

func validateImmutable(oldInst, newInst exoscalev1.Kafka) error {
	err := compareZone(oldInst.Spec.ForProvider, newInst.Spec.ForProvider)
	if err != nil {
		return err
	}
	return compareVersion(oldInst, newInst)
}

func compareZone(oldParams, newParams exoscalev1.KafkaParameters) error {
	if oldParams.Zone != newParams.Zone {
		return fmt.Errorf("field is immutable: %s (old), %s (changed)", oldParams.Zone, newParams.Zone)
	}
	return nil
}

func compareVersion(oldInst, newInst exoscalev1.Kafka) error {
	if oldInst.Spec.ForProvider.Version == newInst.Spec.ForProvider.Version {
		return nil
	}
	if newInst.Spec.ForProvider.Version == "" {
		// Setting version to empyt string should always be fine
		return nil
	}
	if oldInst.Spec.ForProvider.Version == "" {
		// Fall back to reported version if no version was set before
		oldInst.Spec.ForProvider.Version = oldInst.Status.AtProvider.Version
	}
	return webhook.ValidateVersion(oldInst.Status.AtProvider.Version, oldInst.Spec.ForProvider.Version, newInst.Spec.ForProvider.Version)
}
