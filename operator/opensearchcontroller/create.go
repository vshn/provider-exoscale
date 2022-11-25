package opensearchcontroller

import (
	"context"
	"fmt"
	"github.com/vshn/provider-exoscale/operator/mapper"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/exoscale/egoscale/v2/oapi"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Create implements managed.ExternalClient.
func (p *pipeline) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	openSearch := mg.(*exoscalev1.OpenSearch)
	forProvider := openSearch.Spec.ForProvider
	ipFilter := []string(forProvider.IPFilter)
	settings, err := mapper.ToMap(forProvider.OpenSearchSettings)
	if err != nil {
		log.V(1).Error(err, "error parsing settings in OpenSearch/Create")
		return managed.ExternalCreation{}, err
	}

	body := oapi.CreateDbaasServiceOpensearchJSONRequestBody{
		Plan:     forProvider.Size.Plan,
		IpFilter: &ipFilter,
		Maintenance: &struct {
			Dow  oapi.CreateDbaasServiceOpensearchJSONBodyMaintenanceDow `json:"dow"`
			Time string                                                  `json:"time"`
		}{
			Dow:  oapi.CreateDbaasServiceOpensearchJSONBodyMaintenanceDow(forProvider.Maintenance.DayOfWeek),
			Time: forProvider.Maintenance.TimeOfDay.String(),
		},
		OpensearchSettings:    &settings,
		TerminationProtection: &forProvider.TerminationProtection,
		// Version can be only major: ['1','2']
		Version: &forProvider.Version,
	}
	resp, err := p.exo.CreateDbaasServiceOpensearchWithResponse(ctx, oapi.DbaasServiceName(openSearch.GetInstanceName()), body)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("cannot create OpenSearch Instance: %v, \nerr: %w", openSearch.GetInstanceName(), err)
	}
	log.V(1).Info("resource created", "body", string(resp.Body))
	return managed.ExternalCreation{}, nil
}
