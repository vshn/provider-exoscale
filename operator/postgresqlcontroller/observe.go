package postgresqlcontroller

import (
	"context"
	"net/url"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/exoscale/egoscale/v2/oapi"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/mapper"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Observe implements managed.ExternalClient.
func (p *Pipeline) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("Observing resource")

	pgInstance := fromManaged(mg)

	resp, err := p.exo.GetDbaasServicePgWithResponse(ctx, oapi.DbaasServiceName(pgInstance.GetInstanceName()))
	if err != nil {
		return ignoreNotFound(err)
	}
	pgExo := *resp.JSON200
	log.V(2).Info("Response", "raw", resp.JSON200)
	log.V(1).Info("Retrieved instance", "state", pgExo.State)
	mpr := mapper.PostgreSQLMapper{}
	err = mpr.ToStatus(resp.JSON200, &pgInstance.Status.AtProvider)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot parse instance status")
	}
	isUpToDate := mpr.IsResourceUpToDate(pgInstance, resp.JSON200)
	setConditionFromState(pgExo, pgInstance)

	ca, err := p.exo.GetDatabaseCACertificate(ctx, pgInstance.Spec.ForProvider.Zone)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot retrieve CA certificate")
	}

	connDetails, err := mpr.ToConnectionDetails(pgExo, ca)
	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: isUpToDate, ConnectionDetails: connDetails}, errors.Wrap(err, "cannot read connection details")
}

func ignoreNotFound(err error) (managed.ExternalObservation, error) {
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err.Error() == "resource not found" {
		return managed.ExternalObservation{}, nil
	}
	return managed.ExternalObservation{}, errors.Wrap(err, "cannot observe instance")
}

func setConditionFromState(pgExo oapi.DbaasServicePg, pgInstance *exoscalev1.PostgreSQL) {
	switch *pgExo.State {
	case oapi.EnumServiceStateRunning:
		pgInstance.SetConditions(exoscalev1.Running())
	case oapi.EnumServiceStateRebuilding:
		pgInstance.SetConditions(exoscalev1.Rebuilding())
	case oapi.EnumServiceStatePoweroff:
		pgInstance.SetConditions(exoscalev1.PoweredOff())
	case oapi.EnumServiceStateRebalancing:
		pgInstance.SetConditions(exoscalev1.Rebalancing())
	}
}
