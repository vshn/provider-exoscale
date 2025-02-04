package v1

import (
	"testing"

	exoscalesdk "github.com/exoscale/egoscale/v3"
	"github.com/stretchr/testify/assert"
)

func TestIamKeyObservation(t *testing.T) {
	tests := map[string]struct {
		givenObservation    IAMKeyObservation
		observedObservation IAMKeyObservation
		expected            bool
	}{
		"Same": {
			givenObservation:    IAMKeyObservation{RoleID: exoscalesdk.UUID("a58d2c65-6bd4-4cec-9789-f5622ba1f112")},
			observedObservation: IAMKeyObservation{RoleID: "a58d2c65-6bd4-4cec-9789-f5622ba1f112"},
			expected:            true,
		},
		"Different": {
			givenObservation:    IAMKeyObservation{RoleID: exoscalesdk.UUID("a58d2c65-6bd4-4cec-9789-f5622ba1f112")},
			observedObservation: IAMKeyObservation{RoleID: "7617092b-fa3c-4711-aad9-b0b71fdbebde"},
			expected:            false,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.givenObservation.Equals(tc.observedObservation))
		})
	}
}
