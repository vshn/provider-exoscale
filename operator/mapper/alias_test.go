package mapper

import (
	"testing"

	exoscalesdk "github.com/exoscale/egoscale/v3"
	"github.com/stretchr/testify/assert"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
)

func TestToBackupSpec(t *testing.T) {
	tests := map[string]struct {
		givenSchedule *exoscalesdk.DBAASServiceMysqlBackupSchedule
		expectedSpec  exoscalev1.BackupSpec
	}{
		"NilSchedule": {
			givenSchedule: nil,
			expectedSpec:  exoscalev1.BackupSpec{},
		},
		"ScheduleWithZero": {
			givenSchedule: &exoscalesdk.DBAASServiceMysqlBackupSchedule{BackupHour: 0, BackupMinute: 0},
			expectedSpec:  exoscalev1.BackupSpec{TimeOfDay: exoscalev1.TimeOfDay("00:00:00")},
		},
		"ScheduleWithoutNumbers": {
			givenSchedule: &exoscalesdk.DBAASServiceMysqlBackupSchedule{},
			expectedSpec:  exoscalev1.BackupSpec{TimeOfDay: exoscalev1.TimeOfDay("00:00:00")},
		},
		"ScheduleWithNumbers": {
			givenSchedule: &exoscalesdk.DBAASServiceMysqlBackupSchedule{BackupHour: 12, BackupMinute: 34},
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
			expectedSchedule: BackupSchedule{BackupHour: 0, BackupMinute: 0},
		},
		"TimeGiven": {
			givenTime:        "12:34:56",
			expectedSchedule: BackupSchedule{BackupHour: 12, BackupMinute: 34},
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

func TestToNodeState(t *testing.T) {
	roleMaster := exoscalesdk.DBAASNodeStateRoleMaster
	roleReplica := exoscalesdk.DBAASNodeStateRoleReadReplica

	tests := map[string]struct {
		given  *[]exoscalesdk.DBAASNodeState
		expect []exoscalev1.NodeState
	}{
		"Normal": {
			given: &[]exoscalesdk.DBAASNodeState{
				{
					Name:  "foo",
					Role:  roleMaster,
					State: "running",
				},
				{
					Name:  "bar",
					Role:  roleReplica,
					State: "running",
				},
			},
			expect: []exoscalev1.NodeState{
				{
					Name:  "foo",
					Role:  roleMaster,
					State: "running",
				},
				{
					Name:  "bar",
					Role:  roleReplica,
					State: "running",
				},
			},
		},
		"Nil": {},
		"NilRole": {
			given: &[]exoscalesdk.DBAASNodeState{
				{
					Name:  "foo",
					State: "running",
				},
				{
					Name:  "bar",
					State: "leaving",
				},
			},
			expect: []exoscalev1.NodeState{
				{
					Name:  "foo",
					State: "running",
				},
				{
					Name:  "bar",
					State: "leaving",
				},
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				res := ToNodeStates(tc.given)
				assert.EqualValues(t, tc.expect, res)
			})
		})
	}
}
