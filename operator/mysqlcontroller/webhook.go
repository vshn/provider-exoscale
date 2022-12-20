package mysqlcontroller

import (
	"context"
	"fmt"

	exoscalesdk "github.com/exoscale/egoscale/v2"
	"github.com/go-logr/logr"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/pipelineutil"
	"github.com/vshn/provider-exoscale/operator/webhook"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const serviceType = "mysql"

// Validator validates admission requests.
type Validator struct {
	log  logr.Logger
	kube client.Client
}

// ValidateCreate implements admission.CustomValidator.
func (v *Validator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	mySQLInstance, ok := obj.(*exoscalev1.MySQL)
	if !ok {
		return fmt.Errorf("invalid managed resource type %T for mysql webhook", obj)
	}

	v.log.V(1).Info("validate create")

	availableVersions, err := v.getAvailableVersions(ctx, obj)
	if err != nil {
		return err
	}

	err = v.validateVersion(ctx, obj, *availableVersions)
	if err != nil {
		return err
	}

	return validateSpec(mySQLInstance)
}

func (v *Validator) getAvailableVersions(ctx context.Context, obj runtime.Object) (*[]string, error) {
	mySQLInstance := obj.(*exoscalev1.MySQL)

	v.log.V(1).Info("get mysql available versions")
	exo, err := pipelineutil.OpenExoscaleClient(ctx, v.kube, mySQLInstance.GetProviderConfigName(), exoscalesdk.ClientOptWithAPIEndpoint(fmt.Sprintf("https://api-%s.exoscale.com", mySQLInstance.Spec.ForProvider.Zone)))
	if err != nil {
		return nil, fmt.Errorf("open exoscale client failed: %w", err)
	}

	// get mysql available versions
	resp, err := exo.Exoscale.GetDbaasServiceTypeWithResponse(ctx, serviceType)
	if err != nil {
		return nil, fmt.Errorf("get DBaaS service type failed: %w", err)
	}

	v.log.V(1).Info("DBaaS service type", "body", string(resp.Body))

	serviceType := *resp.JSON200
	if serviceType.AvailableVersions == nil {
		return nil, fmt.Errorf("mysql available versions not found")
	}
	return serviceType.AvailableVersions, nil
}

func (v *Validator) validateVersion(ctx context.Context, obj runtime.Object, availableVersions []string) error {
	mySQLInstance := obj.(*exoscalev1.MySQL)

	v.log.V(1).Info("validate version")
	return webhook.ValidateVersions(mySQLInstance.Spec.ForProvider.Version, availableVersions)
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
	if oldInst.Spec.ForProvider.Version == "" {
		// Fall back to reported version if no version was set before
		oldInst.Spec.ForProvider.Version = oldInst.Status.AtProvider.Version
	}
	return webhook.ValidateUpdateVersion(oldInst.Status.AtProvider.Version, oldInst.Spec.ForProvider.Version, newInst.Spec.ForProvider.Version)
}
