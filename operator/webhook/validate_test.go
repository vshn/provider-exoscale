package webhook

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestValidateRawExtension(t *testing.T) {
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
		"EmptyNestedObject": {givenSpec: `{"object":{}}`, expectedError: `validate: value of key "object" is not a supported type (only strings, boolean and numbers): map[]`},
		"NestedObject":      {givenSpec: `{"object":{"nested":"value"}}`, expectedError: `validate: value of key "object" is not a supported type (only strings, boolean and numbers): map[nested:value]`},
		"EmptySlice":        {givenSpec: `{"slice":[]}`, expectedError: `validate: value of key "slice" is not a supported type (only strings, boolean and numbers): []`},
		"NestedSlice":       {givenSpec: `{"slice":["value"]}`, expectedError: `validate: value of key "slice" is not a supported type (only strings, boolean and numbers): [value]`},
		"Slice":             {givenSpec: `[]`, expectedError: `mapper.ToMap({"[]" <nil>}): json: cannot unmarshal array into Go value of type map[string]interface {}`},
		"NestedNull":        {givenSpec: `{"null":null}`, expectedError: `validate: value of key "null" is not a supported type (only strings, boolean and numbers): <nil>`},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := ValidateRawExtension(runtime.RawExtension{
				Raw: []byte(tc.givenSpec),
			})
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateVersion(t *testing.T) {
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
			expectedError:      "field is immutable after creation: 14.0.0 (old), 15.2.0 (changed)",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := ValidateVersion(tc.oldObservedVersion, tc.oldSpecVersion, tc.newDesiredVersion)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
