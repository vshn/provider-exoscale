package mapper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"k8s.io/utils/pointer"
)

func TestToBackupSpec(t *testing.T) {
	tests := map[string]struct {
		givenSchedule *BackupSchedule
		expectedSpec  exoscalev1.BackupSpec
	}{
		"NilSchedule": {
			givenSchedule: nil,
			expectedSpec:  exoscalev1.BackupSpec{},
		},
		"ScheduleWithZero": {
			givenSchedule: &BackupSchedule{BackupHour: pointer.Int64(0), BackupMinute: pointer.Int64(0)},
			expectedSpec:  exoscalev1.BackupSpec{TimeOfDay: exoscalev1.TimeOfDay("00:00:00")},
		},
		"ScheduleWithoutNumbers": {
			givenSchedule: &BackupSchedule{},
			expectedSpec:  exoscalev1.BackupSpec{TimeOfDay: exoscalev1.TimeOfDay("00:00:00")},
		},
		"ScheduleWithNumbers": {
			givenSchedule: &BackupSchedule{BackupHour: pointer.Int64(12), BackupMinute: pointer.Int64(34)},
			expectedSpec:  exoscalev1.BackupSpec{TimeOfDay: exoscalev1.TimeOfDay("12:34:00")},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := ToBackupSpec(tc.givenSchedule)
			assert.Equal(t, tc.expectedSpec, result)
		})
	}
}

func TestToBackupSchedule(t *testing.T) {
	tests := map[string]struct {
		givenTime        exoscalev1.TimeOfDay
		expectedSchedule BackupSchedule
	}{
		"EmptyTime": {
			givenTime:        "0:00:00",
			expectedSchedule: BackupSchedule{BackupHour: pointer.Int64(0), BackupMinute: pointer.Int64(0)},
		},
		"TimeGiven": {
			givenTime:        "12:34:56",
			expectedSchedule: BackupSchedule{BackupHour: pointer.Int64(12), BackupMinute: pointer.Int64(34)},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := ToBackupSchedule(tc.givenTime)
			assert.NoError(t, err)
			assert.EqualValues(t, tc.expectedSchedule, result)
		})
	}
}
