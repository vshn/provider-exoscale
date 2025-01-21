package opensearchcontroller

import (
	"context"
	"encoding/json"
	"fmt"

	exoscalesdk "github.com/exoscale/egoscale/v3"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"

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
	settings := exoscalesdk.JSONSchemaOpensearch{}
	if len(forProvider.OpenSearchSettings.Raw) != 0 {
		err := json.Unmarshal(forProvider.OpenSearchSettings.Raw, &settings)
		if err != nil {
			return managed.ExternalUpdate{}, fmt.Errorf("cannot map opensearchInstance settings: %w", err)
		}
	}
	ipFilter := []string(forProvider.IPFilter)

	body := exoscalesdk.UpdateDBAASServiceOpensearchRequest{
		Maintenance: &exoscalesdk.UpdateDBAASServiceOpensearchRequestMaintenance{
			Dow:  exoscalesdk.UpdateDBAASServiceOpensearchRequestMaintenanceDow(forProvider.Maintenance.DayOfWeek),
			Time: forProvider.Maintenance.TimeOfDay.String()},
		OpensearchSettings:    settings,
		Plan:                  forProvider.Size.Plan,
		IPFilter:              ipFilter,
		TerminationProtection: &forProvider.TerminationProtection,
	}

	resp, err := p.exo.UpdateDBAASServiceOpensearch(ctx, openSearchInstance.GetInstanceName(), body)
	if err != nil {
		log.V(1).Error(err, "Failed do UPDATE resource, ", "instance name: ", openSearchInstance.GetInstanceName())
		return managed.ExternalUpdate{}, err
	}
	log.V(1).Info("response", "message", string(resp.Message))
	return managed.ExternalUpdate{}, nil
}
