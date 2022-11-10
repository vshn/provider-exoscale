package mapper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_CompareMajorVersion(t *testing.T) {
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
