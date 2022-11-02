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

// Create implements managed.ExternalClient.
func (p *pipeline) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("creating resource")

	mySQLInstance := mg.(*exoscalev1.MySQL)

	spec := mySQLInstance.Spec.ForProvider
	ipFilter := []string(spec.IPFilter)
	settings, err := mapper.ToMap(spec.MySQLSettings)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("cannot map mySQLInstance settings: %w", err)
	}
	backupSchedule, err := mapper.ToBackupSchedule(spec.Backup.TimeOfDay)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("cannot parse mySQLInstance backup schedule: %w", err)
	}

	body := oapi.CreateDbaasServiceMysqlJSONRequestBody{
		Maintenance: &struct {
			Dow  oapi.CreateDbaasServiceMysqlJSONBodyMaintenanceDow `json:"dow"`
			Time string                                             `json:"time"`
		}{
			Dow:  oapi.CreateDbaasServiceMysqlJSONBodyMaintenanceDow(spec.Maintenance.DayOfWeek),
			Time: spec.Maintenance.TimeOfDay.String(),
		},
		BackupSchedule: &struct {
			BackupHour   *int64 `json:"backup-hour,omitempty"`
			BackupMinute *int64 `json:"backup-minute,omitempty"`
		}{
			BackupHour:   backupSchedule.BackupHour,
			BackupMinute: backupSchedule.BackupMinute,
		},
		Version:               &spec.Version,
		TerminationProtection: &spec.TerminationProtection,
		Plan:                  spec.Size.Plan,
		IpFilter:              &ipFilter,
		MysqlSettings:         &settings,
	}
	resp, err := p.exo.CreateDbaasServiceMysqlWithResponse(ctx, oapi.DbaasServiceName(mySQLInstance.GetInstanceName()), body)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("cannot create mySQLInstance: %w", err)
	}

	log.V(1).Info("response", "body", string(resp.Body))
	return managed.ExternalCreation{}, nil
}
