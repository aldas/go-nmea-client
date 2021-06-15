package nmea

import (
	"encoding/binary"
	"fmt"
	"github.com/aldas/go-nmea-client/canboat"
	"time"
)

/*
 * TODO: Canboat notes:
 * Notes on the NMEA 2000 packet structure
 * ---------------------------------------
 *
 * http://www.nmea.org/Assets/pgn059392.pdf tells us that:
 * - All messages shall set the reserved bit in the CAN ID field to zero on transmit.
 * - Data field reserve bits or reserve bytes shall be filled with ones. i.e. a reserve
 *   byte will be set to a hex value of FF, a single reservie bit would be set to a value of 1.
 * - Data field extra bytes shall be illed with a hex value of FF.
 * - If the PGN in a Command or Request is not recognized by the destination it shall
 *   reply with the PGN 059392 ACK or NACK message using a destination specific address.
 *
 */

/*
 * TODO: Canboat notes:
 * Some packets include a "SID", explained by Maretron as follows:
 * SID: The sequence identifier field is used to tie related PGNs together. For example,
 * the DST100 will transmit identical SIDs for Speed (PGN 128259) and Water depth
 * (128267) to indicate that the readings are linked together (i.e., the data from each
 * PGN was taken at the same time although reported at slightly different times).
 */

/*
 * TODO: Canboat notes:
 * NMEA 2000 uses the 8 'data' bytes as follows:
 * data[0] is an 'order' that increments, or not (depending a bit on implementation).
 * If the size of the packet <= 7 then the data follows in data[1..7]
 * If the size of the packet > 7 then the next byte data[1] is the size of the payload
 * and data[0] is divided into 5 bits index into the fast packet, and 3 bits 'order
 * that increases.
 * This means that for 'fast packets' the first bucket (sub-packet) contains 6 payload
 * bytes and 7 for remaining. Since the max index is 31, the maximal payload is
 * 6 + 31 * 7 = 223 bytes
 */

// FastRawPacketMaxSize is maximum size of fast packet multiple packets total length
// TODO: Canboat notes:
// NMEA 2000 uses the 8 'data' bytes as follows:  data[0] is an 'order' that increments, or not (depending a bit on
// implementation).
// If the size of the packet <= 7 then the data follows in data[1..7]
// If the size of the packet > 7 then the next byte data[1] is the size of the payload  and data[0] is divided into
// 5 bits index into the fast packet, and 3 bits 'order that increases.
// This means that for 'fast packets' the first bucket (sub-packet) contains 6 payload bytes and 7 for remaining.
// Since the max index is 31, the maximal payload is  6 + 31 * 7 = 223 bytes
const FastRawPacketMaxSize = 223

// RawMessage is raw message read from device containing NMEA message
type RawMessage struct {
	// Time is when message was read from NMEA bus. Filled by the library.
	Time        time.Time
	Priority    uint8
	PGN         uint32 // note: not unique, some messages have same PGN but different fields (messages are used in sequence, see pgn 126208)
	Destination uint8
	Source      uint8
	Timestamp   uint32
	Length      uint8
	Data        []byte
}

// CustomPGN is generic PGN structure that can contain any arbitrary field values
type CustomPGN struct {
	PGN         uint32      `json:"pgn"`
	Destination uint8       `json:"destination"`
	Source      uint8       `json:"source"`
	Fields      FieldValues `json:"fields"`
}

// FieldValues is slice of FieldValue
type FieldValues []FieldValue

// FieldValue hold extracted and processed value for PGN field
type FieldValue struct {
	ID    string      `json:"id"`
	Type  string      `json:"type"`
	Value interface{} `json:"value"` // normalize to: string, float64, int64, uint64, []byte, []uint8
}

// ParsePGN parses message into CustomPGN
func ParsePGN(pgnConf canboat.PGN, raw RawMessage) (CustomPGN, error) {
	tmpFields := make([]FieldValue, 0)

	toField := len(pgnConf.Fields) - int(pgnConf.RepeatingFields)
	for _, f := range pgnConf.Fields[0:toField] {
		if f.Reserved {
			continue
		}
		tmp, err := ParseField(f, raw.Data)
		if err != nil {
			if err == errNoFieldValue {
				continue
			}
			return CustomPGN{}, err
		}
		tmpFields = append(tmpFields, tmp)
	}
	// TODO: for over pgnConf.RepeatingFields

	return CustomPGN{
		PGN:    pgnConf.PGN,
		Fields: tmpFields,
	}, nil
}

// ParseField parses field from message data
func ParseField(field canboat.Field, rawData []byte) (FieldValue, error) {
	// "" | Integer | Lookup table | Bitfield | Binary data | Manufacturer code <-- are printed exactly the same in canboat

	// existing types:
	//      <--- empty, bitlen varies from 2 to 2056  (1;11;12;2;3;30;4;40;5;56;6;7)
	//Bitfield 															(has special reader) (2;4;6)
	//Manufacturer code            <--- note: bitlen seems to be always 11
	//Lookup table                 <--- note: bitlen seems to be 7,3,2,4,1,6,5
	//Integer					   <--- note: bitlen seems to be (2;6;7;10;14;30)
	//
	//Binary data                  <--- note: bitlen seems to be 1,2,3,4,5,6,7,12,19,21,22,40,44,80,112,1736,1768,2040
	//ASCII or UNICODE string starting with length and control byte    	(has special reader) (128,272)
	//ASCII text														(has special reader) (56;80;136;160;256;288;2040)
	//String with start/stop byte										(has special reader) (2040;2056)
	//Decimal encoded number	  <--- note: bitlen seems to be always (40) https://www.nmea.org/Assets/20130720%20%20dsc%20technical%20corrigendum%20v1..pdf
	//ASCII string starting with length byte  							(has special reader) (40;80;96;104;112;256)

	// see: https://github.com/canboat/canboat/issues/92#issuecomment-326537825
	// The CANBoat analyzer has:
	//
	//    RES_ASCII (fixed field length, no byte count)
	//    RES_STRING (starts with 1 byte length byte)
	//    RES_STRINGLZ (starts with 1 byte length byte and terminated by zero byte)
	//    RES_STRINGLAU (starts with 1 byte length byte and 1 byte ASCII/UNICODE byte)
	//
	//The RES_STRINGLZ variation is used by Fusion for the old company specific entertainment (mainly audio) PGNs.
	//
	//The RES_STRINGLAU variation is used by the new N2K PGNs since 2015 for DSC and entertainment.
	//My CANBoat analyzer doesn't yet properly handle the UTF16 content variation. (Why in the world you would choose
	//UTF16 instead of UTF8 when coming up with this in 2014-2015 is really beyond me, but okay.)

	// FIXME: special cases for types with "readers"
	// var reader = fieldTypeReaders[field.Type] if ( reader ) { value = reader(pgn, field, bs)

	switch field.Type {
	case canboat.FieldTypeUnknownReal,
		canboat.FieldTypeInteger,
		canboat.FieldTypeEnumValue,
		canboat.FieldTypeBitValues,
		canboat.FieldTypeManufacturerCode:
		return parseRealField(field, rawData)
	//case FieldTypeBinaryData:
	//	return FieldValue{}, errNoFieldValue
	default:
		return FieldValue{}, fmt.Errorf("failed to parse field value of type: %v", field.Type)
	}
}

var errNoFieldValue = fmt.Errorf("parsed magic no field value")

func parseRealField(field canboat.Field, rawData []byte) (FieldValue, error) {
	result := FieldValue{
		ID: field.ID,
	}

	// normalize values to:
	// 	* uint64
	// 	* int64
	// 	* float64
	//  * <no value> errNoFieldValue when "magic" no value is detected

	var tmpUint64 uint64
	var tmpInt64 int64

	offset := field.BitOffset / 8
	switch field.BitLength {
	case 0: // readVariableLengthField(
	case 8:
		b := rawData[offset]
		if field.Signed {
			if b == 0x7f {
				return FieldValue{}, errNoFieldValue
			}
			tmpInt64 = int64(b)
		} else {
			if b == 0xff {
				return FieldValue{}, errNoFieldValue
			}
			tmpUint64 = uint64(b)
		}
	case 16:
		vTmp := binary.LittleEndian.Uint16(rawData[offset : offset+2])

		if field.Signed {
			if vTmp == 0x7fff {
				return FieldValue{}, errNoFieldValue
			}
			tmpInt64 = int64(vTmp)
		} else {
			if vTmp == 0xffff {
				return FieldValue{}, errNoFieldValue
			}
			tmpUint64 = uint64(vTmp)
		}
	case 24:
	case 32:
		vTmp := binary.LittleEndian.Uint32(rawData[offset : offset+4])
		if field.Signed {
			if vTmp == 0x7fffffff {
				return FieldValue{}, errNoFieldValue
			}
			tmpInt64 = int64(vTmp)
		} else {
			if vTmp == 0xffffffff {
				return FieldValue{}, errNoFieldValue
			}
			tmpUint64 = uint64(vTmp)
		}
	case 48:
	case 64:
		vTmp := binary.LittleEndian.Uint64(rawData[offset : offset+8])
		if field.Signed {
			if vTmp == 0x7fffffffffffffff {
				return FieldValue{}, errNoFieldValue
			}
			tmpInt64 = int64(vTmp)
		} else {
			if vTmp == 0xffffffffffffffff {
				return FieldValue{}, errNoFieldValue
			}
			tmpUint64 = vTmp
		}
	default:
		tmp, err := parseBytesValue(field, rawData)
		if err != nil {
			return FieldValue{}, err
		}
		tmpUint64 = tmp
		tmpInt64 = int64(tmp)

		if field.BitLength > 1 {
			// handle "magick" no value cases signed=>0x7ff(f) , unsigned=>0xfff(f)
			mask := ^uint64(0) >> (64 - int(field.BitLength))
			if field.Signed {
				mask = mask >> 1
			}
			if tmp == mask {
				return FieldValue{}, errNoFieldValue
			}
		}
	}

	result.Type = "uint64"
	result.Value = tmpUint64
	if field.Signed {
		result.Type = "int64"
		result.Value = tmpInt64
	}

	if field.Resolution > 0.0 {
		result.Type = "float64"
		if field.Signed {
			result.Value = float64(tmpInt64) * float64(field.Resolution)
		} else {
			result.Value = float64(tmpUint64) * float64(field.Resolution)
		}
	}

	//if field.EnumValues != nil {
	// FIXME: if (field.Id === "timeStamp" && value < 60) { to string } else { lookup() }
	// some ais "timeStamp" fields have special meaning >60
	//}

	// FIXME: if ( field.Name === 'Industry Code' && _.isNumber(value) && runPostProcessor ) { value = getIndustryName(value)

	return result, nil
}

func parseBytesValue(field canboat.Field, rawData []byte) (uint64, error) {
	if field.BitLength > 64 {
		return 0, fmt.Errorf("field bit length larger than can be parsed to byte value")
	}
	var result uint64
	bytesLen := field.BitLength / 8
	offset := field.BitOffset / 8

	if len(rawData) <= int(offset) {
		return 0, fmt.Errorf("field (bit)offset is out of bounds of message data")
	}
	result = uint64(rawData[offset])
	bitIndex := field.BitOffset % 8
	// in case offset do not start off exactly at the start of byte clear those bits
	if bitIndex != 0 {
		result = result >> bitIndex
	}
	bitIndex = 8 - bitIndex
	if bytesLen > 0 {
		for _, b := range rawData[offset+1 : offset+1+bytesLen] {
			b2 := uint64(b) << bitIndex
			result = result + b2
			bitIndex = bitIndex + 8
		}
	}
	// in case we do not end exactly at the end of last byte clear those bits at the end
	mask := ^uint64(0) >> (64 - field.BitLength)
	result = result & mask

	return result, nil
}
