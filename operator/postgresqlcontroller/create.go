package postgresqlcontroller

import (
	"context"
	"fmt"

	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/mapper"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/exoscale/egoscale/v2/oapi"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Create implements managed.ExternalClient.
func (p *pipeline) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("Creating resource")

	pgInstance := fromManaged(mg)

	spec := pgInstance.Spec.ForProvider

	body, err := fromSpecToCreateBody(spec)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot map spec to API request")
	}
	resp, err := p.exo.CreateDbaasServicePgWithResponse(ctx, oapi.DbaasServiceName(pgInstance.Name), body)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create instance")
	}
	log.V(1).Info("Response", "json", resp.JSON200)
	return managed.ExternalCreation{}, nil
}

// fromSpecToCreateBody places the given spec into the request body.
func fromSpecToCreateBody(spec exoscalev1.PostgreSQLParameters) (oapi.CreateDbaasServicePgJSONRequestBody, error) {
	/**
	NOTE: If you change anything below, also update fromSpecToCreateBody().
	Unfortunately the generated openapi-types in exoscale are unusable for reusing same properties.
	*/
	backupSchedule, err := mapper.ToBackupSchedule(spec.Backup.TimeOfDay)
	if err != nil {
		return oapi.CreateDbaasServicePgJSONRequestBody{}, fmt.Errorf("invalid backup schedule: %w", err)
	}
	settings, err := mapper.ToMap(spec.PGSettings)
	if err != nil {
		return oapi.CreateDbaasServicePgJSONRequestBody{}, fmt.Errorf("invalid pgsettings: %w", err)
	}

	return oapi.CreateDbaasServicePgJSONRequestBody{
		Plan:                  spec.Size.Plan,
		BackupSchedule:        &backupSchedule,
		Variant:               &variantAiven,
		Version:               &spec.Version,
		TerminationProtection: &spec.TerminationProtection,
		Maintenance: &struct {
			Dow  oapi.CreateDbaasServicePgJSONBodyMaintenanceDow `json:"dow"`
			Time string                                          `json:"time"`
		}{
			Dow:  oapi.CreateDbaasServicePgJSONBodyMaintenanceDow(spec.Maintenance.DayOfWeek),
			Time: spec.Maintenance.TimeOfDay.String(),
		},
		IpFilter:   mapper.ToSlicePtr(spec.IPFilter),
		PgSettings: &settings,
	}, nil
}
