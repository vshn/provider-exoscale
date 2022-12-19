package mapper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestIsSameStringSet(t *testing.T) {
	tests := map[string]struct {
		given    []string
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
			result := IsSameStringSet(tc.given, &tc.arg)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCompareMajorVersion(t *testing.T) {
	tests := map[string]struct {
		versionA      string
		versionB      string
		expectedBool  bool
		expectedError string
	}{
		"same major version but not minor": {
			versionA:      "14",
			versionB:      "14.5",
			expectedBool:  true,
			expectedError: "",
		},
		"same major and minor version": {
			versionA:      "14.0.0",
			versionB:      "14.0.1",
			expectedBool:  true,
			expectedError: "",
		},
		"different major version": {
			versionA:      "14.0.0",
			versionB:      "15.0.0",
			expectedBool:  false,
			expectedError: "",
		},
		"same major version but not minor with release info": {
			versionA:      "1.2.3-beta.1",
			versionB:      "1.3.0-beta.1",
			expectedBool:  true,
			expectedError: "",
		},
		"different major version with release info": {
			versionA:      "1.2.3-alpha.1",
			versionB:      "2.0.0-alpha.1",
			expectedBool:  false,
			expectedError: "",
		},
		"same major version but not minor starting with v": {
			versionA:      "v1.0.0",
			versionB:      "v1.2.3",
			expectedBool:  true,
			expectedError: "",
		},
		"wrong version": {
			versionA:      "random",
			versionB:      "v1.2.3",
			expectedBool:  false,
			expectedError: "Malformed version: random",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			b, err := CompareMajorVersion(tc.versionA, tc.versionB)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, b, tc.expectedBool)
			}
		})
	}
}

func TestCompareSettings(t *testing.T) {
	tests := map[string]struct {
		givenSpec    string
		observedSpec map[string]interface{}
		expected     bool
	}{
		"BothEmpty":       {givenSpec: "", observedSpec: nil, expected: true},
		"Null":            {givenSpec: "null", observedSpec: nil, expected: true},
		"EmptyObserved":   {givenSpec: `{"key":"value"}`, observedSpec: map[string]interface{}{}, expected: false},
		"EmptySpec":       {givenSpec: ``, observedSpec: map[string]interface{}{"key": "value"}, expected: true},
		"EmptySpecObject": {givenSpec: `{}`, observedSpec: nil, expected: true},
		"SameString":      {givenSpec: `{"string":"value"}`, observedSpec: map[string]interface{}{"string": "value"}, expected: true},
		"SameNumber":      {givenSpec: `{"number":0.5}`, observedSpec: map[string]interface{}{"number": 0.5}, expected: true},
		"SameBoolean":     {givenSpec: `{"bool":true}`, observedSpec: map[string]interface{}{"bool": true}, expected: true},
		"NestedNull":      {givenSpec: `{"null":null}`, observedSpec: map[string]interface{}{"null": nil}, expected: true},
		"MultipleValues": {
			givenSpec:    `{"bool":true,"number":0.01, "string": ""}`,
			observedSpec: map[string]interface{}{"bool": true, "number": 0.01, "string": ""},
			expected:     true,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			given := runtime.RawExtension{Raw: []byte(tc.givenSpec)}
			exo, err := ToRawExtension(&tc.observedSpec)
			assert.NoError(t, err)

			result := CompareSettings(given, exo)
			assert.Equal(t, tc.expected, result)
		})
	}
}
