package canboat

import (
	"github.com/aldas/go-nmea-client"
	test_test "github.com/aldas/go-nmea-client/test"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
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

func TestUnmarshalString(t *testing.T) {
	var testCases = []struct {
		name        string
		when        string
		expect      nmea.RawMessage
		expectError string
	}{
		{
			name: "ok",
			when: "2022-10-11T11:47:22Z,6,60928,16,255,8,99,ad,22,22,00,a0,64,c0",
			expect: nmea.RawMessage{
				Time: test_test.UTCTime(1665488842), // Tue Oct 11 2022 11:47:22 GMT+0000
				Header: nmea.CanBusHeader{
					Priority:    6,
					PGN:         60928,
					Destination: 255,
					Source:      16,
				},
				Data: []byte{0x99, 0xad, 0x22, 0x22, 0x00, 0xa0, 0x64, 0xc0},
			},
		},
		{
			name: "ok, milliseconds",
			when: "2021-07-29T10:18:31.758Z,6,126208,36,0,7,02,82,ff,00,10,02,00",
			expect: nmea.RawMessage{
				Time: time.Unix(0, 1627553911758000000).In(time.UTC),
				Header: nmea.CanBusHeader{
					Priority:    6,
					PGN:         126208,
					Destination: 0,
					Source:      36,
				},
				Data: []byte{0x02, 0x82, 0xff, 0x00, 0x10, 0x02, 0x00},
			},
		},
		{
			name: "ok, nano seconds",
			when: "2023-02-07T11:55:11.002803898+02:00,2,127245,13,255,8,ff,07,ff,7f,00,00,ff,ff",
			expect: nmea.RawMessage{
				Time: time.Unix(0, 1675763711002803898).In(time.UTC),
				Header: nmea.CanBusHeader{
					Priority:    2,
					PGN:         127245,
					Destination: 255,
					Source:      13,
				},
				Data: []byte{0xff, 0x07, 0xff, 0x7f, 0x0, 0x0, 0xff, 0xff},
			},
		},
		{
			name:        "nok, too few parts",
			when:        "2023-02-07T11:55:11.x,2,127245,13,255,8",
			expect:      nmea.RawMessage{},
			expectError: "canboat input has fewer components than expected",
		},
		{
			name:        "nok, too few parts",
			when:        "2023-02-07T11:55:11.x,2,127245,13,255,8",
			expect:      nmea.RawMessage{},
			expectError: "canboat input has fewer components than expected",
		},
		{
			name:        "nok, invalid data len",
			when:        "2021-07-29T10:18:31.758Z,6,126208,36,0,7x,02,82,ff,00,10,02,00",
			expect:      nmea.RawMessage{},
			expectError: "canboat input invalid data length, err: strconv.ParseUint: parsing \"7x\": invalid syntax",
		},
		{
			name:        "nok, data len != actual count",
			when:        "2021-07-29T10:18:31.758Z,6,126208,36,0,6,02,82,ff,00,10,02,00",
			expect:      nmea.RawMessage{},
			expectError: "canboat input data length does not match bytes count",
		},
		{
			name:        "nok, invalid time",
			when:        "2023-02-07T11:55:11.x,2,127245,13,255,8,ff,07,ff,7f,00,00,ff,ff",
			expect:      nmea.RawMessage{},
			expectError: "canboat input invalid time format, err: parsing time \"2023-02-07T11:55:11.x\" as \"2006-01-02T15:04:05.999999999Z07:00\": cannot parse \".x\" as \"Z07:00\"",
		},
		{
			name:        "nok, invalid priority",
			when:        "2021-07-29T10:18:31.758Z,x,126208,36,0,7,02,82,ff,00,10,02,00",
			expect:      nmea.RawMessage{},
			expectError: "canboat input invalid priority, err: strconv.ParseUint: parsing \"x\": invalid syntax",
		},
		{
			name:        "nok, invalid PGN",
			when:        "2021-07-29T10:18:31.758Z,6,126208x,36,0,7,02,82,ff,00,10,02,00",
			expect:      nmea.RawMessage{},
			expectError: "canboat input invalid PGN, err: strconv.ParseUint: parsing \"126208x\": invalid syntax",
		},
		{
			name:        "nok, invalid source",
			when:        "2021-07-29T10:18:31.758Z,6,126208,36x,0,7,02,82,ff,00,10,02,00",
			expect:      nmea.RawMessage{},
			expectError: "canboat input invalid source, err: strconv.ParseUint: parsing \"36x\": invalid syntax",
		},
		{
			name:        "nok, invalid destination",
			when:        "2021-07-29T10:18:31.758Z,6,126208,36,0x,7,02,82,ff,00,10,02,00",
			expect:      nmea.RawMessage{},
			expectError: "canboat input invalid destination, err: strconv.ParseUint: parsing \"0x\": invalid syntax",
		},
		{
			name:        "nok, invalid hex bytes",
			when:        "2021-07-29T10:18:31.758Z,6,126208,36,0,7,02,82,ff,00,1x,02,00",
			expect:      nmea.RawMessage{},
			expectError: "canboat input failure to convert hex into bytes, err: encoding/hex: invalid byte: U+0078 'x'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := UnmarshalString(tc.when)

			assert.Equal(t, tc.expect, result)
			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
