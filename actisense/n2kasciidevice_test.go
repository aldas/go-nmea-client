package actisense

import (
	"context"
	"github.com/aldas/go-nmea-client"
	test_test "github.com/aldas/go-nmea-client/test"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestN2kAsciiDevice_ReadRawMessage(t *testing.T) {
	now := test_test.UTCTime(1665488842) // Tue Oct 11 2022 11:47:22 GMT+0000

	var testCases = []struct {
		name             string
		reads            []test_test.ReadResult
		expect           nmea.RawMessage
		expectReadBuffer []byte
		expectError      string
	}{
		{
			name: "ok, single read",
			reads: []test_test.ReadResult{
				{
					Read: []byte("A173321.107 23FF7 1F513 012F3070002F30709F    \n"),
					Err:  nil,
				},
			},
			expect: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					PGN:         0x1F513, // 1F513 -> 128275 Distance Log
					Source:      35,      // 0x23
					Destination: 255,     // 0xFF
					Priority:    7,       // 0x07
				},
				Data: []byte{0x01, 0x2F, 0x30, 0x70, 0x00, 0x2F, 0x30, 0x70, 0x9F},
			},
			expectReadBuffer: []byte{},
		},
		{
			name: "ok, multiple reads to assemble message",
			reads: []test_test.ReadResult{
				{Read: []byte("A173321.107 23FF7 "), Err: nil},
				{Read: []byte("1F513 012F3070002F30709F    \nAXXX"), Err: nil},
			},
			expect: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					PGN:         0x1F513, // 1F513 -> 128275 Distance Log
					Source:      35,      // 0x23
					Destination: 255,     // 0xFF
					Priority:    7,       // 0x07
				},
				Data: []byte{0x01, 0x2F, 0x30, 0x70, 0x00, 0x2F, 0x30, 0x70, 0x9F},
			},
			expectReadBuffer: []byte(`AXXX`),
		},
		{
			name: "ok, multiple reads to assemble message, first in incomplete message",
			reads: []test_test.ReadResult{
				{Read: []byte("GARBAGE\nA173321.107 23FF7 "), Err: nil},
				{Read: []byte("1F513 012F3070002F30709F    \nAXXX"), Err: nil},
			},
			expect: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					PGN:         0x1F513, // 1F513 -> 128275 Distance Log
					Source:      35,      // 0x23
					Destination: 255,     // 0xFF
					Priority:    7,       // 0x07
				},
				Data: []byte{0x01, 0x2F, 0x30, 0x70, 0x00, 0x2F, 0x30, 0x70, 0x9F},
			},
			expectReadBuffer: []byte(`AXXX`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockReader := &test_test.MockReaderWriter{Reads: tc.reads}

			device := NewN2kASCIIDevice(mockReader)
			device.timeNow = func() time.Time {
				return now
			}

			result, err := device.ReadRawMessage(context.Background())

			assert.Equal(t, tc.expect, result)
			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}

			// internal checks
			assert.Equal(t, tc.expectReadBuffer, device.readBuffer[0:len(tc.expectReadBuffer)])
			assert.Equal(t, len(tc.expectReadBuffer), device.readIndex)
		})
	}
}

func TestParseN2KAscii(t *testing.T) {
	now := test_test.UTCTime(1665488842) // Tue Oct 11 2022 11:47:22 GMT+0000

	var testCases = []struct {
		name        string
		when        []byte
		expect      nmea.RawMessage
		expectSkip  bool
		expectError string
	}{
		{
			name: "ok",
			when: []byte("A173321.107 23FF7 1F513 012F3070002F30709F    \n"),
			expect: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					PGN:         0x1F513, // 1F513 -> 128275 Distance Log
					Source:      35,      // 0x23
					Destination: 255,     // 0xFF
					Priority:    7,       // 0x07
				},
				Data: []byte{0x01, 0x2F, 0x30, 0x70, 0x00, 0x2F, 0x30, 0x70, 0x9F},
			},
			expectSkip: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, skip, err := parseN2KAscii(tc.when, now)

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
