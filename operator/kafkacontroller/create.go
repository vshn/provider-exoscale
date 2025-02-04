package kafkacontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	exoscalesdk "github.com/exoscale/egoscale/v3"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Create idempotently creates a Kafka instance.
// It will not return an "already exits" error.
func (p *pipeline) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("creating resource")

	instance, ok := mg.(*exoscalev1.Kafka)
	if !ok {
		return managed.ExternalCreation{}, fmt.Errorf("invalid managed resource type %T for kafka connection", mg)
	}

	spec := instance.Spec.ForProvider
	ipFilter := spec.IPFilter
	settings := exoscalesdk.JSONSchemaKafka{}
	if len(spec.KafkaSettings.Raw) != 0 {
		err := json.Unmarshal(spec.KafkaSettings.Raw, &settings)
		if err != nil {
			return managed.ExternalCreation{}, fmt.Errorf("cannot map kafkaInstance settings: %w", err)
		}
	}
	restSettings := exoscalesdk.JSONSchemaKafkaRest{}
	if len(spec.KafkaRestSettings.Raw) != 0 {
		err := json.Unmarshal(spec.KafkaRestSettings.Raw, &restSettings)
		if err != nil {
			return managed.ExternalCreation{}, fmt.Errorf("invalid kafka rest settings: %w", err)
		}
	}

	body := exoscalesdk.CreateDBAASServiceKafkaRequest{
		IPFilter:      ipFilter,
		KafkaSettings: settings,
		Maintenance: &exoscalesdk.CreateDBAASServiceKafkaRequestMaintenance{
			Dow:  exoscalesdk.CreateDBAASServiceKafkaRequestMaintenanceDow(spec.Maintenance.DayOfWeek),
			Time: spec.Maintenance.TimeOfDay.String(),
		},
		Plan:                  spec.Size.Plan,
		Version:               spec.Version,
		TerminationProtection: &spec.TerminationProtection,
		KafkaRestEnabled:      &spec.KafkaRestEnabled,
		KafkaRestSettings:     restSettings,
	}

	resp, err := p.exo.CreateDBAASServiceKafka(ctx, instance.GetInstanceName(), body)
	if err != nil {
		if strings.Contains(err.Error(), "Service name is already taken") {
			// According to the ExternalClient Interface, create needs to be idempotent.
			// However the exoscale client doesn't return very helpful errors, so we need to make this brittle matching to find if we get an already exits error
			return managed.ExternalCreation{}, nil
		}
		return managed.ExternalCreation{}, fmt.Errorf("unable to create instance: %w", err)
	}
	log.V(2).Info("response", "message", string(resp.Message))
	return managed.ExternalCreation{}, nil
}
