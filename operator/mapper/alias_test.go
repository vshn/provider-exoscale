package mapper

import (
	"testing"

	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/stretchr/testify/assert"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"k8s.io/utils/ptr"
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
			givenSchedule: &BackupSchedule{BackupHour: ptr.To[int64](0), BackupMinute: ptr.To[int64](0)},
			expectedSpec:  exoscalev1.BackupSpec{TimeOfDay: exoscalev1.TimeOfDay("00:00:00")},
		},
		"ScheduleWithoutNumbers": {
			givenSchedule: &BackupSchedule{},
			expectedSpec:  exoscalev1.BackupSpec{TimeOfDay: exoscalev1.TimeOfDay("00:00:00")},
		},
		"ScheduleWithNumbers": {
			givenSchedule: &BackupSchedule{BackupHour: ptr.To[int64](12), BackupMinute: ptr.To[int64](34)},
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
			expectedSchedule: BackupSchedule{BackupHour: ptr.To[int64](0), BackupMinute: ptr.To[int64](0)},
		},
		"TimeGiven": {
			givenTime:        "12:34:56",
			expectedSchedule: BackupSchedule{BackupHour: ptr.To[int64](12), BackupMinute: ptr.To[int64](34)},
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
	roleMaster := oapi.DbaasNodeStateRoleMaster
	roleReplica := oapi.DbaasNodeStateRoleReadReplica

	tests := map[string]struct {
		given  *[]oapi.DbaasNodeState
		expect []exoscalev1.NodeState
	}{
		"Normal": {
			given: &[]oapi.DbaasNodeState{
				{
					Name:  "foo",
					Role:  &roleMaster,
					State: "running",
				},
				{
					Name:  "bar",
					Role:  &roleReplica,
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
			given: &[]oapi.DbaasNodeState{
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
