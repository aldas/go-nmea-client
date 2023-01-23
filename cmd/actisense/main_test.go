package main

import (
	"github.com/aldas/go-nmea-client"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestParseLine(t *testing.T) {
	line := "6,59904,0,255,3,14,f0,01" // Request (pgn=59904) for PGN 126996 (product info) from all devices (dst=255)
	msg, err := parseLine(line)
	assert.NoError(t, err)
	assert.Equal(t, nmea.RawMessage{
		Time: time.Time{},
		Header: nmea.CanBusHeader{
			PGN:         59904,
			Source:      0,
			Destination: 255,
			Priority:    6,
		},
		Data: []byte{0x14, 0xf0, 0x01},
	}, msg)
}
