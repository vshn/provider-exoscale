package postgresqlcontroller

import (
	"context"
	"fmt"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/exoscale/egoscale/v2/oapi"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/mapper"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Update implements managed.ExternalClient.
func (p *pipeline) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("Updating resource")

	pgInstance := fromManaged(mg)

	spec := pgInstance.Spec.ForProvider
	body, err := fromSpecToUpdateBody(spec)
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

// fromSpecToUpdateBody places the given spec into the request body.
func fromSpecToUpdateBody(spec exoscalev1.PostgreSQLParameters) (oapi.UpdateDbaasServicePgJSONRequestBody, error) {
	/**
	NOTE: If you change anything below, also update fromSpecToUpdateBody().
	Unfortunately the generated openapi-types in exoscale are unusable for reusing same properties.
	*/
	backupSchedule, err := mapper.ToBackupSchedule(spec.Backup.TimeOfDay)
	if err != nil {
		return oapi.UpdateDbaasServicePgJSONRequestBody{}, fmt.Errorf("invalid backup schedule: %w", err)
	}
	settings, err := mapper.ToMap(spec.PGSettings)
	if err != nil {
		return oapi.UpdateDbaasServicePgJSONRequestBody{}, fmt.Errorf("invalid pgsettings: %w", err)
	}

	return oapi.UpdateDbaasServicePgJSONRequestBody{
		Plan:           &spec.Size.Plan,
		BackupSchedule: &backupSchedule,
		Variant:        &variantAiven,
		// Version: pointer.String(spec.Version) -> Version cannot be changed,
		TerminationProtection: &spec.TerminationProtection,
		Maintenance: &struct {
			Dow  oapi.UpdateDbaasServicePgJSONBodyMaintenanceDow `json:"dow"`
			Time string                                          `json:"time"`
		}{
			Dow:  oapi.UpdateDbaasServicePgJSONBodyMaintenanceDow(spec.Maintenance.DayOfWeek),
			Time: spec.Maintenance.TimeOfDay.String(),
		},
		IpFilter:   mapper.ToSlicePtr(spec.IPFilter),
		PgSettings: &settings,
	}, nil
}
