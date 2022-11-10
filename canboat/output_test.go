package canboat

import (
	"github.com/aldas/go-nmea-client"
	test_test "github.com/aldas/go-nmea-client/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMarshalRawMessage(t *testing.T) {
	now := test_test.UTCTime(1665488842) // Tue Oct 11 2022 11:47:22 GMT+0000

	var testCases = []struct {
		name        string
		when        nmea.RawMessage
		expect      string
		expectError string
	}{
		{
			name: "ok",
			when: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    6,
					PGN:         60928,
					Destination: 255,
					Source:      16,
				},
				Data: []byte{
					0x99, 0xad, 0x22, 0x22, 0x00, 0xa0, 0x64, 0xc0,
				},
			},
			expect: "2022-10-11T11:47:22Z,6,60928,16,255,8,99,ad,22,22,00,a0,64,c0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := MarshalRawMessage(tc.when)

			assert.Equal(t, tc.expect, string(result))
			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
