package postgresqlcontroller

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
)

func TestValidator_validatePGSettings(t *testing.T) {
	tests := map[string]struct {
		givenSpec     string
		expectedError string
	}{
		"EmptySettings":     {givenSpec: "", expectedError: ""},
		"EmptyObject":       {givenSpec: "{}", expectedError: ""},
		"Null":              {givenSpec: `null`, expectedError: ""},
		"Float":             {givenSpec: `{"number":0.5}`, expectedError: ""},
		"Zero":              {givenSpec: `{"number":0}`, expectedError: ""},
		"Integer":           {givenSpec: `{"number":1}`, expectedError: ""},
		"NegativeNumber":    {givenSpec: `{"number":-1.5}`, expectedError: ""},
		"String":            {givenSpec: `{"string":"value"}`, expectedError: ""},
		"EmptyString":       {givenSpec: `{"string":""}`, expectedError: ""},
		"Boolean":           {givenSpec: `{"bool":true}`, expectedError: ""},
		"EmptyNestedObject": {givenSpec: `{"object":{}}`, expectedError: `value in key "object" in pgSettings is not a supported type (only strings, boolean and numbers): map[]`},
		"NestedObject":      {givenSpec: `{"object":{"nested":"value"}}`, expectedError: `value in key "object" in pgSettings is not a supported type (only strings, boolean and numbers): map[nested:value]`},
		"EmptySlice":        {givenSpec: `{"slice":[]}`, expectedError: `value in key "slice" in pgSettings is not a supported type (only strings, boolean and numbers): []`},
		"NestedSlice":       {givenSpec: `{"slice":["value"]}`, expectedError: `value in key "slice" in pgSettings is not a supported type (only strings, boolean and numbers): [value]`},
		"Slice":             {givenSpec: `[]`, expectedError: `pgSettings with value "[]" cannot be converted: json: cannot unmarshal array into Go value of type map[string]interface {}`},
		"NestedNull":        {givenSpec: `{"null":null}`, expectedError: `value in key "null" in pgSettings is not a supported type (only strings, boolean and numbers): <nil>`},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			v := Validator{log: logr.Discard()}
			params := exoscalev1.PostgreSQLParameters{}
			params.PGSettings.Raw = []byte(tc.givenSpec)
			err := v.validatePGSettings(params)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
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
