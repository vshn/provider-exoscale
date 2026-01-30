package postgresqlcontroller

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	exoscalesdk "github.com/exoscale/egoscale/v3"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/common"
	"github.com/vshn/provider-exoscale/operator/pipelineutil"
	"github.com/vshn/provider-exoscale/operator/webhook"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
)

const serviceType = "pg"

// Validator validates admission requests.
type Validator struct {
	log  logr.Logger
	kube client.Client
}

// ValidateCreate implements admission.CustomValidator.
func (v *Validator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	instance, ok := obj.(*exoscalev1.PostgreSQL)
	if !ok {
		return nil, fmt.Errorf("invalid managed resource type %T for postgres webhook", obj)
	}
	v.log.V(1).Info("Validate create")

	// Validate zone exists
	warnings, err := webhook.ValidateZoneExists(ctx, string(instance.Spec.ForProvider.Zone))
	if err != nil {
		return warnings, err
	}

	availableVersions, err := v.getAvailableVersions(ctx, obj)
	if err != nil {
		return warnings, err
	}

	err = v.validateVersion(ctx, obj, availableVersions)
	if err != nil {
		return warnings, err
	}

	return warnings, v.validateSpec(instance)
}

func (v *Validator) getAvailableVersions(ctx context.Context, obj runtime.Object) ([]string, error) {
	instance := obj.(*exoscalev1.PostgreSQL)

	v.log.V(1).Info("get postgres available versions")
	exo, err := pipelineutil.OpenExoscaleClient(ctx, v.kube, instance.GetProviderConfigName(), exoscalesdk.ClientOptWithEndpoint(common.ZoneTranslation[instance.Spec.ForProvider.Zone]))
	if err != nil {
		return nil, fmt.Errorf("open exoscale client failed: %w", err)
	}

	resp, err := exo.Exoscale.GetDBAASServiceType(ctx, serviceType)
	if err != nil {
		return nil, fmt.Errorf("get DBaaS service type failed: %w", err)
	}

	v.log.V(1).Info("DBaaS service type", "name", string(resp.Name), "description", string(resp.Description))

	if resp.AvailableVersions == nil {
		return nil, fmt.Errorf("postgres available versions not found")
	}
	return resp.AvailableVersions, nil
}

func (v *Validator) validateVersion(ctx context.Context, obj runtime.Object, availableVersions []string) error {
	instance := obj.(*exoscalev1.PostgreSQL)

	v.log.V(1).Info("validate version")
	return webhook.ValidateVersions(instance.Spec.ForProvider.Version, availableVersions)
}

// ValidateUpdate implements admission.CustomValidator.
func (v *Validator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	newInstance, ok := newObj.(*exoscalev1.PostgreSQL)
	if !ok {
		return nil, fmt.Errorf("invalid managed resource type %T for postgres webhook", newObj)
	}
	oldInstance, ok := oldObj.(*exoscalev1.PostgreSQL)
	if !ok {
		return nil, fmt.Errorf("invalid managed resource type %T for postgres webhook", oldObj)
	}
	v.log.V(1).Info("Validate update")

	err := v.validateSpec(newInstance)
	if err != nil {
		return nil, err
	}
	return nil, validateImmutable(*oldInstance, *newInstance)
}

// ValidateDelete implements admission.CustomValidator.
func (v *Validator) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	v.log.V(1).Info("Validate delete (noop)")
	return nil, nil
}

func (v *Validator) validateSpec(obj *exoscalev1.PostgreSQL) error {
	for _, validatorFn := range []func(exoscalev1.PostgreSQLParameters) error{
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
	if oldInst.Spec.ForProvider.Version == "" {
		// Fall back to reported version if no version was set before
		oldInst.Spec.ForProvider.Version = oldInst.Status.AtProvider.Version
	}
	return webhook.ValidateUpdateVersion(oldInst.Status.AtProvider.Version, oldInst.Spec.ForProvider.Version, newInst.Spec.ForProvider.Version)
}
