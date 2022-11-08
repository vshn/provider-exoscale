package rediscontroller

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

func (p pipeline) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("creating resource")

	redisInstance := mg.(*exoscalev1.Redis)

	spec := redisInstance.Spec.ForProvider
	ipFilter := []string(spec.IPFilter)
	settings, err := mapper.ToMap(spec.RedisSettings)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("invalid redis settings: %w", err)
	}

	body := oapi.CreateDbaasServiceRedisJSONRequestBody{
		IpFilter: &ipFilter,
		Maintenance: &struct {
			Dow oapi.CreateDbaasServiceRedisJSONBodyMaintenanceDow `json:"dow"`

			// Time for installing updates, UTC
			Time string `json:"time"`
		}{
			Dow:  oapi.CreateDbaasServiceRedisJSONBodyMaintenanceDow(spec.Maintenance.DayOfWeek),
			Time: spec.Maintenance.TimeOfDay.String(),
		},
		Plan:                  spec.Size.Plan,
		RedisSettings:         &settings,
		TerminationProtection: &spec.TerminationProtection,
	}
	resp, err := p.exo.CreateDbaasServiceRedisWithResponse(ctx, oapi.DbaasServiceName(redisInstance.GetInstanceName()), body)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("unable to create instance: %w", err)
	}

	log.V(1).Info("response", "body", string(resp.Body))
	return managed.ExternalCreation{}, nil
}
