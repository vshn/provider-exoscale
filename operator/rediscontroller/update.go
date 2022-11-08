package rediscontroller

import (
	"context"
	"fmt"

	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/vshn/provider-exoscale/operator/mapper"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

func (p pipeline) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("updating resource")

	redisInstance := mg.(*exoscalev1.Redis)

	spec := redisInstance.Spec.ForProvider
	ipFilter := []string(spec.IPFilter)
	settings, err := mapper.ToMap(spec.RedisSettings)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("invalid redis settings: %w", err)
	}

	body := oapi.UpdateDbaasServiceRedisJSONRequestBody{
		IpFilter: &ipFilter,
		Maintenance: &struct {
			Dow oapi.UpdateDbaasServiceRedisJSONBodyMaintenanceDow `json:"dow"`

			// Time for installing updates, UTC
			Time string `json:"time"`
		}{
			Dow:  oapi.UpdateDbaasServiceRedisJSONBodyMaintenanceDow(spec.Maintenance.DayOfWeek),
			Time: spec.Maintenance.TimeOfDay.String(),
		},
		Plan:                  &spec.Size.Plan,
		RedisSettings:         &settings,
		TerminationProtection: &spec.TerminationProtection,
	}
	resp, err := p.exo.UpdateDbaasServiceRedisWithResponse(ctx, oapi.DbaasServiceName(redisInstance.GetInstanceName()), body)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("unable to create instance: %w", err)
	}
	log.V(1).Info("response", "body", string(resp.Body))
	return managed.ExternalUpdate{}, nil
}
