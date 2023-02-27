package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
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
			given: "129025:latitude,longitude;65280:_time_ms(100ms),manufacturerCode,industryCode",
			expect: []csvPGNFields{
				{
					PGN:      129025,
					fileName: "129025_4fab33037f3639c5414b9f62a96a4263.csv",
					names:    []string{"latitude", "longitude"},
					fields: []field{
						{name: "latitude"},
						{name: "longitude"},
					},
				},
				{
					PGN:      65280,
					fileName: "65280_43bc0dc3dcedc05f5d70bd34b04f3835.csv",
					names:    []string{"_time_ms", "manufacturerCode", "industryCode"},
					fields: []field{
						{name: "_time_ms", truncate: 100 * time.Millisecond},
						{name: "manufacturerCode"},
						{name: "industryCode"},
					},
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
