package socketcan

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseISO11783ID(t *testing.T) {
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
			header := ParseISO11783ID(tc.canID)
			assert.Equal(t, tc.expect, header)
		})
	}
}
