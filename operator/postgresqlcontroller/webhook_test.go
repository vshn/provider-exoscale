package postgresqlcontroller

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
)

func TestValidator_compareVersion(t *testing.T) {
	tests := map[string]struct {
		oldObservedVersion string
		oldSpecVersion     string
		newDesiredVersion  string
		expectedError      string
	}{
		"NoChange_SameVersion": {
			oldSpecVersion:     "14",
			newDesiredVersion:  "14",
			oldObservedVersion: "14.5",
			expectedError:      "",
		},
		"NoChange_DifferentObservedVersion": {
			oldSpecVersion:     "14",
			newDesiredVersion:  "14",
			oldObservedVersion: "15.1",
			expectedError:      "",
		},
		"Change_ToSameObservedVersion": {
			oldSpecVersion:     "14",
			newDesiredVersion:  "15.1",
			oldObservedVersion: "15.1",
			expectedError:      "",
		},
		"Change_ToOtherValue": {
			oldSpecVersion:     "14",
			newDesiredVersion:  "15.2",
			oldObservedVersion: "15.1",
			expectedError:      "field is immutable after creation: 14 (old), 15.2 (changed)",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			v := Validator{log: logr.Discard()}
			oldObj := &exoscalev1.PostgreSQL{}
			newObj := &exoscalev1.PostgreSQL{}
			oldObj.Status.AtProvider.Version = tc.oldObservedVersion
			oldObj.Spec.ForProvider.Version = tc.oldSpecVersion
			newObj.Spec.ForProvider.Version = tc.newDesiredVersion
			err := v.compareVersion(oldObj, newObj)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
