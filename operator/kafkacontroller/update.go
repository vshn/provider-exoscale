package kafkacontroller

import (
	"context"
	"fmt"

	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/mapper"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/exoscale/egoscale/v2/oapi"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Update the provided kafka instance.
func (c connection) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("updating resource")

	instance, ok := mg.(*exoscalev1.Kafka)
	if !ok {
		return managed.ExternalUpdate{}, fmt.Errorf("invalid managed resource type %T for kafka connection", mg)
	}

	spec := instance.Spec.ForProvider
	ipFilter := []string(spec.IPFilter)
	settings, err := mapper.ToMap(spec.KafkaSettings)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("invalid kafka settings: %w", err)
	}
	restSettings, err := mapper.ToMap(spec.KafkaRestSettings)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("invalid kafka rest settings: %w", err)
	}

	body := oapi.UpdateDbaasServiceKafkaJSONRequestBody{
		IpFilter:      &ipFilter,
		KafkaSettings: &settings,
		Maintenance: &struct {
			Dow  oapi.UpdateDbaasServiceKafkaJSONBodyMaintenanceDow "json:\"dow\""
			Time string                                             "json:\"time\""
		}{
			Dow:  oapi.UpdateDbaasServiceKafkaJSONBodyMaintenanceDow(spec.Maintenance.DayOfWeek),
			Time: spec.Maintenance.TimeOfDay.String(),
		},
		Plan:                  &spec.Size.Plan,
		TerminationProtection: &spec.TerminationProtection,
		KafkaRestEnabled:      &spec.KafkaRestEnabled,
		KafkaRestSettings:     &restSettings,
	}

	resp, err := c.exo.UpdateDbaasServiceKafkaWithResponse(ctx, oapi.DbaasServiceName(instance.GetInstanceName()), body)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("unable to update instance: %w", err)
	}
	log.V(2).Info("response", "body", string(resp.Body))
	return managed.ExternalUpdate{}, nil
}
