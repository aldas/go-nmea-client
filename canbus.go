package nmea

type CanBusHeader struct {
	PGN         uint32 `json:"pgn"`
	Source      uint8  `json:"source"`
	Destination uint8  `json:"destination"`
	Priority    uint8  `json:"priority"`
}

func (h CanBusHeader) ProprietaryType() string { // FIXME: is it necessary?
	if h.PGN >= 0xEF00 && h.PGN <= 0xEFFF {
		return "PDU1 (addressed) single-frame"
	} else if h.PGN >= 0xFF00 && h.PGN <= 0xFFFF {
		return "PDU2 (nonaddressed) single-frame"
	} else if h.PGN >= 0x1EF00 && h.PGN <= 0x1EFFF {
		return "PDU1 (addressed) fast-packet"
	} else if h.PGN >= 0x1FF00 && h.PGN <= 0x1FFFF {
		return "PDU2 (nonaddressed) fast-packet"
	}
	return ""
}

// ParseCANID parses can bus header fields from CANID (29 bits of 32 bit).
func ParseCANID(canID uint32) CanBusHeader {
	result := CanBusHeader{
		Priority: uint8((canID >> 26) & 0x7), // bit 26,27,28
		Source:   uint8(canID),               // bit 0-7
	}
	pduFormat := uint8(canID >> 16) // bits 16-23
	ps := uint8(canID >> 8)         // bits 8-15
	rAndDP := uint8(canID>>24) & 3  // 3 = first and second bit
	pgn := (uint32(rAndDP) << 16) + uint32(pduFormat)<<8
	if pduFormat < 240 {
		result.Destination = ps
		result.PGN = pgn
	} else {
		result.Destination = 0xff // 0xff is broadcast to all
		result.PGN = pgn + uint32(ps)
	}
	return result
}
