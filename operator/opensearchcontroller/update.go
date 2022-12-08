package opensearchcontroller

import (
	"context"
	"fmt"
	"github.com/exoscale/egoscale/v2/oapi"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/mapper"
	controllerruntime "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

// Update implements managed.ExternalClient.
func (p *pipeline) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("updating resource")

	openSearchInstance := mg.(*exoscalev1.OpenSearch)

	forProvider := openSearchInstance.Spec.ForProvider
	settings, err := mapper.ToMap(forProvider.OpenSearchSettings)
	ipFilter := []string(forProvider.IPFilter)

	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("cannot map openSearchInstance settings: %w", err)
	}

	body := oapi.UpdateDbaasServiceOpensearchJSONRequestBody{
		Maintenance: &struct {
			Dow  oapi.UpdateDbaasServiceOpensearchJSONBodyMaintenanceDow `json:"dow"`
			Time string                                                  `json:"time"`
		}{
			Dow:  oapi.UpdateDbaasServiceOpensearchJSONBodyMaintenanceDow(forProvider.Maintenance.DayOfWeek),
			Time: forProvider.Maintenance.TimeOfDay.String()},
		OpensearchSettings:    &settings,
		Plan:                  &forProvider.Size.Plan,
		IpFilter:              &ipFilter,
		TerminationProtection: &forProvider.TerminationProtection,
	}

	resp, err := p.exo.UpdateDbaasServiceOpensearchWithResponse(ctx, oapi.DbaasServiceName(openSearchInstance.GetInstanceName()), body)
	if err != nil {
		log.V(1).Error(err, "Failed do UPDATE resource, ", "instance name: ", openSearchInstance.GetInstanceName())
		return managed.ExternalUpdate{}, err
	}
	log.V(1).Info("response", "body", string(resp.Body))
	return managed.ExternalUpdate{}, nil
}
