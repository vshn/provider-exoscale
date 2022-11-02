package mysqlcontroller

import (
	"context"
	"fmt"

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
	log.V(1).Info("updating resource")

	mySQLInstance := mg.(*exoscalev1.MySQL)

	spec := mySQLInstance.Spec.ForProvider
	ipFilter := []string(spec.IPFilter)
	settings, err := mapper.ToMap(spec.MySQLSettings)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("cannot map mySQLInstance settings: %w", err)
	}
	backupSchedule, err := mapper.ToBackupSchedule(spec.Backup.TimeOfDay)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("cannot parse mySQLInstance backup schedule: %w", err)
	}

	body := oapi.UpdateDbaasServiceMysqlJSONRequestBody{
		Maintenance: &struct {
			Dow  oapi.UpdateDbaasServiceMysqlJSONBodyMaintenanceDow `json:"dow"`
			Time string                                             `json:"time"`
		}{
			Dow:  oapi.UpdateDbaasServiceMysqlJSONBodyMaintenanceDow(spec.Maintenance.DayOfWeek),
			Time: spec.Maintenance.TimeOfDay.String(),
		},
		BackupSchedule: &struct {
			BackupHour   *int64 `json:"backup-hour,omitempty"`
			BackupMinute *int64 `json:"backup-minute,omitempty"`
		}{
			BackupHour:   backupSchedule.BackupHour,
			BackupMinute: backupSchedule.BackupMinute,
		},
		TerminationProtection: &spec.TerminationProtection,
		Plan:                  &spec.Size.Plan,
		IpFilter:              &ipFilter,
		MysqlSettings:         &settings,
	}
	resp, err := p.exo.UpdateDbaasServiceMysqlWithResponse(ctx, oapi.DbaasServiceName(mySQLInstance.GetInstanceName()), body)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("cannot update mySQLInstance: %w", err)
	}
	log.V(1).Info("response", "body", string(resp.Body))
	return managed.ExternalUpdate{}, nil
}
