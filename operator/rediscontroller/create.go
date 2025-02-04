package rediscontroller

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

func (p pipeline) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("creating resource")

	redisInstance := mg.(*exoscalev1.Redis)

	spec := redisInstance.Spec.ForProvider
	ipFilter := []string(spec.IPFilter)
	settings := exoscalesdk.JSONSchemaRedis{}
	if len(spec.RedisSettings.Raw) != 0 {
		err := json.Unmarshal(spec.RedisSettings.Raw, &settings)
		if err != nil {
			return managed.ExternalCreation{}, fmt.Errorf("cannot map redisInstance settings: %w", err)
		}
	}

	body := exoscalesdk.CreateDBAASServiceRedisRequest{
		IPFilter: ipFilter,
		Maintenance: &exoscalesdk.CreateDBAASServiceRedisRequestMaintenance{
			Dow:  exoscalesdk.CreateDBAASServiceRedisRequestMaintenanceDow(spec.Maintenance.DayOfWeek),
			Time: spec.Maintenance.TimeOfDay.String(),
		},
		Plan:                  spec.Size.Plan,
		RedisSettings:         &settings,
		TerminationProtection: &spec.TerminationProtection,
	}
	resp, err := p.exo.CreateDBAASServiceRedis(ctx, redisInstance.GetInstanceName(), body)
	if err != nil {
		if strings.Contains(err.Error(), "Service name is already taken") {
			// According to the ExternalClient Interface, create needs to be idempotent.
			// However the exoscale client doesn't return very helpful errors, so we need to make this brittle matching to find if we get an already exits error
			return managed.ExternalCreation{}, nil
		}
		return managed.ExternalCreation{}, fmt.Errorf("unable to create instance: %w", err)
	}

	log.V(1).Info("response", "message", string(resp.Message))
	return managed.ExternalCreation{}, nil
}
