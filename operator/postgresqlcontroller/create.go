package postgresqlcontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	exoscalesdk "github.com/exoscale/egoscale/v3"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/mapper"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Create implements managed.ExternalClient.
func (p *pipeline) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("Creating resource")

	pgInstance := mg.(*exoscalev1.PostgreSQL)

	spec := pgInstance.Spec.ForProvider

	body, err := fromSpecToCreateBody(spec)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot map spec to API request")
	}
	resp, err := p.exo.CreateDBAASServicePG(ctx, pgInstance.Name, body)
	if err != nil {
		if strings.Contains(err.Error(), "Service name is already taken") {
			// According to the ExternalClient Interface, create needs to be idempotent.
			// However the exoscale client doesn't return very helpful errors, so we need to make this brittle matching to find if we get an already exits error
			return managed.ExternalCreation{}, nil
		}
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create instance")
	}
	log.V(1).Info("Response", "message", resp.Message)
	return managed.ExternalCreation{}, nil
}

// fromSpecToCreateBody places the given spec into the request body.
func fromSpecToCreateBody(spec exoscalev1.PostgreSQLParameters) (exoscalesdk.CreateDBAASServicePGRequest, error) {
	/**
	NOTE: If you change anything below, also update fromSpecToCreateBody().
	Unfortunately the generated openapi-types in exoscale are unusable for reusing same properties.
	*/
	backupSchedule, err := mapper.ToBackupSchedule(spec.Backup.TimeOfDay)
	if err != nil {
		return exoscalesdk.CreateDBAASServicePGRequest{}, fmt.Errorf("invalid backup schedule: %w", err)
	}
	settings := exoscalesdk.JSONSchemaPG{}
	if len(spec.PGSettings.Raw) != 0 {
		err = json.Unmarshal(spec.PGSettings.Raw, &settings)
		if err != nil {
			return exoscalesdk.CreateDBAASServicePGRequest{}, fmt.Errorf("invalid pgsettings: %w", err)
		}
	}

	return exoscalesdk.CreateDBAASServicePGRequest{
		Plan: spec.Size.Plan,
		BackupSchedule: &exoscalesdk.CreateDBAASServicePGRequestBackupSchedule{
			BackupHour:   backupSchedule.BackupHour,
			BackupMinute: backupSchedule.BackupMinute,
		},
		Variant:               variantAiven,
		Version:               exoscalesdk.DBAASPGTargetVersions(spec.Version),
		TerminationProtection: &spec.TerminationProtection,
		Maintenance: &exoscalesdk.CreateDBAASServicePGRequestMaintenance{
			Dow:  exoscalesdk.CreateDBAASServicePGRequestMaintenanceDow(spec.Maintenance.DayOfWeek),
			Time: spec.Maintenance.TimeOfDay.String(),
		},
		IPFilter:   spec.IPFilter,
		PGSettings: settings,
	}, nil
}
