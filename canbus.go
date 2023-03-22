package nmea

type CanBusHeader struct {
	PGN         uint32 `json:"pgn"`
	Priority    uint8  `json:"priority"`
	Source      uint8  `json:"source"`
	Destination uint8  `json:"destination"`
}

func (h CanBusHeader) Uint32() uint32 {
	canID := uint32(h.Source) // bit 0-7

	pf := uint8(h.PGN)
	if pf < 240 {
		canID |= uint32(h.Destination) << 8 // bits 8-15
	}
	canID |= h.PGN << 8                        // bits 16-24
	canID = canID | uint32(h.Priority&0x7)<<26 // bit 26,27,28
	return canID                               // this need to be turned to big endian when written to the wire
}

// ParseCANID parses can bus header fields from CANID (29 bits of 32 bit).
func ParseCANID(canID uint32) CanBusHeader {
	result := CanBusHeader{
		Priority: uint8((canID >> 26) & 0x7), // bit 26,27,28
		Source:   uint8(canID),               // bit 0-7
	}
	ps := uint8(canID >> 8)         // bits 8-15
	pduFormat := uint8(canID >> 16) // bits 16-23
	rAndDP := uint8(canID>>24) & 3  // bits 24,25
	pgn := (uint32(rAndDP) << 16) + uint32(pduFormat)<<8
	if pduFormat < 240 {
		result.Destination = ps
		result.PGN = pgn
	} else {
		result.Destination = AddressGlobal // 0xff is broadcast to all
		result.PGN = pgn + uint32(ps)
	}
	return result
}
