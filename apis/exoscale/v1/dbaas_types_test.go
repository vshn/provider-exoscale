package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTimeOfDay_Parse(t *testing.T) {
	tests := []struct {
		givenInput     string
		expectedMinute int64
		expectedHour   int64
		expectedSecond int64
		expectedError  string
	}{
		{givenInput: "00:00:00", expectedHour: 0, expectedMinute: 0, expectedSecond: 0},
		{givenInput: "23:59:00", expectedHour: 23, expectedMinute: 59, expectedSecond: 0},
		{givenInput: "01:01:01", expectedHour: 1, expectedMinute: 1, expectedSecond: 1},
		{givenInput: "1:59:59", expectedHour: 1, expectedMinute: 59, expectedSecond: 59},
		{givenInput: "19:59:01", expectedHour: 19, expectedMinute: 59, expectedSecond: 1},
		{givenInput: "4:59:01", expectedHour: 4, expectedMinute: 59, expectedSecond: 1},
		{givenInput: "04:59:01", expectedHour: 4, expectedMinute: 59, expectedSecond: 1},
		{givenInput: "9:01:01", expectedHour: 9, expectedMinute: 1, expectedSecond: 1},
		{givenInput: "", expectedError: "time cannot be empty"},
		{givenInput: "invalid", expectedError: "invalid format for time of day (hh:mm:ss): invalid"},
		{givenInput: "🕗", expectedError: "invalid format for time of day (hh:mm:ss): 🕗"},
		{givenInput: "-1:1", expectedError: "invalid format for time of day (hh:mm:ss): -1:1"},
		{givenInput: "24:01:30", expectedError: "invalid format for time of day (hh:mm:ss): 24:01:30"},
		{givenInput: "23:60:00", expectedError: "invalid format for time of day (hh:mm:ss): 23:60:00"},
		{givenInput: "foo:01:02", expectedError: "invalid format for time of day (hh:mm:ss): foo:01:02"},
		{givenInput: "01:bar:02", expectedError: "invalid format for time of day (hh:mm:ss): 01:bar:02"},
	}
	for _, tc := range tests {
		t.Run(tc.givenInput, func(t *testing.T) {
			hour, minute, second, err := TimeOfDay(tc.givenInput).Parse()
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
				assert.Equal(t, int64(-1), hour)
				assert.Equal(t, int64(-1), minute)
				assert.Equal(t, int64(-1), second)

			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedHour, hour)
				assert.Equal(t, tc.expectedMinute, minute)
				assert.Equal(t, tc.expectedSecond, second)
			}
		})
	}
}
