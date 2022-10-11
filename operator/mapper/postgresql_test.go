package mapper

import (
	"testing"

	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/stretchr/testify/assert"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"k8s.io/utils/pointer"
)

func TestIsSameIPFilter(t *testing.T) {
	tests := map[string]struct {
		given    exoscalev1.IPFilter
		arg      []string
		expected bool
	}{
		"EmptyFilter_EmptyArg": {
			given:    []string{},
			arg:      []string{},
			expected: true,
		},
		"EmptyFilter_GivenArg": {
			given:    []string{},
			arg:      []string{"arg1"},
			expected: false,
		},
		"GivenFilter_EmptyArg": {
			given:    []string{"filter1"},
			arg:      []string{},
			expected: false,
		},
		"SingleValue_Same": {
			given:    []string{"1"},
			arg:      []string{"1"},
			expected: true,
		},
		"SingleValue_Different": {
			given:    []string{"1"},
			arg:      []string{"2"},
			expected: false,
		},
		"MultipleValues_Unordered": {
			given:    []string{"1", "2"},
			arg:      []string{"2", "1"},
			expected: true,
		},
		"MultipleValues_Difference": {
			given:    []string{"1", "2"},
			arg:      []string{"3", "1"},
			expected: false,
		},
		"MultipleValues_Duplicates": {
			given:    []string{"1", "2"},
			arg:      []string{"2", "1", "1"},
			expected: true,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			instance := &exoscalev1.PostgreSQL{}
			instance.Spec.ForProvider.IPFilter = tc.given
			exo := &oapi.DbaasServicePg{IpFilter: &tc.arg}

			result := IsSameIPFilter(instance, exo)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHasSameBackupSchedule(t *testing.T) {
	tests := map[string]struct {
		givenSpec    exoscalev1.BackupSpec
		observedSpec BackupSchedule
		expected     bool
	}{
		"Same": {
			givenSpec:    exoscalev1.BackupSpec{TimeOfDay: "12:00:00"},
			observedSpec: BackupSchedule{BackupHour: pointer.Int64(12), BackupMinute: pointer.Int64(0)},
			expected:     true,
		},
		"Different": {
			givenSpec:    exoscalev1.BackupSpec{TimeOfDay: "13:00:00"},
			observedSpec: BackupSchedule{BackupHour: pointer.Int64(12), BackupMinute: pointer.Int64(0)},
			expected:     false,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			instance := &exoscalev1.PostgreSQL{}
			instance.Spec.ForProvider.Backup = tc.givenSpec
			exo := &oapi.DbaasServicePg{BackupSchedule: &tc.observedSpec}

			result := HasSameBackupSchedule(instance, exo)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHasSameMaintenanceSchedule(t *testing.T) {
	tests := map[string]struct {
		givenSpec    exoscalev1.MaintenanceSpec
		observedSpec oapi.DbaasServiceMaintenance
		expected     bool
	}{
		"Same": {
			givenSpec:    exoscalev1.MaintenanceSpec{DayOfWeek: oapi.DbaasServiceMaintenanceDowFriday, TimeOfDay: "12:34:56"},
			observedSpec: oapi.DbaasServiceMaintenance{Dow: oapi.DbaasServiceMaintenanceDowFriday, Time: "12:34:56"},
			expected:     true,
		},
		"DifferentTime": {
			givenSpec:    exoscalev1.MaintenanceSpec{DayOfWeek: oapi.DbaasServiceMaintenanceDowFriday, TimeOfDay: "12:00:23"},
			observedSpec: oapi.DbaasServiceMaintenance{Dow: oapi.DbaasServiceMaintenanceDowFriday, Time: "12:34:56"},
			expected:     false,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			instance := &exoscalev1.PostgreSQL{}
			instance.Spec.ForProvider.Maintenance = tc.givenSpec
			exo := &oapi.DbaasServicePg{Maintenance: &tc.observedSpec}

			result := HasSameMaintenanceSchedule(instance, exo)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHasSameMajorVersion(t *testing.T) {
	tests := map[string]struct {
		givenSpec    string
		observedSpec string
		expected     bool
	}{
		"MajorVersionMatch":     {givenSpec: "14", observedSpec: "14.5", expected: true},
		"DifferentMajorVersion": {givenSpec: "13", observedSpec: "14.5", expected: false},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			instance := &exoscalev1.PostgreSQL{}
			instance.Spec.ForProvider.Version = tc.givenSpec
			exo := &oapi.DbaasServicePg{Version: &tc.observedSpec}

			result := HasSameMajorVersion(instance, exo)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHasSameSizing(t *testing.T) {
	tests := map[string]struct {
		givenSpec    exoscalev1.SizeSpec
		observedPlan string
		expected     bool
	}{
		"PlanMatches":           {givenSpec: exoscalev1.SizeSpec{Plan: "hobbyist-2"}, observedPlan: "hobbyist-2", expected: true},
		"PlanMatchesIgnoreCase": {givenSpec: exoscalev1.SizeSpec{Plan: "HOBBYIST-2"}, observedPlan: "hobbyist-2", expected: true},
		"PlanNoMatch":           {givenSpec: exoscalev1.SizeSpec{Plan: "🐔"}, observedPlan: "🦃", expected: false},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			instance := &exoscalev1.PostgreSQL{}
			instance.Spec.ForProvider.Size = tc.givenSpec
			exo := &oapi.DbaasServicePg{Plan: tc.observedPlan}

			result := HasSameSizing(instance, exo)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestPostgreSQLMapper_ToConnectionDetails(t *testing.T) {
	tests := map[string]struct {
		givenUri         string
		expectedUser     string
		expectedPassword string
		expectedUri      string
		expectedHost     string
		expectedPort     string
		expectedDatabase string
	}{
		"FullURL": {
			givenUri:         "postgres://avnadmin:SUPERSECRET@instance-name-UUID.aivencloud.com:21699/defaultdb?sslmode=require",
			expectedUser:     "avnadmin",
			expectedPassword: "SUPERSECRET",
			expectedUri:      "postgres://avnadmin:SUPERSECRET@instance-name-UUID.aivencloud.com:21699/defaultdb?sslmode=require",
			expectedHost:     "instance-name-UUID.aivencloud.com",
			expectedPort:     "21699",
			expectedDatabase: "defaultdb",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			exo := oapi.DbaasServicePg{Uri: &tc.givenUri}
			secrets, err := PostgreSQLMapper{}.ToConnectionDetails(exo, "somebase64string")
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedUser, string(secrets["POSTGRESQL_USER"]), "username")
			assert.Equal(t, tc.expectedPassword, string(secrets["POSTGRESQL_PASSWORD"]), "password")
			assert.Equal(t, tc.expectedUri, string(secrets["POSTGRESQL_URL"]), "full url")
			assert.Equal(t, tc.expectedHost, string(secrets["POSTGRESQL_HOST"]), "host name")
			assert.Equal(t, tc.expectedPort, string(secrets["POSTGRESQL_PORT"]), "port number")
			assert.Equal(t, tc.expectedDatabase, string(secrets["POSTGRESQL_DB"]), "database")
			assert.Equal(t, "somebase64string", string(secrets["ca.crt"]), "ca certificate")
		})
	}
}
