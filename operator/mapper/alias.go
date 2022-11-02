package mapper

import (
	"fmt"

	"github.com/exoscale/egoscale/v2/oapi"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/pointer"
)

// BackupSchedule is a type alias for the embedded struct in opai.CreateDbaasServicePgJSONRequestBody.
type BackupSchedule = struct {
	BackupHour   *int64 `json:"backup-hour,omitempty"`
	BackupMinute *int64 `json:"backup-minute,omitempty"`
}

// MaintenanceScheduleCreateRequest is a type alias for the embedded struct in opai.CreateDbaasServicePgJSONRequestBody.
type MaintenanceScheduleCreateRequest = struct {
	// Day of week for installing updates
	Dow oapi.CreateDbaasServicePgJSONBodyMaintenanceDow `json:"dow"`

	// Time for installing updates, UTC
	Time string `json:"time"`
}

// MaintenanceScheduleUpdateRequest is a type alias for the embedded struct in opai.UpdateDbaasServicePgJSONRequestBody.
type MaintenanceScheduleUpdateRequest = struct {
	// Day of week for installing updates
	Dow oapi.UpdateDbaasServicePgJSONBodyMaintenanceDow `json:"dow"`

	// Time for installing updates, UTC
	Time string `json:"time"`
}

func ToBackupSchedule(day exoscalev1.TimeOfDay) (BackupSchedule, error) {
	backupHour, backupMin, _, err := day.Parse()
	return BackupSchedule{
		BackupHour:   pointer.Int64(backupHour),
		BackupMinute: pointer.Int64(backupMin),
	}, err
}

func toMaintenanceScheduleCreateRequest(spec exoscalev1.MaintenanceSpec) MaintenanceScheduleCreateRequest {
	return MaintenanceScheduleCreateRequest{
		Dow:  oapi.CreateDbaasServicePgJSONBodyMaintenanceDow(spec.DayOfWeek),
		Time: spec.TimeOfDay.String(),
	}
}

func toMaintenanceScheduleUpdateRequest(spec exoscalev1.MaintenanceSpec) MaintenanceScheduleUpdateRequest {
	return MaintenanceScheduleUpdateRequest{
		Dow:  oapi.UpdateDbaasServicePgJSONBodyMaintenanceDow(spec.DayOfWeek),
		Time: spec.TimeOfDay.String(),
	}
}

func toSlicePtr(arr []string) *[]string {
	return &arr
}

func ToSlice(arr *[]string) []string {
	if arr != nil {
		return *arr
	}
	return []string{}
}

func ToNodeStates(states *[]oapi.DbaasNodeState) []exoscalev1.NodeState {
	s := make([]exoscalev1.NodeState, len(*states))
	for i, state := range *states {
		s[i] = exoscalev1.NodeState{
			Name:  state.Name,
			Role:  *state.Role,
			State: state.State,
		}
	}
	return s
}

func ToNotifications(notifications *[]oapi.DbaasServiceNotification) ([]exoscalev1.Notification, error) {
	if notifications == nil {
		return nil, nil
	}

	s := make([]exoscalev1.Notification, len(*notifications))
	for i, notification := range *notifications {
		metadata, err := ToRawExtension(&notification.Metadata)
		if err != nil {
			return nil, fmt.Errorf("unable to convert metadata: %w", err)
		}
		s[i] = exoscalev1.Notification{
			Level:    notification.Level,
			Message:  notification.Message,
			Type:     notification.Type,
			Metadata: metadata,
		}
	}
	return s, nil
}

func ToBackupSpec(schedule *BackupSchedule) exoscalev1.BackupSpec {
	if schedule == nil {
		return exoscalev1.BackupSpec{}
	}
	hour, min := pointer.Int64Deref(schedule.BackupHour, 0), pointer.Int64Deref(schedule.BackupMinute, 0)
	return exoscalev1.BackupSpec{TimeOfDay: exoscalev1.TimeOfDay(fmt.Sprintf("%02d:%02d:00", hour, min))}
}

func ToMap(raw runtime.RawExtension) (map[string]interface{}, error) {
	m := make(map[string]interface{}, 0)
	if len(raw.Raw) == 0 {
		return m, nil
	}
	err := json.Unmarshal(raw.Raw, &m)
	return m, err
}

func ToRawExtension(m *map[string]interface{}) (runtime.RawExtension, error) {
	if m == nil {
		return runtime.RawExtension{}, nil
	}
	raw, err := json.Marshal(*m)
	return runtime.RawExtension{Raw: raw}, err
}

func ToDBaaSParameters(tp *bool, plan string, ipf *[]string) exoscalev1.DBaaSParameters {
	return exoscalev1.DBaaSParameters{
		TerminationProtection: pointer.BoolDeref(tp, false),
		Size: exoscalev1.SizeSpec{
			Plan: plan,
		},
		IPFilter: ToSlice(ipf),
	}
}

func ToMaintenance(m *oapi.DbaasServiceMaintenance) exoscalev1.MaintenanceSpec {
	return exoscalev1.MaintenanceSpec{
		DayOfWeek: m.Dow,
		TimeOfDay: exoscalev1.TimeOfDay(m.Time),
	}
}
