package rediscontroller

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

func (p pipeline) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("updating resource")

	redisInstance := mg.(*exoscalev1.Redis)

	spec := redisInstance.Spec.ForProvider
	ipFilter := []string(spec.IPFilter)
	settings := exoscalesdk.JSONSchemaRedis{}
	if len(spec.RedisSettings.Raw) != 0 {
		err := json.Unmarshal(spec.RedisSettings.Raw, &settings)
		if err != nil {
			return managed.ExternalUpdate{}, fmt.Errorf("cannot map redisInstance settings: %w", err)
		}
	}

	body := exoscalesdk.UpdateDBAASServiceRedisRequest{
		IPFilter: ipFilter,
		Maintenance: &exoscalesdk.UpdateDBAASServiceRedisRequestMaintenance{
			Dow:  exoscalesdk.UpdateDBAASServiceRedisRequestMaintenanceDow(spec.Maintenance.DayOfWeek),
			Time: spec.Maintenance.TimeOfDay.String(),
		},
		Plan:                  spec.Size.Plan,
		RedisSettings:         &settings,
		TerminationProtection: &spec.TerminationProtection,
	}
	resp, err := p.exo.UpdateDBAASServiceRedis(ctx, redisInstance.GetInstanceName(), body)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("unable to create instance: %w", err)
	}
	log.V(1).Info("response", "message", string(resp.Message))
	return managed.ExternalUpdate{}, nil
}
