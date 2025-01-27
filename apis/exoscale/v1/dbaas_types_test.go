package v1

import (
	"testing"

	exoscalesdk "github.com/exoscale/egoscale/v3"
	"github.com/stretchr/testify/assert"
)

func TestTimeOfDay_Parse(t *testing.T) {
	tests := []struct {
		givenInput     string
		expectedMinute int64
		expectedHour   int64
		expectedSecond int64
		expectedError  string
	}{
		{givenInput: "00:00:00", expectedHour: 0, expectedMinute: 0, expectedSecond: 0},
		{givenInput: "23:59:00", expectedHour: 23, expectedMinute: 59, expectedSecond: 0},
		{givenInput: "01:01:01", expectedHour: 1, expectedMinute: 1, expectedSecond: 1},
		{givenInput: "1:59:59", expectedHour: 1, expectedMinute: 59, expectedSecond: 59},
		{givenInput: "19:59:01", expectedHour: 19, expectedMinute: 59, expectedSecond: 1},
		{givenInput: "4:59:01", expectedHour: 4, expectedMinute: 59, expectedSecond: 1},
		{givenInput: "04:59:01", expectedHour: 4, expectedMinute: 59, expectedSecond: 1},
		{givenInput: "9:01:01", expectedHour: 9, expectedMinute: 1, expectedSecond: 1},
		{givenInput: "", expectedError: "time cannot be empty"},
		{givenInput: "invalid", expectedError: "invalid format for time of day (hh:mm:ss): invalid"},
		{givenInput: "🕗", expectedError: "invalid format for time of day (hh:mm:ss): 🕗"},
		{givenInput: "-1:1", expectedError: "invalid format for time of day (hh:mm:ss): -1:1"},
		{givenInput: "24:01:30", expectedError: "invalid format for time of day (hh:mm:ss): 24:01:30"},
		{givenInput: "23:60:00", expectedError: "invalid format for time of day (hh:mm:ss): 23:60:00"},
		{givenInput: "foo:01:02", expectedError: "invalid format for time of day (hh:mm:ss): foo:01:02"},
		{givenInput: "01:bar:02", expectedError: "invalid format for time of day (hh:mm:ss): 01:bar:02"},
	}
	for _, tc := range tests {
		t.Run(tc.givenInput, func(t *testing.T) {
			hour, minute, second, err := TimeOfDay(tc.givenInput).Parse()
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
				assert.Equal(t, int64(-1), hour)
				assert.Equal(t, int64(-1), minute)
				assert.Equal(t, int64(-1), second)

			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedHour, hour)
				assert.Equal(t, tc.expectedMinute, minute)
				assert.Equal(t, tc.expectedSecond, second)
			}
		})
	}
}

func TestMaintenanceSpec_Equals(t *testing.T) {
	tests := map[string]struct {
		ms    MaintenanceSpec
		other MaintenanceSpec
		want  bool
	}{
		"empty equals": {
			ms: MaintenanceSpec{
				DayOfWeek: "",
				TimeOfDay: "",
			},
			other: MaintenanceSpec{
				DayOfWeek: "",
				TimeOfDay: "",
			},
			want: true,
		},
		"same equals": {
			ms: MaintenanceSpec{
				DayOfWeek: exoscalesdk.DBAASServiceMaintenanceDowFriday,
				TimeOfDay: "12:00:00",
			},
			other: MaintenanceSpec{
				DayOfWeek: exoscalesdk.DBAASServiceMaintenanceDowFriday,
				TimeOfDay: "12:00:00",
			},
			want: true,
		},
		"day diff": {
			ms: MaintenanceSpec{
				DayOfWeek: exoscalesdk.DBAASServiceMaintenanceDowMonday,
				TimeOfDay: "12:00:00",
			},
			other: MaintenanceSpec{
				DayOfWeek: exoscalesdk.DBAASServiceMaintenanceDowFriday,
				TimeOfDay: "12:00:00",
			},
			want: false,
		},
		"time diff": {
			ms: MaintenanceSpec{
				DayOfWeek: exoscalesdk.DBAASServiceMaintenanceDowFriday,
				TimeOfDay: "12:00:01",
			},
			other: MaintenanceSpec{
				DayOfWeek: exoscalesdk.DBAASServiceMaintenanceDowFriday,
				TimeOfDay: "12:00:00",
			},
			want: false,
		},
		"date & time diff": {
			ms: MaintenanceSpec{
				DayOfWeek: exoscalesdk.DBAASServiceMaintenanceDowFriday,
				TimeOfDay: "12:00:01",
			},
			other: MaintenanceSpec{
				DayOfWeek: exoscalesdk.DBAASServiceMaintenanceDowFriday,
				TimeOfDay: "12:00:00",
			},
			want: false,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equalf(t, tt.want, tt.ms.Equals(tt.other), "Equals(%v)", tt.other)
		})
	}
}

func TestSizeSpec_Equals(t *testing.T) {
	tests := map[string]struct {
		s     SizeSpec
		other SizeSpec
		want  bool
	}{
		"empty equals": {
			s: SizeSpec{
				Plan: "",
			},
			other: SizeSpec{
				Plan: "",
			},
			want: true,
		},
		"empty plan differs": {
			s: SizeSpec{
				Plan: "",
			},
			other: SizeSpec{
				Plan: "b",
			},
			want: false,
		},
		"plan differs equals": {
			s: SizeSpec{
				Plan: "a",
			},
			other: SizeSpec{
				Plan: "b",
			},
			want: false,
		},
		"plan ignores case": {
			s: SizeSpec{
				Plan: "a",
			},
			other: SizeSpec{
				Plan: "A",
			},
			want: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equalf(t, tt.want, tt.s.Equals(tt.other), "Equals(%v)", tt.other)
		})
	}
}

func TestBackupSpec_Equals(t *testing.T) {
	tests := map[string]struct {
		givenSpec    BackupSpec
		observedSpec BackupSpec
		expected     bool
	}{
		"Same": {
			givenSpec:    BackupSpec{TimeOfDay: "12:00:00"},
			observedSpec: BackupSpec{TimeOfDay: "12:00:00"},
			expected:     true,
		},
		"Different": {
			givenSpec:    BackupSpec{TimeOfDay: "13:00:00"},
			observedSpec: BackupSpec{TimeOfDay: "12:00:00"},
			expected:     false,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.givenSpec.Equals(tc.observedSpec))
		})
	}
}
