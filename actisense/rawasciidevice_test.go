package actisense

import (
	"github.com/aldas/go-nmea-client"
	test_test "github.com/aldas/go-nmea-client/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseRawAscii(t *testing.T) {
	now := test_test.UTCTime(1665488842) // Tue Oct 11 2022 11:47:22 GMT+0000
	var testCases = []struct {
		name        string
		when        []byte
		expect      nmea.RawFrame
		expectSkip  bool
		expectError string
	}{
		{
			name: "ok",
			when: []byte(`00:34:02.718 R 15FD0800 FF 00 01 CA 6F FF FF FF`),
			expect: nmea.RawFrame{
				Time: now,
				Header: nmea.CanBusHeader{
					PGN:         0x1FD08, // 1FD08 -> 130312 Temperature
					Source:      0,       // 0x0
					Destination: 255,     // 0xff - broadcast
					Priority:    5,       // 0x05
				},
				Length: 8,
				Data:   [8]byte{0xFF, 0x0, 0x01, 0xCA, 0x6F, 0xFF, 0xFF, 0xFF},
			},
		},
		{
			name: "ok, carriage return (0x13) at the end",
			when: []byte(`00:34:02.718 R 15FD0800 FF 00 01 CA 6F FF FF FF` + "\r"),
			expect: nmea.RawFrame{
				Time: now,
				Header: nmea.CanBusHeader{
					PGN:         0x1FD08, // 1FD08 -> 130312 Temperature
					Source:      0,       // 0x0
					Destination: 255,     // 0xff - broadcast
					Priority:    5,       // 0x05
				},
				Length: 8,
				Data:   [8]byte{0xFF, 0x0, 0x01, 0xCA, 0x6F, 0xFF, 0xFF, 0xFF},
			},
		},
		{
			name: "ok, line feed (0x10) at the end",
			when: []byte(`00:34:02.718 R 15FD0800 FF 00 01 CA 6F FF FF FF` + "\n"),
			expect: nmea.RawFrame{
				Time: now,
				Header: nmea.CanBusHeader{
					PGN:         0x1FD08, // 1FD08 -> 130312 Temperature
					Source:      0,       // 0x0
					Destination: 255,     // 0xff - broadcast
					Priority:    5,       // 0x05
				},
				Length: 8,
				Data:   [8]byte{0xFF, 0x0, 0x01, 0xCA, 0x6F, 0xFF, 0xFF, 0xFF},
			},
		},
		{
			name: "ok, fast-packet first frame",
			when: []byte(`00:34:02.802 R 1DFF0400 80 07 3F 9F 00 40 00 00`),
			expect: nmea.RawFrame{
				Time: now,
				Header: nmea.CanBusHeader{
					PGN:         0x1FF04, // 1FF04 -> 130820 Proprietary
					Source:      0,       // 0x0
					Destination: 255,     // 0xff
					Priority:    7,       // 0x07
				},
				Length: 8,
				Data:   [8]byte{0x80, 0x07, 0x3F, 0x9F, 0x00, 0x40, 0x00, 0x00},
			},
		},
		{
			name: "ok, 127251 Rate of Turn",
			when: []byte(`00:34:03.239 R 09F11323 3A 9C 63 01 00 FF FF FF`),
			expect: nmea.RawFrame{
				Time: now,
				Header: nmea.CanBusHeader{
					PGN:         0x1F113, // 1F113 -> 127251 Rate of Turn
					Source:      35,      // 0x23
					Destination: 255,     // 0xff
					Priority:    2,       // 0x02
				},
				Length: 8,
				Data:   [8]byte{0x3a, 0x9c, 0x63, 0x01, 0x00, 0xff, 0xff, 0xff},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, skip, err := parseRawASCII(tc.when, now)

			assert.Equal(t, tc.expect, result)
			assert.Equal(t, tc.expectSkip, skip)
			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestToRawASCIIBytes(t *testing.T) {
	var testCases = []struct {
		name   string
		given  nmea.RawFrame
		expect []byte
	}{
		{
			name: "ok",
			given: nmea.RawFrame{
				Header: nmea.CanBusHeader{
					PGN:         0x1F113, // 1F113 -> 127251 Rate of Turn
					Source:      35,      // 0x23
					Destination: 255,     // 0xff
					Priority:    2,       // 0x02
				},
				Length: 8,
				Data:   [8]byte{0x3a, 0x9c, 0x63, 0x01, 0x00, 0xff, 0xff, 0xff},
			},
			expect: []byte("00:00:00.000 S 09F1FF23 3A 9C 63 01 00 FF FF FF\r\n"),
		},
		{
			name: "ok, shorter, 7 bytes",
			given: nmea.RawFrame{
				Header: nmea.CanBusHeader{
					PGN:         0x1F113, // 1F113 -> 127251 Rate of Turn
					Source:      35,      // 0x23
					Destination: 255,     // 0xff
					Priority:    2,       // 0x02
				},
				Length: 7,
				Data:   [8]byte{0x3a, 0x9c, 0x63, 0x01, 0x00, 0xff, 0xff},
			},
			expect: []byte("00:00:00.000 S 09F1FF23 3A 9C 63 01 00 FF FF\r\n"),
		},
		{
			name: "ok, shorter, 1 byte",
			given: nmea.RawFrame{
				Header: nmea.CanBusHeader{
					PGN:         0x1F113, // 1F113 -> 127251 Rate of Turn
					Source:      35,      // 0x23
					Destination: 255,     // 0xff
					Priority:    2,       // 0x02
				},
				Length: 1,
				Data:   [8]byte{0x3a},
			},
			expect: []byte("00:00:00.000 S 09F1FF23 3A\r\n"),
		},
		{
			name: "ok, ISORequest name",
			given: nmea.RawFrame{
				Header: nmea.CanBusHeader{
					PGN:         uint32(nmea.PGNISORequest), // ISO Request
					Priority:    6,
					Source:      nmea.AddressNull,
					Destination: nmea.AddressGlobal, // everyone/broadcast
				},
				Length: 3,
				Data:   [8]byte{0x0, 0xEE, 0x0}, // PGN 60928 - ISO address claim
			},
			expect: []byte("00:00:00.000 S 18EAFFFE 00 EE 00\r\n"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := toRawASCIIBytes(tc.given)
			assert.Equal(t, tc.expect, result)
		})
	}
}
