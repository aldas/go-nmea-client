package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseCSVFieldsRaw(t *testing.T) {
	var testCases = []struct {
		name        string
		given       string
		expect      []csvPGNFields
		expectError string
	}{
		{
			name:  "ok",
			given: "129025:latitude,longitude;65280:manufacturerCode,industryCode",
			expect: []csvPGNFields{
				{
					PGN:      129025,
					fileName: "129025_4fab33037f3639c5414b9f62a96a4263.csv",
					fields:   []string{"latitude", "longitude"},
				},
				{
					PGN:      65280,
					fileName: "65280_9d3436508a611ed40cb8b58aa5df44a5.csv",
					fields:   []string{"manufacturerCode", "industryCode"},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseCSVFieldsRaw(tc.given)

			assert.Equal(t, tc.expect, result)
			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
