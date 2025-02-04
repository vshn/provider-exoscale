package mysqlcontroller

import (
	"context"
	"encoding/json"
	"fmt"

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
	log.V(1).Info("updating resource")

	mySQLInstance := mg.(*exoscalev1.MySQL)

	spec := mySQLInstance.Spec.ForProvider
	ipFilter := []string(spec.IPFilter)
	settings := exoscalesdk.JSONSchemaMysql{}
	if len(spec.MySQLSettings.Raw) != 0 {
		err := json.Unmarshal(spec.MySQLSettings.Raw, &settings)
		if err != nil {
			return managed.ExternalUpdate{}, fmt.Errorf("cannot map mySQLInstance settings: %w", err)
		}
	}
	backupSchedule, err := mapper.ToBackupSchedule(spec.Backup.TimeOfDay)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("cannot parse mySQLInstance backup schedule: %w", err)
	}

	body := exoscalesdk.UpdateDBAASServiceMysqlRequest{
		Maintenance: &exoscalesdk.UpdateDBAASServiceMysqlRequestMaintenance{
			Dow:  exoscalesdk.UpdateDBAASServiceMysqlRequestMaintenanceDow(spec.Maintenance.DayOfWeek),
			Time: spec.Maintenance.TimeOfDay.String(),
		},
		BackupSchedule: &exoscalesdk.UpdateDBAASServiceMysqlRequestBackupSchedule{
			BackupHour:   backupSchedule.BackupHour,
			BackupMinute: backupSchedule.BackupMinute,
		},
		TerminationProtection: &spec.TerminationProtection,
		Plan:                  spec.Size.Plan,
		IPFilter:              ipFilter,
		MysqlSettings:         settings,
	}
	resp, err := p.exo.UpdateDBAASServiceMysql(ctx, mySQLInstance.GetInstanceName(), body)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("cannot update mySQLInstance: %w", err)
	}
	log.V(1).Info("response", "message", string(resp.Message))
	return managed.ExternalUpdate{}, nil
}
