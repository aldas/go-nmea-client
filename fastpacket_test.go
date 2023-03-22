package nmea

import (
	test_test "github.com/aldas/go-nmea-client/test"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// Example fast-packet
// PGN: 1FD13 - Meteorological Station Data
// candump output:
//
// Canboat analyzer output:
//
//	{
//	 "timestamp": "2023-03-17T00:05:10.046",
//	 "prio": 6,
//	 "src": 35,
//	 "dst": 255,
//	 "pgn": 130323,
//	 "description": "Meteorological Station Data",
//	 "fields": {
//	   "Mode": 0,
//	   "Measurement Date": "2022.09.13",
//	   "Measurement Time": "08:23:36.5000",
//	   "Station Latitude": 58.2156683,
//	   "Station Longitude": 22.39509,
//	   "Wind Speed": 1.64,
//	   "Wind Direction": 293.3,
//	   "Wind Reference": "Apparent",
//	   "Atmospheric Pressure": 1.008,
//	   "Ambient Temperature": 12.5
//	 }
//	}
//
//00:05:10.032 R 19FD1323 60 1E F0 30 4B 08 AC 02
//00:05:10.038 R 19FD1323 61 12 8B 01 B3 22 34 38
//00:05:10.041 R 19FD1323 62 59 0D A4 00 F5 C7 FA
//00:05:10.041 R 19FD1323 63 FF FF F0 03 95 6F 02
//00:05:10.046 R 19FD1323 64 01 02 01 FF FF FF FF
func exampleFPS() fastPacketSequence {
	return fastPacketSequence{
		header:                CanBusHeader{PGN: 130323, Priority: 6, Source: 35, Destination: 255},
		lastReceivedFrameTime: test_test.UTCTime(1665488842),
		receivedFramesCount:   5,
		sequence:              6,
		length:                30, // 0x1E, 5 frames, 6,7,7,7,3
		completeFramesMask:    0b11111,
		receivedFramesMask:    0b11111,
		data: [223]byte{
			0xF0, 0x30, 0x4B, 0x08, 0xAC, 0x02, // 00:05:10.032 R 19FD1323 60 1E F0 30 4B 08 AC 02
			0x12, 0x8B, 0x01, 0xB3, 0x22, 0x34, 0x38, //00:05:10.038 R 19FD1323 61 12 8B 01 B3 22 34 38
			0x59, 0x0D, 0xA4, 0x00, 0xF5, 0xC7, 0xFA, //00:05:10.041 R 19FD1323 62 59 0D A4 00 F5 C7 FA
			0xFF, 0xFF, 0xF0, 0x03, 0x95, 0x6F, 0x02, //00:05:10.041 R 19FD1323 63 FF FF F0 03 95 6F 02
			0x01, 0x02, 0x01, 0xFF, 0xFF, 0xFF, 0xFF, //00:05:10.046 R 19FD1323 64 01 02 01 FF FF FF FF
		},
	}
}

func TestFastPacketMessage_Append(t *testing.T) {
	now := test_test.UTCTime(1665488842) // Tue Oct 11 2022 11:47:22 GMT+0000

	var testCases = []struct {
		name       string
		given      fastPacketSequence
		when       RawFrame
		expect     fastPacketSequence
		expectDone bool
	}{
		{
			name: "ok, append second frame, in order",
			given: fastPacketSequence{
				header:                CanBusHeader{PGN: 130323, Priority: 6, Source: 35, Destination: 255},
				lastReceivedFrameTime: now.Add(-50 * time.Millisecond),
				receivedFramesCount:   1,
				sequence:              6,
				length:                30, // 0x1E, 5 frames, 6,7,7,7,3
				completeFramesMask:    0b11111,
				receivedFramesMask:    0b1,
				data: [223]byte{
					0xF0, 0x30, 0x4B, 0x08, 0xAC, 0x02, // 00:05:10.032 R 19FD1323 60 1E F0 30 4B 08 AC 02
				},
			},
			when: RawFrame{
				Time:   now,
				Header: CanBusHeader{PGN: 130323, Priority: 6, Source: 35, Destination: 255},
				Length: 8,
				Data:   [8]byte{0x61, 0x12, 0x8B, 0x01, 0xB3, 0x22, 0x34, 0x38},
			},
			expectDone: false,
			expect: fastPacketSequence{
				header: CanBusHeader{PGN: 130323, Priority: 6, Source: 35, Destination: 255},

				lastReceivedFrameTime: now,
				receivedFramesCount:   2,
				completeFramesMask:    0b11111,

				sequence:           6,
				length:             30, // 0x1E, 5 frames, 6,7,7,7,3
				receivedFramesMask: 0b11,

				data: [223]byte{
					0xF0, 0x30, 0x4B, 0x08, 0xAC, 0x02, //00:05:10.032 R 19FD1323 60 1E F0 30 4B 08 AC 02
					0x12, 0x8B, 0x01, 0xB3, 0x22, 0x34, 0x38, //00:05:10.038 R 19FD1323 61 12 8B 01 B3 22 34 38
				},
			},
		},
		{
			name: "ok, append last frame, in order",
			given: fastPacketSequence{
				header:                CanBusHeader{PGN: 130323, Priority: 6, Source: 35, Destination: 255},
				lastReceivedFrameTime: now.Add(-50 * time.Millisecond),
				receivedFramesCount:   4,
				sequence:              6,
				length:                30, // 0x1E, 5 frames, 6,7,7,7,3
				completeFramesMask:    0b11111,
				receivedFramesMask:    0b1111,
				data: [223]byte{
					0xF0, 0x30, 0x4B, 0x08, 0xAC, 0x02, // 00:05:10.032 R 19FD1323 60 1E F0 30 4B 08 AC 02
					0x12, 0x8B, 0x01, 0xB3, 0x22, 0x34, 0x38, //00:05:10.038 R 19FD1323 61 12 8B 01 B3 22 34 38
					0x59, 0x0D, 0xA4, 0x00, 0xF5, 0xC7, 0xFA, //00:05:10.041 R 19FD1323 62 59 0D A4 00 F5 C7 FA
					0xFF, 0xFF, 0xF0, 0x03, 0x95, 0x6F, 0x02, //00:05:10.041 R 19FD1323 63 FF FF F0 03 95 6F 02
				},
			},
			when: RawFrame{
				Time:   now,
				Header: CanBusHeader{PGN: 130323, Priority: 6, Source: 35, Destination: 255},
				Length: 8,
				Data:   [8]byte{0x64, 0x01, 0x02, 0x01, 0xFF, 0xFF, 0xFF, 0xFF}, //00:05:10.046 R 19FD1323 64 01 02 01 FF FF FF FF
			},
			expectDone: true,
			expect: fastPacketSequence{
				header:                CanBusHeader{PGN: 130323, Priority: 6, Source: 35, Destination: 255},
				lastReceivedFrameTime: now,
				receivedFramesCount:   5,
				sequence:              6,
				length:                30, // 0x1E, 5 frames, 6,7,7,7,3
				completeFramesMask:    0b11111,
				receivedFramesMask:    0b11111,
				data: [223]byte{
					0xF0, 0x30, 0x4B, 0x08, 0xAC, 0x02, // 00:05:10.032 R 19FD1323 60 1E F0 30 4B 08 AC 02
					0x12, 0x8B, 0x01, 0xB3, 0x22, 0x34, 0x38, //00:05:10.038 R 19FD1323 61 12 8B 01 B3 22 34 38
					0x59, 0x0D, 0xA4, 0x00, 0xF5, 0xC7, 0xFA, //00:05:10.041 R 19FD1323 62 59 0D A4 00 F5 C7 FA
					0xFF, 0xFF, 0xF0, 0x03, 0x95, 0x6F, 0x02, //00:05:10.041 R 19FD1323 63 FF FF F0 03 95 6F 02
					0x01, 0x02, 0x01, 0xFF, 0xFF, 0xFF, 0xFF, //00:05:10.046 R 19FD1323 64 01 02 01 FF FF FF FF
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fpm := tc.given
			done := fpm.Append(tc.when)

			assert.Equal(t, tc.expectDone, done)
			assert.Equal(t, tc.expect, fpm)
		})
	}
}

func TestFastPacketSequence_To(t *testing.T) {
	fp := exampleFPS()

	msg := RawMessage{} // allocates data
	fp.To(&msg)

	expected := RawMessage{
		Time:   test_test.UTCTime(1665488842),
		Header: CanBusHeader{PGN: 130323, Priority: 6, Source: 35, Destination: 255},
		Data: []byte{
			0xF0, 0x30, 0x4B, 0x08, 0xAC, 0x02,
			0x12, 0x8B, 0x01, 0xB3, 0x22, 0x34, 0x38,
			0x59, 0x0D, 0xA4, 0x00, 0xF5, 0xC7, 0xFA,
			0xFF, 0xFF, 0xF0, 0x03, 0x95, 0x6F, 0x02,
			0x01, 0x02, 0x01,
		},
	}
	assert.Equal(t, expected, msg)
}

func TestFastPacketSequence_As(t *testing.T) {
	fp := exampleFPS()

	msg := fp.As()
	expected := RawMessage{
		Time:   test_test.UTCTime(1665488842),
		Header: CanBusHeader{PGN: 130323, Priority: 6, Source: 35, Destination: 255},
		Data: []byte{
			0xF0, 0x30, 0x4B, 0x08, 0xAC, 0x02,
			0x12, 0x8B, 0x01, 0xB3, 0x22, 0x34, 0x38,
			0x59, 0x0D, 0xA4, 0x00, 0xF5, 0xC7, 0xFA,
			0xFF, 0xFF, 0xF0, 0x03, 0x95, 0x6F, 0x02,
			0x01, 0x02, 0x01,
		},
	}
	assert.Equal(t, expected, msg)
}

func TestFastPacketSequence_Reset(t *testing.T) {
	fp := exampleFPS()

	fp.Reset()

	expected := fastPacketSequence{
		data: [223]byte{
			0xF0, 0x30, 0x4B, 0x08, 0xAC, 0x02, // 00:05:10.032 R 19FD1323 60 1E F0 30 4B 08 AC 02
			0x12, 0x8B, 0x01, 0xB3, 0x22, 0x34, 0x38, //00:05:10.038 R 19FD1323 61 12 8B 01 B3 22 34 38
			0x59, 0x0D, 0xA4, 0x00, 0xF5, 0xC7, 0xFA, //00:05:10.041 R 19FD1323 62 59 0D A4 00 F5 C7 FA
			0xFF, 0xFF, 0xF0, 0x03, 0x95, 0x6F, 0x02, //00:05:10.041 R 19FD1323 63 FF FF F0 03 95 6F 02
			0x01, 0x02, 0x01, 0xFF, 0xFF, 0xFF, 0xFF, //00:05:10.046 R 19FD1323 64 01 02 01 FF FF FF FF
		},
	}
	assert.Equal(t, expected, fp)
}

func TestFastPacketAssembler_Assemble(t *testing.T) {
	now := test_test.UTCTime(1665488842)
	var testCases = []struct {
		name           string
		whenFrames     []RawFrame
		expectComplete bool
		expectMessage  RawMessage
	}{
		{
			name: "ok, 130323 fast-packet",
			whenFrames: []RawFrame{
				{
					Time:   now.Add(-4 * 50 * time.Millisecond),
					Header: CanBusHeader{PGN: 130323, Priority: 6, Source: 35, Destination: 255},
					Length: 8,
					Data:   [8]byte{0x60, 0x1E, 0xF0, 0x30, 0x4B, 0x08, 0xAC, 0x02},
				},
				{
					Time:   now.Add(-3 * 50 * time.Millisecond),
					Header: CanBusHeader{PGN: 130323, Priority: 6, Source: 35, Destination: 255},
					Length: 8,
					Data:   [8]byte{0x61, 0x12, 0x8B, 0x01, 0xB3, 0x22, 0x34, 0x38},
				},
				{
					Time:   now.Add(-2 * 50 * time.Millisecond),
					Header: CanBusHeader{PGN: 130323, Priority: 6, Source: 35, Destination: 255},
					Length: 8,
					Data:   [8]byte{0x62, 0x59, 0x0D, 0xA4, 0x00, 0xF5, 0xC7, 0xFA},
				},
				{
					Time:   now.Add(-1 * 50 * time.Millisecond),
					Header: CanBusHeader{PGN: 130323, Priority: 6, Source: 35, Destination: 255},
					Length: 8,
					Data:   [8]byte{0x63, 0xFF, 0xFF, 0xF0, 0x03, 0x95, 0x6F, 0x02},
				},
				{
					Time:   now,
					Header: CanBusHeader{PGN: 130323, Priority: 6, Source: 35, Destination: 255},
					Length: 8,
					Data:   [8]byte{0x64, 0x01, 0x02, 0x01, 0xFF, 0xFF, 0xFF, 0xFF},
				},
			},
			expectComplete: true,
			expectMessage: RawMessage{
				Time:   now,
				Header: CanBusHeader{PGN: 130323, Priority: 6, Source: 35, Destination: 255},
				Data: []byte{
					0xF0, 0x30, 0x4B, 0x08, 0xAC, 0x02, // 00:05:10.032 R 19FD1323 60 1E F0 30 4B 08 AC 02
					0x12, 0x8B, 0x01, 0xB3, 0x22, 0x34, 0x38, //00:05:10.038 R 19FD1323 61 12 8B 01 B3 22 34 38
					0x59, 0x0D, 0xA4, 0x00, 0xF5, 0xC7, 0xFA, //00:05:10.041 R 19FD1323 62 59 0D A4 00 F5 C7 FA
					0xFF, 0xFF, 0xF0, 0x03, 0x95, 0x6F, 0x02, //00:05:10.041 R 19FD1323 63 FF FF F0 03 95 6F 02
					0x01, 0x02, 0x01, //00:05:10.046 R 19FD1323 64 01 02 01 FF FF FF FF
				},
			},
		},
		{
			name: "ok, single packet",
			whenFrames: []RawFrame{
				{
					Time:   now,
					Header: CanBusHeader{PGN: uint32(PGNISORequest), Priority: 6, Source: AddressNull, Destination: 32},
					Length: 3,
					Data:   [8]byte{0x0, 0xEE, 0x0},
				},
			},
			expectComplete: true,
			expectMessage: RawMessage{
				Time:   now,
				Header: CanBusHeader{PGN: uint32(PGNISORequest), Priority: 6, Source: AddressNull, Destination: 32},
				Data: []byte{
					0x0, 0xEE, 0x0,
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fpa := NewFastPacketAssembler([]uint32{126983, 61184, 130323})
			fpa.now = func() time.Time {
				return now
			}

			complete := false
			var msg RawMessage
			for _, f := range tc.whenFrames {
				complete = fpa.Assemble(f, &msg)
			}
			assert.Equal(t, tc.expectComplete, complete)
			assert.Equal(t, tc.expectMessage, msg)
		})
	}
}
