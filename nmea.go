package nmea

import (
	"encoding/binary"
	"time"
)

// FastRawPacketMaxSize is maximum size of fast packet multiple packets total length
//
// NMEA200 frame is 8 bytes and to send longer payloads `Fast packet` protocol could be used. In case of fast packet
// nmea message consist of multiple frames where:
// * first frame of message has 2 first bytes reserved and up to 6 following bytes for actual payload
//   - first byte (data[0]) identifies message counter (first 3 bits) and frame counter (5 bits) for that PGN.
//     Message counter is to distinguish simultaneously sent message frames. Frame counter is always 0 for first frame.
//   - second byte (data[1]) indicates message total size in bytes
//
// * second and consecutive frames reserve 1 byte for message counter and frame counter and up to 7 bytes for payload
// Fast packet maximum payload size 223 comes from the fact that first packet can have only 6 bytes of data and following
// frames 7 bytes. As frame counter is 5 bits (0-31 dec) we get maximum by 6 + 31 * 7 = 223 bytes.
const FastRawPacketMaxSize = 223

const ISOTPDataMaxSize = 1785

type RawFrame struct {
	// Time is when frame was read from NMEA bus. Filled by this library.
	Time time.Time

	Header CanBusHeader
	Length uint8 // 1-8
	Data   [8]byte
}

// RawMessage is complete message that is created from single or multiple raw frames assembled together. RawMessage
// could be assembled from multiple nmea/canbus frames thus data length can vary up to 1785 bytes.
type RawMessage struct {
	// Time is when message was read from NMEA bus. Filled by this library.
	Time time.Time

	Header CanBusHeader
	Data   RawData // usually 8 bytes but fast-packets can be up to 223 bytes, assembled multi-packets (ISO-TP) up to 1785 bytes
}

// Message is parsed value of PGN packet(s). Message could be assembled from multiple RawMessage instances.
type Message struct {
	// NodeNAME is unique identifier (ISO Address Claim) for Node in NMEA bus.
	//
	// Helps to identify which physical/logical  device/node was author/source of that message. CanBusHeader.Source is
	// not reliable to identify who/what sent the message as source is "randomly" assigned/claimed with ISO address
	// claim process
	//
	// Value `0` means that Node NAME was unknown. For example AddressMapper may have not yet been able to process NAME
	// for that source. For small/fixed NMEA networks this is perfectly fine as you always know what was the actual source
	// for this Message (PGN).
	NodeNAME uint64 `json:"node_name"`

	Header CanBusHeader `json:"header"`
	Fields FieldValues  `json:"fields"`
}

// +127 = Data not available or Do Not Change; 0x7F
//+126 = Out of range; 0x7E
//+125 = Reserved, 0x7D

// FrameAssembler propose is to assemble multi-frame PGN into single raw NMEA message. Used for fast-packet and ISO-TP
// assembly.
type FrameAssembler interface {
	Assemble(rawFrame RawFrame) (RawMessage, bool, error)
}

type MessageDecoder interface {
	Decode(raw RawMessage) (Message, error)
}

func MarshalRawMessage(raw RawMessage) []byte {
	b := make([]byte, 8+2+3+len(raw.Data))

	binary.LittleEndian.PutUint64(b, uint64(raw.Time.UnixNano())) // 0 - 7
	binary.LittleEndian.PutUint32(b, raw.Header.PGN)              // 8,9
	b[10] = raw.Header.Priority                                   // 10
	b[11] = raw.Header.Source                                     // 11
	b[12] = raw.Header.Destination                                // 12
	copy(b[13:], raw.Data)                                        // 13 - ...

	return b
}
