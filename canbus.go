package nmea

type CanBusHeader struct {
	PGN         uint32 `json:"pgn"`
	Priority    uint8  `json:"priority"`
	Source      uint8  `json:"source"`
	Destination uint8  `json:"destination"`
}

func (h CanBusHeader) ProprietaryType() string { // FIXME: is it necessary?
	// https://copperhilltech.com/blog/design-of-proprietary-parameter-group-numbers-pgns/
	// J1939 PGN groups:
	// 0x000000 - 0x00EE00	PDU1 addressed (SAE assigned) (0 - 60928)
	// 0x00EF00				PDU1 addressed (Manufacturer assigned) (61184)
	// 0x00F000 - 0x00FEFF	PDU2 broadcast (SAE) (61440 - 65279)
	// 0x00FF00 - 0x00FFFF	PDU2 broadcast (Manufacturer assigned) (65280 - 65535)
	// 0x010000 - 0x01EE00	PDU1 addressed (SAE assigned) (65536 - 126464)
	// 0x01EF00				PDU1 addressed (Manufacturer assigned) (126720)
	// 0x01F000 - 0x01FEFF	PDU2 broadcast (SAE) (126976 - 130815)
	// 0x01FF00 - 0x01FFFF	PDU2 broadcast (Manufacturer assigned) (130816 - 131071)

	if h.PGN >= 0xEF00 && h.PGN <= 0xEFFF { // 61184 - 61439
		return "PDU1 (addressed) single-frame"
	} else if h.PGN >= 0xFF00 && h.PGN <= 0xFFFF { // 65280 - 65535
		return "PDU2 (broadcast) single-frame"
	} else if h.PGN >= 0x1EF00 && h.PGN <= 0x1EFFF { // 126720 - 126975
		return "PDU1 (addressed) fast-packet"
	} else if h.PGN >= 0x1FF00 && h.PGN <= 0x1FFFF { // 130816 - 131071
		return "PDU2 (broadcast) fast-packet"
	}
	return ""
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
