package mapper

import (
	"strings"

	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/minio/minio-go/v7/pkg/set"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"k8s.io/utils/pointer"
)

// PostgreSQLMapper is a mapper to convert exoscale's SDK resources to Kubernetes API specs.
type PostgreSQLMapper struct{}

// FromSpec places the given spec into the request body.
func (PostgreSQLMapper) FromSpec(spec exoscalev1.PostgreSQLParameters, into *oapi.CreateDbaasServicePgJSONBody) error {
	backupSchedule, backupErr := toBackupSchedule(spec.Backup.TimeOfDay)
	maintenanceSchedule := toMaintenanceSchedule(spec.Maintenance)
	if err := firstOf(backupErr); err != nil {
		return err
	}

	into.Plan = spec.Size.Plan
	into.BackupSchedule = &backupSchedule
	into.Variant = &variantAiven
	into.Version = pointer.String(spec.Version)
	into.TerminationProtection = pointer.Bool(spec.TerminationProtection)
	into.Maintenance = &maintenanceSchedule
	into.IpFilter = toSlicePtr(spec.IPFilter)
	return nil
}

// ToStatus fills the status fields from the given response body.
func (PostgreSQLMapper) ToStatus(pg *oapi.DbaasServicePg, into *exoscalev1.PostgreSQLObservation) error {
	into.NoteStates = toNodeStates(pg.NodeStates)
	into.Version = pointer.StringDeref(pg.Version, "")
	into.Maintenance = exoscalev1.MaintenanceSpec{
		DayOfWeek: pg.Maintenance.Dow,
		TimeOfDay: exoscalev1.TimeOfDay(pg.Maintenance.Time),
	}
	into.IPFilter = *pg.IpFilter
	into.Backup = toBackupSpec(pg.BackupSchedule)
	return nil
}

// IsResourceUpToDate returns true if the observed response body matches the desired spec.
func (PostgreSQLMapper) IsResourceUpToDate(pgInstance *exoscalev1.PostgreSQL, pgExo *oapi.DbaasServicePg) bool {
	if pgExo == nil {
		return false
	}
	for _, fn := range []func(*exoscalev1.PostgreSQL, *oapi.DbaasServicePg) bool{
		IsSameIPFilter,
		HasSameMajorVersion,
		HasSameMaintenanceSchedule,
		HasSameBackupSchedule,
		HasSameSizing,
		HasSameTerminationProtection,
	} {
		same := fn(pgInstance, pgExo)
		if !same {
			return false
		}
	}
	return true
}

// IsSameIPFilter returns true if both slices have the same unique elements in any order.
// Returns true if both slices are empty or in case duplicates are found.
func IsSameIPFilter(pgInstance *exoscalev1.PostgreSQL, pgExo *oapi.DbaasServicePg) bool {
	f := pgInstance.Spec.ForProvider.IPFilter
	if pgExo.IpFilter == nil {
		return len(f) == 0
	}
	set1 := set.CreateStringSet(f...)
	set2 := set.CreateStringSet(*pgExo.IpFilter...)
	return set1.Equals(set2)
}

// HasSameMaintenanceSchedule returns true if both types describe the same maintenance schedule.
func HasSameMaintenanceSchedule(pgInstance *exoscalev1.PostgreSQL, pgExo *oapi.DbaasServicePg) bool {
	if pgInstance.Spec.ForProvider.Maintenance.DayOfWeek != pgExo.Maintenance.Dow {
		return false
	}
	if pgInstance.Spec.ForProvider.Maintenance.TimeOfDay.String() != pgExo.Maintenance.Time {
		return false
	}
	return true
}

// HasSameBackupSchedule returns true if both types describe the same backup schedule.
func HasSameBackupSchedule(pgInstance *exoscalev1.PostgreSQL, pgExo *oapi.DbaasServicePg) bool {
	if hour, min, _, err := pgInstance.Spec.ForProvider.Backup.TimeOfDay.Parse(); err != nil || hour != pointer.Int64Deref(pgExo.BackupSchedule.BackupHour, 0) || min != pointer.Int64Deref(pgExo.BackupSchedule.BackupMinute, 0) {
		return false
	}
	return true
}

// HasSameMajorVersion returns true if the observed version has the desired major version.
func HasSameMajorVersion(pgInstance *exoscalev1.PostgreSQL, pgExo *oapi.DbaasServicePg) bool {
	// TODO: strings.HasPrefix is a very cheap check if the major version are the same. Maybe add a SemVer library later on
	return strings.HasPrefix(pointer.StringDeref(pgExo.Version, ""), pgInstance.Spec.ForProvider.Version)
}

// HasSameSizing returns true if the observed sizing parameters match the desired spec.
func HasSameSizing(pgInstance *exoscalev1.PostgreSQL, pgExo *oapi.DbaasServicePg) bool {
	return strings.EqualFold(pgInstance.Spec.ForProvider.Size.Plan, pgExo.Plan)
}

// HasSameTerminationProtection returns true if the termination protection flags are the same.
func HasSameTerminationProtection(pgInstance *exoscalev1.PostgreSQL, pgExo *oapi.DbaasServicePg) bool {
	return pgInstance.Spec.ForProvider.TerminationProtection == pointer.BoolDeref(pgExo.TerminationProtection, false)
}

var variantAiven = oapi.EnumPgVariantAiven
