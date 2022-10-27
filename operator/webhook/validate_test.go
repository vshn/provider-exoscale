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
