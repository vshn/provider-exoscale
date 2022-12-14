package opensearchcontroller

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

var admittedVersions = []string{"1", "2"}

// Validator validates admission requests.
type Validator struct {
	log logr.Logger
}

// ValidateCreate implements admission.CustomValidator.
func (v *Validator) ValidateCreate(_ context.Context, obj runtime.Object) error {
	openSearchInstance := obj.(*exoscalev1.OpenSearch)
	v.log.V(1).Info("validate create")

	return v.validateSpec(openSearchInstance)
}

// ValidateUpdate implements admission.CustomValidator.
func (v *Validator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) error {
	newInstance, ok := newObj.(*exoscalev1.OpenSearch)
	if !ok {
		return fmt.Errorf("invalid managed resource type %T for opensearch webhook", newObj)
	}
	oldInstance, ok := oldObj.(*exoscalev1.OpenSearch)
	if !ok {
		return fmt.Errorf("invalid managed resource type %T for opensearch webhook", oldObj)
	}
	v.log.V(1).Info("validate update")

	err := v.validateSpec(newInstance)
	if err != nil {
		return err
	}
	return v.compare(oldInstance, newInstance)
}

// ValidateDelete implements admission.CustomValidator.
func (v *Validator) ValidateDelete(_ context.Context, obj runtime.Object) error {
	v.log.V(1).Info("validate delete (noop)")
	return nil
}

func (v *Validator) validateSpec(obj *exoscalev1.OpenSearch) error {
	for _, validatorFn := range []func(exoscalev1.OpenSearchParameters) error{
		validateIpFilter,
		validateMaintenanceSchedule,
		validateSettings,
		validateVersion,
	} {
		if err := validatorFn(obj.Spec.ForProvider); err != nil {
			return err
		}
	}
	return nil
}

func validateIpFilter(obj exoscalev1.OpenSearchParameters) error {
	if len(obj.IPFilter) == 0 {
		return fmt.Errorf("IP filter cannot be empty")
	}
	return nil
}

func validateMaintenanceSchedule(obj exoscalev1.OpenSearchParameters) error {
	if _, _, _, err := obj.Maintenance.TimeOfDay.Parse(); err != nil {
		return err
	}
	return nil
}

func validateSettings(obj exoscalev1.OpenSearchParameters) error {
	return webhook.ValidateRawExtension(obj.OpenSearchSettings)
}

func validateVersion(obj exoscalev1.OpenSearchParameters) error {
	return webhook.ValidateVersions(obj.MajorVersion, admittedVersions)
}

func (v *Validator) compare(old, new *exoscalev1.OpenSearch) error {
	if !v.isCreated(old) {
		// comparing immutable fields is only necessary after creation.
		return nil
	}
	for _, compareFn := range []func(_, _ *exoscalev1.OpenSearch) error{
		v.compareZone,
		v.compareVersion,
	} {
		if err := compareFn(old, new); err != nil {
			return err
		}
	}
	return nil
}

func (v *Validator) compareZone(old, new *exoscalev1.OpenSearch) error {
	if old.Spec.ForProvider.Zone != new.Spec.ForProvider.Zone {
		return fmt.Errorf("field is immutable after creation: %s (old), %s (changed)", old.Spec.ForProvider.Zone, new.Spec.ForProvider.Zone)
	}
	return nil
}

func (v *Validator) compareVersion(old, new *exoscalev1.OpenSearch) error {
	if old.Spec.ForProvider.MajorVersion == new.Spec.ForProvider.MajorVersion {
		return nil
	}
	if new.Spec.ForProvider.MajorVersion == "" {
		// Setting version to empty string should always be fine
		return nil
	}
	if old.Spec.ForProvider.MajorVersion == "" {
		// Fall back to reported version if no version was set before
		old.Spec.ForProvider.MajorVersion = old.Status.AtProvider.MajorVersion
	}
	return webhook.ValidateUpdateVersion(old.Status.AtProvider.MajorVersion, old.Spec.ForProvider.MajorVersion, new.Spec.ForProvider.MajorVersion)
}

func (v *Validator) isCreated(obj *exoscalev1.OpenSearch) bool {
	cond := mapper.FindStatusCondition(obj.Status.Conditions, xpv1.Available().Type)
	return cond != nil
}
