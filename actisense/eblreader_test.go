package actisense

import (
	"bufio"
	"bytes"
	"context"
	"github.com/aldas/go-nmea-client"
	test_test "github.com/aldas/go-nmea-client/test"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestEBLFormatDevice_Read(t *testing.T) {
	now := test_test.UTCTime(1665488842) // Tue Oct 11 2022 11:47:22 GMT+0000

	exampleData := test_test.LoadBytes(t, "actisense_w2k1_bst95.ebl")
	r := bytes.NewReader(exampleData)
	wr := bufio.NewReadWriter(bufio.NewReader(r), nil)

	device := NewEBLFormatDevice(wr)
	device.timeNow = func() time.Time {
		return now
	}
	packet, err := device.ReadRawMessage(context.Background())
	if err != nil {
		assert.NoError(t, err)
		return
	}

	firstPacket := nmea.RawMessage{
		Time: now,
		Header: nmea.CanBusHeader{
			PGN:         129025,
			Priority:    2,
			Source:      0,
			Destination: 255,
		},
		Data: nmea.RawData{0x3d, 0x0d, 0xb3, 0x22, 0x48, 0x32, 0x59, 0x0d},
	}
	assert.Equal(t, firstPacket, packet)

	packet, err = device.ReadRawMessage(context.Background())
	if err != nil {
		assert.NoError(t, err)
		return
	}

	secondPacket := nmea.RawMessage{
		Time: now,
		Header: nmea.CanBusHeader{
			PGN:         130843,
			Priority:    7,
			Source:      0,
			Destination: 255,
		},
		Data: nmea.RawData{0x40, 0x0a, 0x3f, 0x9f, 0x09, 0xff, 0x0c, 0x01},
	}
	assert.Equal(t, secondPacket, packet)
}

func TestFromActisenseBST95Message(t *testing.T) {
	now := test_test.UTCTime(1665488842) // Tue Oct 11 2022 11:47:22 GMT+0000

	var testCases = []struct {
		name        string
		whenRaw     []byte
		expect      nmea.RawMessage
		expectError string
	}{
		{
			name:    "ok",
			whenRaw: []byte{0x0e, 0x28, 0x9a, 0x00, 0x01, 0xf8, 0x09, 0x3d, 0x0d, 0xb3, 0x22, 0x48, 0x32, 0x59, 0x0d},
			expect: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					PGN:         129025,
					Priority:    2,
					Source:      0,
					Destination: 255,
				},
				Data: nmea.RawData{0x3d, 0x0d, 0xb3, 0x22, 0x48, 0x32, 0x59, 0x0d},
			},
		},
		{
			name:    "nok, too short, missing data",
			whenRaw: []byte{0x0e, 0x28, 0x9a, 0x00, 0x01, 0xf8, 0x09},
			expect: nmea.RawMessage{
				Time:   time.Time{},
				Header: nmea.CanBusHeader{},
				Data:   nil,
			},
			expectError: "raw message actual length too short to be valid BST-95 message",
		},
		{
			name:    "nok, incorrect length value",
			whenRaw: []byte{0x0e, 0x28, 0x9a, 0x00, 0x01, 0xf8, 0x09, 0x3d, 0x0d, 0xb3, 0x22, 0x48, 0x32, 0x59},
			expect: nmea.RawMessage{
				Time:   time.Time{},
				Header: nmea.CanBusHeader{},
				Data:   nil,
			},
			expectError: "raw message length field does not match actual length",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := fromActisenseBST95Message(tc.whenRaw, now)

			assert.Equal(t, tc.expect, result)
			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
