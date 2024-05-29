package kafkacontroller

import (
	"context"
	"fmt"
	"github.com/exoscale/egoscale/v2/oapi"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"strings"

	exoscalesdk "github.com/exoscale/egoscale/v2"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/pipelineutil"
	"github.com/vshn/provider-exoscale/operator/webhook"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
)

const serviceType = "kafka"

// SetupWebhook adds a webhook for kafka resources.
func SetupWebhook(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&exoscalev1.Kafka{}).
		WithValidator(&Validator{
			log:  mgr.GetLogger().WithName("webhook").WithName(strings.ToLower(exoscalev1.KafkaKind)),
			kube: mgr.GetClient(),
		}).
		Complete()
}

// Validator validates kafka admission requests.
type Validator struct {
	log  logr.Logger
	kube client.Client
}

// ValidateCreate validates the spec of a created kafka resource.
func (v *Validator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	instance := obj.(*exoscalev1.Kafka)
	v.log.V(1).Info("get kafka available versions")
	exo, err := pipelineutil.OpenExoscaleClient(ctx, v.kube, instance.GetProviderConfigName(), exoscalesdk.ClientOptWithAPIEndpoint(fmt.Sprintf("https://api-%s.exoscale.com", instance.Spec.ForProvider.Zone)))
	if err != nil {
		return nil, fmt.Errorf("open exoscale client failed: %w", err)
	}
	return nil, v.validateCreateWithExoClient(ctx, obj, exo.Exoscale)
}

func (v *Validator) validateCreateWithExoClient(ctx context.Context, obj runtime.Object, exo oapi.ClientWithResponsesInterface) error {
	instance, ok := obj.(*exoscalev1.Kafka)
	if !ok {
		return fmt.Errorf("invalid managed resource type %T for kafka webhook", obj)
	}
	v.log.V(2).WithValues("instance", instance).Info("validate create")

	if instance.Spec.ForProvider.Version != "" {
		availableVersions, err := v.getAvailableVersions(ctx, exo)
		if err != nil {
			return err
		}

		err = v.validateVersion(ctx, obj, *availableVersions)
		if err != nil {
			return fmt.Errorf("invalid version, allowed versions are %v: %w", *availableVersions, err)
		}
	}

	return validateSpec(instance.Spec.ForProvider)
}

func (v *Validator) getAvailableVersions(ctx context.Context, exo oapi.ClientWithResponsesInterface) (*[]string, error) {
	// get kafka available versions
	resp, err := exo.GetDbaasServiceTypeWithResponse(ctx, serviceType)
	if err != nil {
		return nil, fmt.Errorf("get DBaaS service type failed: %w", err)
	}

	v.log.V(1).Info("DBaaS service type", "body", string(resp.Body))

	serviceType := *resp.JSON200
	if serviceType.AvailableVersions == nil {
		return nil, fmt.Errorf("kafka available versions not found")
	}
	return serviceType.AvailableVersions, nil
}

func (v *Validator) validateVersion(_ context.Context, obj runtime.Object, availableVersions []string) error {
	instance := obj.(*exoscalev1.Kafka)

	v.log.V(1).Info("validate version")
	return webhook.ValidateVersions(instance.Spec.ForProvider.Version, availableVersions)
}

// ValidateUpdate validates the spec of an updated kafka resource and checks that no immutable field has been modified.
func (v *Validator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	newInstance, ok := newObj.(*exoscalev1.Kafka)
	if !ok {
		return nil, fmt.Errorf("invalid managed resource type %T for kafka webhook", newObj)
	}
	oldInstance, ok := oldObj.(*exoscalev1.Kafka)
	if !ok {
		return nil, fmt.Errorf("invalid managed resource type %T for kafka webhook", oldObj)
	}
	v.log.V(2).WithValues("old", oldInstance, "new", newInstance).Info("VALIDATE update")

	err := validateSpec(newInstance.Spec.ForProvider)
	if err != nil {
		return nil, err
	}
	return nil, validateImmutable(*oldInstance, *newInstance)
}

// ValidateDelete validates a delete. Currently does not validate anything.
func (v *Validator) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	v.log.V(2).Info("validate delete (noop)")
	return nil, nil
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
	if oldInst.Spec.ForProvider.Version == "" {
		// Fall back to reported version if no version was set before
		oldInst.Spec.ForProvider.Version = oldInst.Status.AtProvider.Version
	}
	return webhook.ValidateUpdateVersion(oldInst.Status.AtProvider.Version, oldInst.Spec.ForProvider.Version, newInst.Spec.ForProvider.Version)
}
