package nmea

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseCANID(t *testing.T) {
	var testCases = []struct {
		name   string
		canID  uint32
		expect CanBusHeader
	}{
		{
			name:  "ok, 0F001DA1",
			canID: 251665825, // 0F001DA1
			expect: CanBusHeader{
				Priority:    3,
				PGN:         196608, // 0x30000
				Destination: 29,     // 1D
				Source:      161,    // A1
			},
		},
		{
			name:  "ok, 0F101DB5",
			canID: 252714421, // 0F101DB5
			expect: CanBusHeader{
				Priority:    3,
				PGN:         0x31000,
				Destination: 29,  // 1D
				Source:      181, // B5
			},
		},
		{
			name:  "ok, 0F101DA1",
			canID: 252714401, // 0F101DA1
			expect: CanBusHeader{
				Priority:    3,
				PGN:         0x31000,
				Destination: 29,  // 1D
				Source:      161, // A1
			},
		},
		{
			name:  "ok, 0F0007B8",
			canID: 251660216, // 0F0007B8
			expect: CanBusHeader{
				Priority:    3,
				PGN:         196608, // 0x30000
				Destination: 7,      // 07
				Source:      184,    // B8
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			header := ParseCANID(tc.canID)
			assert.Equal(t, tc.expect, header)
		})
	}
}

func TestCanBusHeader_Uint32(t *testing.T) {
	var testCases = []struct {
		name   string
		when   CanBusHeader
		expect uint32
	}{
		{
			name: "ok, 59904 ISORequest broadcast from nulladdr",
			when: CanBusHeader{
				PGN:         uint32(PGNISORequest), // ISO Request
				Priority:    6,
				Source:      AddressNull,
				Destination: AddressGlobal, // everyone/broadcast
			},
			expect: 0x18eafffe,
		},
		{
			name: "ok, 130311",
			when: CanBusHeader{
				PGN:         130311, // 0x1FD07
				Priority:    5,
				Source:      23,  // 0x17
				Destination: 255, // 0xFF
			},
			expect: 0x15fdff17,
		},
		{
			name: "ok, 130310",
			when: CanBusHeader{
				PGN:         130310,
				Priority:    5,
				Source:      23,  // 0x17
				Destination: 255, // 0xFF
			},
			expect: 0x15fdff17,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.when.Uint32()
			assert.Equal(t, tc.expect, result)
		})
	}
}
