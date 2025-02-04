package kafkacontroller

import (
	"context"
	"encoding/json"
	"fmt"

	exoscalesdk "github.com/exoscale/egoscale/v3"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Update the provided kafka instance.
func (p *pipeline) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("updating resource")

	instance, ok := mg.(*exoscalev1.Kafka)
	if !ok {
		return managed.ExternalUpdate{}, fmt.Errorf("invalid managed resource type %T for kafka connection", mg)
	}

	spec := instance.Spec.ForProvider
	ipFilter := []string(spec.IPFilter)
	settings := exoscalesdk.JSONSchemaKafka{}
	if len(spec.KafkaSettings.Raw) != 0 {
		err := json.Unmarshal(spec.KafkaSettings.Raw, &settings)
		if err != nil {
			return managed.ExternalUpdate{}, fmt.Errorf("cannot map kafkaInstance settings: %w", err)
		}
	}

	restSettings := exoscalesdk.JSONSchemaKafkaRest{}
	if len(spec.KafkaRestSettings.Raw) != 0 {
		err := json.Unmarshal(spec.KafkaRestSettings.Raw, &restSettings)
		if err != nil {
			return managed.ExternalUpdate{}, fmt.Errorf("invalid kafka rest settings: %w", err)
		}
	}

	body := exoscalesdk.UpdateDBAASServiceKafkaRequest{
		IPFilter:      ipFilter,
		KafkaSettings: settings,
		Maintenance: &exoscalesdk.UpdateDBAASServiceKafkaRequestMaintenance{
			Dow:  exoscalesdk.UpdateDBAASServiceKafkaRequestMaintenanceDow(spec.Maintenance.DayOfWeek),
			Time: spec.Maintenance.TimeOfDay.String(),
		},
		Plan:                  spec.Size.Plan,
		TerminationProtection: &spec.TerminationProtection,
		KafkaRestEnabled:      &spec.KafkaRestEnabled,
		KafkaRestSettings:     restSettings,
	}

	resp, err := p.exo.UpdateDBAASServiceKafka(ctx, instance.GetInstanceName(), body)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("unable to update instance: %w", err)
	}
	log.V(2).Info("response", "message", string(resp.Message))
	return managed.ExternalUpdate{}, nil
}
