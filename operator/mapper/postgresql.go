package mapper

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/exoscale/egoscale/v2/oapi"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"k8s.io/utils/pointer"
)

// PostgreSQLMapper is a mapper to convert exoscale's SDK resources to Kubernetes API specs.
type PostgreSQLMapper struct{}

// FromSpecToCreateBody places the given spec into the request body.
func (PostgreSQLMapper) FromSpecToCreateBody(spec exoscalev1.PostgreSQLParameters, into *oapi.CreateDbaasServicePgJSONBody) error {
	/**
	NOTE: If you change anything below, also update FromSpecToUpdateBody().
	Unfortunately the generated openapi-types in exoscale are unusable for reusing same properties.
	*/
	backupSchedule, backupErr := toBackupSchedule(spec.Backup.TimeOfDay)
	maintenanceSchedule := toMaintenanceScheduleCreateRequest(spec.Maintenance)
	pgSettings, parseErr := ToMap(spec.PGSettings)
	if err := firstOf(backupErr, parseErr); err != nil {
		return err
	}

	into.Plan = spec.Size.Plan
	into.BackupSchedule = &backupSchedule
	into.Variant = &variantAiven
	into.Version = pointer.String(spec.Version)
	into.TerminationProtection = pointer.Bool(spec.TerminationProtection)
	into.Maintenance = &maintenanceSchedule
	into.IpFilter = toSlicePtr(spec.IPFilter)
	into.PgSettings = &pgSettings
	return nil
}

// FromSpecToUpdateBody places the given spec into the request body.
func (PostgreSQLMapper) FromSpecToUpdateBody(spec exoscalev1.PostgreSQLParameters, into *oapi.UpdateDbaasServicePgJSONRequestBody) error {
	/**
	NOTE: If you change anything below, also update FromSpecToCreateBody().
	Unfortunately the generated openapi-types in exoscale are unusable for reusing same properties.
	*/
	backupSchedule, backupErr := toBackupSchedule(spec.Backup.TimeOfDay)
	maintenanceSchedule := toMaintenanceScheduleUpdateRequest(spec.Maintenance)
	pgSettings, parseErr := ToMap(spec.PGSettings)
	if err := firstOf(backupErr, parseErr); err != nil {
		return err
	}

	into.Plan = pointer.String(spec.Size.Plan)
	into.BackupSchedule = &backupSchedule
	into.Variant = &variantAiven
	// into.Version = pointer.String(spec.Version) -> Version cannot be changed
	into.TerminationProtection = pointer.Bool(spec.TerminationProtection)
	into.Maintenance = &maintenanceSchedule
	into.IpFilter = toSlicePtr(spec.IPFilter)
	into.PgSettings = &pgSettings
	return nil
}

// ToStatus fills the status fields from the given response body.
func (PostgreSQLMapper) ToStatus(pg *oapi.DbaasServicePg, into *exoscalev1.PostgreSQLObservation) error {
	into.NodeStates = ToNodeStates(pg.NodeStates)
	into.Version = pointer.StringDeref(pg.Version, "")
	into.Maintenance = exoscalev1.MaintenanceSpec{
		DayOfWeek: pg.Maintenance.Dow,
		TimeOfDay: exoscalev1.TimeOfDay(pg.Maintenance.Time),
	}
	into.Size.Plan = pg.Plan
	into.IPFilter = *pg.IpFilter
	into.Backup = toBackupSpec(pg.BackupSchedule)
	parsed, err := ToRawExtension(pg.PgSettings)
	into.PGSettings = parsed
	return errors.Wrap(err, "cannot marshal json")
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
		HasSamePGSettings,
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
	return IsSameStringSet(pgInstance.Spec.ForProvider.IPFilter, pgExo.IpFilter)
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

// HasSamePGSettings returns true if the pg settings spec are the same as defined in the exoscale payload.
func HasSamePGSettings(pgInstance *exoscalev1.PostgreSQL, pgExo *oapi.DbaasServicePg) bool {
	spec, err := ToMap(pgInstance.Spec.ForProvider.PGSettings)
	if err != nil {
		// we have to assume they're not the same
		return false
	}
	if len(spec) == 0 && (pgExo.PgSettings == nil || len(*pgExo.PgSettings) == 0) {
		// both are effectively empty
		return true
	}
	return reflect.DeepEqual(spec, *pgExo.PgSettings)
}

var variantAiven = oapi.EnumPgVariantAiven

// ToConnectionDetails parses the connection details from the given observation.
func (PostgreSQLMapper) ToConnectionDetails(pgExo oapi.DbaasServicePg, ca string) (managed.ConnectionDetails, error) {
	raw := pointer.StringDeref(pgExo.Uri, "")
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("cannot parse connection URL: %w", err)
	}
	password, _ := parsed.User.Password()
	return map[string][]byte{
		"POSTGRESQL_USER":     []byte(parsed.User.Username()),
		"POSTGRESQL_PASSWORD": []byte(password),
		"POSTGRESQL_URL":      []byte(raw),
		"POSTGRESQL_DB":       []byte(strings.TrimPrefix(parsed.Path, "/")),
		"POSTGRESQL_HOST":     []byte(parsed.Hostname()),
		"POSTGRESQL_PORT":     []byte(parsed.Port()),
		"ca.crt":              []byte(ca),
	}, nil
}
