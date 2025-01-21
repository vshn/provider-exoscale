package postgresqlcontroller

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalesdk "github.com/exoscale/egoscale/v3"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/mapper"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Update implements managed.ExternalClient.
func (p *pipeline) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("Updating resource")

	pgInstance := mg.(*exoscalev1.PostgreSQL)

	spec := pgInstance.Spec.ForProvider
	body, err := fromSpecToUpdateBody(spec)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot map spec to API request")
	}
	resp, err := p.exo.UpdateDBAASServicePG(ctx, pgInstance.Name, body)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update instance")
	}
	log.V(1).Info("Response", "message", resp.Message)
	return managed.ExternalUpdate{}, nil
}

// fromSpecToUpdateBody places the given spec into the request body.
func fromSpecToUpdateBody(spec exoscalev1.PostgreSQLParameters) (exoscalesdk.UpdateDBAASServicePGRequest, error) {
	/**
	NOTE: If you change anything below, also update fromSpecToUpdateBody().
	Unfortunately the generated openapi-types in exoscale are unusable for reusing same properties.
	*/
	backupSchedule, err := mapper.ToBackupSchedule(spec.Backup.TimeOfDay)
	if err != nil {
		return exoscalesdk.UpdateDBAASServicePGRequest{}, fmt.Errorf("invalid backup schedule: %w", err)
	}
	settings := exoscalesdk.JSONSchemaPG{}
	if len(spec.PGSettings.Raw) != 0 {
		err = json.Unmarshal(spec.PGSettings.Raw, &settings)
		if err != nil {
			return exoscalesdk.UpdateDBAASServicePGRequest{}, fmt.Errorf("invalid pgsettings: %w", err)
		}
	}

	return exoscalesdk.UpdateDBAASServicePGRequest{
		Plan: spec.Size.Plan,
		BackupSchedule: &exoscalesdk.UpdateDBAASServicePGRequestBackupSchedule{
			BackupHour:   backupSchedule.BackupHour,
			BackupMinute: backupSchedule.BackupMinute,
		},
		Variant: variantAiven,
		// Version: pointer.String(spec.Version) -> Version cannot be changed,
		TerminationProtection: &spec.TerminationProtection,
		Maintenance: &exoscalesdk.UpdateDBAASServicePGRequestMaintenance{
			Dow:  exoscalesdk.UpdateDBAASServicePGRequestMaintenanceDow(spec.Maintenance.DayOfWeek),
			Time: spec.Maintenance.TimeOfDay.String(),
		},
		IPFilter:   spec.IPFilter,
		PGSettings: settings,
	}, nil
}
