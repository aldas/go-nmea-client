package socketcan

type CanBusHeader struct {
	Priority    uint8
	PGN         uint32
	Destination uint8
	Source      uint8
}

// ParseISO11783ID parses can bus header fields from candump CANID (29 bits of 32 bit).
func ParseISO11783ID(canID uint32) CanBusHeader {
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
