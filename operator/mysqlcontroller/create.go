package mysqlcontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalesdk "github.com/exoscale/egoscale/v3"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"

	"github.com/vshn/provider-exoscale/operator/mapper"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Create implements managed.ExternalClient.
func (p *pipeline) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("creating resource")

	mySQLInstance := mg.(*exoscalev1.MySQL)

	spec := mySQLInstance.Spec.ForProvider
	ipFilter := []string(spec.IPFilter)
	settings := exoscalesdk.JSONSchemaMysql{}
	if len(spec.MySQLSettings.Raw) != 0 {
		err := json.Unmarshal(spec.MySQLSettings.Raw, &settings)
		if err != nil {
			return managed.ExternalCreation{}, fmt.Errorf("cannot map mySQLInstance settings: %w", err)
		}
	}
	backupSchedule, err := mapper.ToBackupSchedule(spec.Backup.TimeOfDay)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("cannot parse mySQLInstance backup schedule: %w", err)
	}

	body := exoscalesdk.CreateDBAASServiceMysqlRequest{
		Maintenance: &exoscalesdk.CreateDBAASServiceMysqlRequestMaintenance{
			Dow:  exoscalesdk.CreateDBAASServiceMysqlRequestMaintenanceDow(spec.Maintenance.DayOfWeek),
			Time: spec.Maintenance.TimeOfDay.String(),
		},
		BackupSchedule: &exoscalesdk.CreateDBAASServiceMysqlRequestBackupSchedule{
			BackupHour:   backupSchedule.BackupHour,
			BackupMinute: backupSchedule.BackupMinute,
		},
		Version:               spec.Version,
		TerminationProtection: &spec.TerminationProtection,
		Plan:                  spec.Size.Plan,
		IPFilter:              ipFilter,
		MysqlSettings:         settings,
	}
	resp, err := p.exo.CreateDBAASServiceMysql(ctx, mySQLInstance.GetInstanceName(), body)
	if err != nil {
		if strings.Contains(err.Error(), "Service name is already taken") {
			// According to the ExternalClient Interface, create needs to be idempotent.
			// However the exoscale client doesn't return very helpful errors, so we need to make this brittle matching to find if we get an already exits error
			return managed.ExternalCreation{}, nil
		}
		return managed.ExternalCreation{}, fmt.Errorf("cannot create mySQLInstance: %w", err)
	}

	log.V(1).Info("response", "message", resp.Message)
	return managed.ExternalCreation{}, nil
}
