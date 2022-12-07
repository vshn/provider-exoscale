package kafkacontroller

import (
	"context"
	"errors"
	"fmt"
	"strings"

	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/mapper"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscaleapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/egoscale/v2/oapi"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Create idempotently creates a Kafka instance.
// It will not return an "already exits" error.
func (c connection) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("creating resource")

	instance, ok := mg.(*exoscalev1.Kafka)
	if !ok {
		return managed.ExternalCreation{}, fmt.Errorf("invalid managed resource type %T for kafka connection", mg)
	}

	spec := instance.Spec.ForProvider
	ipFilter := []string(spec.IPFilter)
	settings, err := mapper.ToMap(spec.KafkaSettings)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("invalid kafka settings: %w", err)
	}

	restSettings, err := mapper.ToMap(spec.KafkaRestSettings)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("invalid kafka rest settings: %w", err)
	}
	var version *string
	if spec.Version != "" {
		version = &spec.Version
	}

	body := oapi.CreateDbaasServiceKafkaJSONRequestBody{
		IpFilter:      &ipFilter,
		KafkaSettings: &settings,
		Maintenance: &struct {
			Dow  oapi.CreateDbaasServiceKafkaJSONBodyMaintenanceDow "json:\"dow\""
			Time string                                             "json:\"time\""
		}{
			Dow:  oapi.CreateDbaasServiceKafkaJSONBodyMaintenanceDow(spec.Maintenance.DayOfWeek),
			Time: spec.Maintenance.TimeOfDay.String(),
		},
		Plan:                  spec.Size.Plan,
		Version:               version,
		TerminationProtection: &spec.TerminationProtection,
		KafkaRestEnabled:      &spec.KafkaRestEnabled,
		KafkaRestSettings:     &restSettings,
	}

	resp, err := c.exo.CreateDbaasServiceKafkaWithResponse(ctx, oapi.DbaasServiceName(instance.GetInstanceName()), body)
	if err != nil {
		if errors.Is(err, exoscaleapi.ErrInvalidRequest) && strings.Contains(err.Error(), "Service name is already taken") {
			// According to the ExternalClient Interface, create needs to be idempotent.
			// However the exoscale client doesn't return very helpful errors, so we need to make this brittle matching to find if we get an already exits error
			return managed.ExternalCreation{}, nil
		}
		return managed.ExternalCreation{}, fmt.Errorf("unable to create instance: %w", err)
	}
	log.V(2).Info("response", "body", string(resp.Body))
	return managed.ExternalCreation{}, nil
}
