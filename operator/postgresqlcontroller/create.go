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

// Create implements managed.ExternalClient.
func (p *Pipeline) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("Creating resource")

	pgInstance := fromManaged(mg)

	spec := pgInstance.Spec.ForProvider
	mpr := &mapper.PostgreSQLMapper{}
	body := oapi.CreateDbaasServicePgJSONBody{}
	err := mpr.FromSpecToCreateBody(spec, &body)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot map spec to API request")
	}
	resp, err := p.exo.CreateDbaasServicePgWithResponse(ctx, oapi.DbaasServiceName(pgInstance.Name), oapi.CreateDbaasServicePgJSONRequestBody(body))
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create instance")
	}
	log.V(1).Info("Response", "json", resp.JSON200)
	return managed.ExternalCreation{}, nil
}
