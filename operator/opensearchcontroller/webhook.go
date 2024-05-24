package opensearchcontroller

import (
	"context"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	exoscalesdk "github.com/exoscale/egoscale/v2"
	"github.com/go-logr/logr"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/mapper"
	"github.com/vshn/provider-exoscale/operator/pipelineutil"
	"github.com/vshn/provider-exoscale/operator/webhook"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const serviceType = "opensearch"

// Validator validates admission requests.
type Validator struct {
	log  logr.Logger
	kube client.Client
}

// ValidateCreate implements admission.CustomValidator.
func (v *Validator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	openSearchInstance, ok := obj.(*exoscalev1.OpenSearch)
	if !ok {
		return nil, fmt.Errorf("invalid managed resource type %T for opensearch webhook", obj)
	}
	v.log.V(1).Info("validate create")

	availableVersions, err := v.getAvailableVersions(ctx, obj)
	if err != nil {
		return nil, err
	}

	err = v.validateVersion(ctx, obj, *availableVersions)
	if err != nil {
		return nil, err
	}

	return nil, v.validateSpec(openSearchInstance)
}

func (v *Validator) getAvailableVersions(ctx context.Context, obj runtime.Object) (*[]string, error) {
	openSearchInstance := obj.(*exoscalev1.OpenSearch)

	v.log.V(1).Info("get opensearch available versions")
	exo, err := pipelineutil.OpenExoscaleClient(ctx, v.kube, openSearchInstance.GetProviderConfigReference().Name, exoscalesdk.ClientOptWithAPIEndpoint(fmt.Sprintf("https://api-%s.exoscale.com", openSearchInstance.Spec.ForProvider.Zone)))
	if err != nil {
		return nil, fmt.Errorf("open exoscale client failed: %w", err)
	}

	// get opensearch available versions
	resp, err := exo.Exoscale.GetDbaasServiceTypeWithResponse(ctx, serviceType)
	if err != nil {
		return nil, fmt.Errorf("get DBaaS service type failed: %w", err)
	}

	v.log.V(1).Info("DBaaS service type", "body", string(resp.Body))

	serviceType := *resp.JSON200
	if serviceType.AvailableVersions == nil {
		return nil, fmt.Errorf("opensearch available versions not found")
	}
	return serviceType.AvailableVersions, nil
}

func (v *Validator) validateVersion(_ context.Context, obj runtime.Object, availableVersions []string) error {
	openSearchInstance := obj.(*exoscalev1.OpenSearch)

	v.log.V(1).Info("validate version")
	return webhook.ValidateVersions(openSearchInstance.Spec.ForProvider.MajorVersion, availableVersions)
}

// ValidateUpdate implements admission.CustomValidator.
func (v *Validator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	newInstance, ok := newObj.(*exoscalev1.OpenSearch)
	if !ok {
		return nil, fmt.Errorf("invalid managed resource type %T for opensearch webhook", newObj)
	}
	oldInstance, ok := oldObj.(*exoscalev1.OpenSearch)
	if !ok {
		return nil, fmt.Errorf("invalid managed resource type %T for opensearch webhook", oldObj)
	}
	v.log.V(1).Info("validate update")

	err := v.validateSpec(newInstance)
	if err != nil {
		return nil, err
	}
	return nil, v.compare(oldInstance, newInstance)
}

// ValidateDelete implements admission.CustomValidator.
func (v *Validator) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	v.log.V(1).Info("validate delete (noop)")
	return nil, nil
}

func (v *Validator) validateSpec(obj *exoscalev1.OpenSearch) error {
	for _, validatorFn := range []func(exoscalev1.OpenSearchParameters) error{
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
