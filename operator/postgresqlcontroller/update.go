package postgresqlcontroller

import (
	"context"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/vshn/provider-exoscale/operator/mapper"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Update implements managed.ExternalClient.
func (p *Pipeline) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("Updating resource")

	pgInstance := fromManaged(mg)

	spec := pgInstance.Spec.ForProvider
	mpr := &mapper.PostgreSQLMapper{}
	body := oapi.UpdateDbaasServicePgJSONRequestBody{}
	err := mpr.FromSpecToUpdateBody(spec, &body)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot map spec to API request")
	}
	resp, err := p.exo.UpdateDbaasServicePgWithResponse(ctx, oapi.DbaasServiceName(pgInstance.Name), body)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update instance")
	}
	log.V(1).Info("Response", "json", resp.JSON200)
	return managed.ExternalUpdate{}, nil
}
