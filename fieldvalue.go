package nmea

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"time"
	"unicode/utf16"
)

// https://www.nmea.org/Assets/2000-explained-white-paper.pdf Page 14
// Refers 3 special values "no data", "out of range" and "reserved"
// https://www.maretron.com/support/manuals/EMS100UM_1.1.html from `EMS100 Engine Monitoring System User's Manual`
// `Appendix A 'NMEA 2000' Interfacing`
// Quote "Note: For integer values, the most positive three values are reserved; e.g., for 8-bit unsigned integers,
// the values 0xFD, 0xFE, 0xFF are reserved, and for 8-bit signed integers, the values 0x7D, 0x7E, 0x7F are
// reserved. The most positive value (0xFF and 0x7F, respectively, for the 8-bit examples) represents
// Data Not Available."
var (
	// ErrValueNoData indicates that field has no data (for example 8bits uint8=>0xFF, int8=>0x7F)
	ErrValueNoData = errors.New("field value has no data")
	// ErrValueOutOfRange indicates that field value is out of valid range (for example 8bits uint8=>0xFE, int8=>0x7E)
	ErrValueOutOfRange = errors.New("field value out of range")
	// ErrValueReserved indicates that field is reserved (for example 8bits uint8=>0xFD, int8=>0x7D)
	ErrValueReserved = errors.New("field value is reserved")
)

var epoch = time.Unix(0, 0).UTC()

// FieldValues is slice of FieldValue
type FieldValues []FieldValue

// FieldValue hold extracted and processed value for PGN field
type FieldValue struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	// normalized to:
	// * string,
	// * float64,
	// * int64,
	// * uint64,
	// * []byte,
	// * time.Duration,
	// * time.Time,
	// * nmea.EnumValue,
	// * [][]nmea.EnumValue <-- for repeating fieldsets/groups
	Value interface{} `json:"value"`
}

// AsFloat64 converts value to float64 if it is possible.
func (f FieldValue) AsFloat64() (float64, bool) {
	switch v := f.Value.(type) {
	case float64:
		return v, true
	case int64:
		return float64(v), true
	case uint64:
		return float64(v), true
	case time.Duration:
		return float64(v), true
	case time.Time:
		return float64(v.UnixNano()), true
	}
	return 0, false
}

func (fvs FieldValues) FindByID(ID string) (FieldValue, bool) {
	for _, f := range fvs {
		if f.ID == ID {
			return f, true
		}
	}
	return FieldValue{}, false
}

type RawData []byte

func (d *RawData) DecodeBytes(bitOffset uint16, bitLength uint16, isVariableSize bool) ([]byte, uint16, error) {
	rawData := []byte(*d)

	endByteIndex := (bitOffset + bitLength - 1) / 8
	if int(endByteIndex) > len(rawData)-1 {
		if isVariableSize { // variable length caps bit length to packet end so we can read shorter data
			endByteIndex = uint16(len(rawData) - 1)
			bitLength -= (bitOffset + bitLength) - uint16(len(rawData)*8)
		} else {
			return nil, 0, fmt.Errorf("bitoffset is out of bounds of data")
		}
	}

	length := (bitLength + 7) / 8
	result := make([]byte, length)

	startByteIndex := bitOffset / 8
	startBitIndex := bitOffset % 8
	if startByteIndex == endByteIndex { // single byte, everything starts and ends at the same byte
		result[0] = rawData[startByteIndex] >> startBitIndex
		if unnecessaryBits := bitLength % 8; unnecessaryBits != 0 {
			result[0] &= 0xFF >> (8 - unnecessaryBits)
		}
	} else if startBitIndex != 0 { // multibyte, we need to sift bits to get rid of unneeded leading bits
		maskLeading := uint8(0xFF >> startBitIndex)

		result[0] = rawData[startByteIndex] >> startBitIndex
		remainingBits := int(bitLength) - int(startBitIndex)
		for i := uint16(1); i <= length; i++ {
			current := rawData[startByteIndex+i]
			leadingAsTrailing := (current & maskLeading) << startBitIndex
			result[i-1] |= leadingAsTrailing

			remainingBits -= 8
			if remainingBits > 0 {
				result[i] = current >> startBitIndex
			}
		}
	} else { // multibyte, but starts exactly at byte border
		copy(result, rawData[startByteIndex:endByteIndex+1])
		unnecessaryBits := bitLength % 8
		if unnecessaryBits != 0 {
			result[len(result)-1] &= 0xFF >> (8 - unnecessaryBits)
		}
	}

	return result, bitLength, nil
}

func (d *RawData) DecodeVariableUint(bitOffset uint16, bitLength uint16) (uint64, error) {
	return d.decodeVariableInt(bitOffset, bitLength, false)
}

func (d *RawData) DecodeVariableInt(bitOffset uint16, bitLength uint16) (int64, error) {
	variableUInt, err := d.decodeVariableInt(bitOffset, bitLength, true)
	return int64(variableUInt), err
}

func (d *RawData) decodeVariableInt(bitOffset uint16, bitLength uint16, signed bool) (uint64, error) {
	if bitLength > 64 {
		return 0, fmt.Errorf("bit length larger than can be decoded")
	}
	startByteIndex := bitOffset / 8
	endByteIndex := ((bitOffset + bitLength + 7) / 8) - 1
	rawData := []byte(*d)
	if int(endByteIndex) >= len(rawData) {
		return 0, fmt.Errorf("bitoffset is out of bounds of data")
	}

	var result uint64

	rawBytes := make([]byte, 8)
	copy(rawBytes, rawData[startByteIndex:endByteIndex+1])
	result = binary.LittleEndian.Uint64(rawBytes)

	// in case we do not start of the byte then the rightmost bits are what interest us, and we clear leading bits off
	result >>= bitOffset % 8
	mask := (^uint64(0)) >> (64 - bitLength)
	// in case we do not end exactly at the end of last byte, clear those bits at the end
	result = result & mask

	isNegative := false
	if signed {
		// we need to move current most significant bit as uint64 MSB so cast to int64 would have correct sign
		isNegative = result&(1<<(bitLength-1)) != 0 // check if at current bit length MSB is set
		mask = mask >> 1                            // for special value checking
	}

	if bitLength >= 8 { // FIXME: I do not know if these special values (can) work with small bit lengths - does not make real sense
		if result == mask {
			return 0, ErrValueNoData
		} else if result == (mask - 1) {
			return 0, ErrValueOutOfRange
		} else if result == (mask - 2) {
			return 0, ErrValueReserved
		}
	}

	if isNegative {
		// negative numbers have all higher bits toggled
		negativeMask := ^((^uint64(0)) >> (64 - bitLength))
		result |= negativeMask
	}
	return result, nil
}

func (d *RawData) DecodeTime(bitOffset uint16, bitLength uint16, resolution float64) (time.Duration, error) {
	// From Canboat: Absolute times in NMEA2000 are expressed as seconds since midnight(in an undefined timezone)
	rawSeconds, err := d.DecodeVariableUint(bitOffset, bitLength)
	if err != nil {
		return 0, err
	}

	result := time.Duration(uint64(float64(rawSeconds)*resolution)) * time.Second
	if resolution < 1 { // we need to extract decimal parts as smaller than seconds units
		// 1 / resolution => 1 / 0.001 => 1 second is 1000 units (millisecond)
		unitsInSecond := uint64(1 / resolution)
		fraction := rawSeconds % unitsInSecond
		// convert fraction to nanoseconds and then add to result
		result += time.Duration((uint64(time.Second) / unitsInSecond) * fraction)
	}

	return result, nil
}

func (d *RawData) DecodeStringFix(bitOffset uint16, bitLength uint16) (string, error) {
	rawBytes, _, err := d.DecodeBytes(bitOffset, bitLength, false)
	if err != nil {
		return "", err
	}
	length := 0
	for length < len(rawBytes) {
		b := rawBytes[length]
		if b == 0xFF || b == 0x0 || b == '@' {
			break
		}
		length++
	}
	if length == 0 {
		return "", nil
	} else if length == len(rawBytes) {
		return string(rawBytes), nil
	}
	return string(rawBytes[0:length]), nil
}

func (d *RawData) DecodeStringLAU(bitOffset uint16) (string, uint16, error) {
	headerBytes, _, err := d.DecodeBytes(bitOffset, 16, false)
	if err != nil {
		return "", 0, err
	}
	length := uint16(headerBytes[0])
	if length == 2 {
		return "", 16, nil
	} else if length < 2 {
		return "", 0, fmt.Errorf("string lau has invalid size below 2")
	}
	length -= 2 // remove length and encoding bytes size
	encoding := headerBytes[1]
	rawBytes, readBits, err := d.DecodeBytes(bitOffset+16, length*8, true)
	if err != nil {
		return "", 0, err
	}

	readBits += 16 // put len and encoding bits back to report correct read number
	switch encoding {
	case 0: // utf16
		// Credits to: https://gist.github.com/juergenhoetzel/2d9447cdf5c5b30278adfa7e22ec660e
		bom := [2]byte{rawBytes[0], rawBytes[1]}
		var s string
		switch bom {
		case [2]byte{0xff, 0xfe}:
			s, err = decodeUtf16(rawBytes[2:], binary.LittleEndian)
		case [2]byte{0xfe, 0xff}:
			s, err = decodeUtf16(rawBytes[2:], binary.BigEndian)
		default:
			s, err = decodeUtf16(rawBytes, binary.LittleEndian)
		}
		if err != nil {
			return "", 0, err
		}
		return s, readBits, err
	case 1: // utf8/ascii
		// trip trailing 0x0 and 0xFF off. these mean "no data"
		usableBytesLen := 0
		for _, b := range rawBytes {
			if b == 0 || b == 0xFF {
				break
			}
			usableBytesLen++
		}
		if usableBytesLen != len(rawBytes) {
			rawBytes = rawBytes[0:usableBytesLen]
		}
		return string(rawBytes), readBits, nil
	default:
		return "", 0, fmt.Errorf("invalid string lau encoding")
	}
}

func decodeUtf16(b []byte, order binary.ByteOrder) (string, error) {
	ints := make([]uint16, len(b)/2)
	if err := binary.Read(bytes.NewReader(b), order, &ints); err != nil {
		return "", fmt.Errorf("failed to decode utf16 string, err: %w", err)
	}
	return string(utf16.Decode(ints)), nil
}

func (d *RawData) DecodeStringLZ(bitOffset uint16, bitLength uint16) (string, uint16, error) {
	rawData := []byte(*d)
	lengthByteIndex := bitOffset / 8

	actualLength := uint16(rawData[lengthByteIndex])
	fieldLength := (bitLength + 7) / 8
	if actualLength > fieldLength {
		actualLength = fieldLength
	} else if actualLength == 0 {
		return "", 8, nil // empty string
	}

	rawBytes, readBits, err := d.DecodeBytes(bitOffset+8, actualLength*8, true)
	if err != nil {
		return "", 0, err
	}
	return string(rawBytes), readBits, nil
}

func (d *RawData) DecodeDate(bitOffset uint16, bitLength uint16) (time.Time, error) {
	if bitLength != 16 {
		return time.Time{}, fmt.Errorf("can only decode date with 16 bits")
	}
	rawBytes, _, err := d.DecodeBytes(bitOffset, bitLength, false)
	if err != nil {
		return time.Time{}, err
	}
	daysSinceEpoch := binary.LittleEndian.Uint16(rawBytes)

	if daysSinceEpoch == math.MaxUint16 {
		return time.Time{}, ErrValueNoData
	} else if daysSinceEpoch == (math.MaxUint16 - 1) {
		return time.Time{}, ErrValueOutOfRange
	} else if daysSinceEpoch == (math.MaxUint16 - 2) {
		return time.Time{}, ErrValueReserved
	}

	result := epoch.AddDate(0, 0, int(daysSinceEpoch))
	return result, nil
}

func (d *RawData) DecodeDecimal(bitOffset uint16, bitLength uint16) (uint64, error) {
	rawBytes, _, err := d.DecodeBytes(bitOffset, bitLength, false)
	if err != nil {
		return 0, err
	}
	result := uint64(0)
	digits := uint64(1)
	isNoData := true
	for i := len(rawBytes) - 1; i >= 0; i-- {
		b := rawBytes[i]
		if b == 0xff {
			continue
		}
		if b > 99 { // 100+ has 3 digits
			return 0, fmt.Errorf("decimal contains byte with value larger than 2 digits")
		}
		isNoData = false
		right := uint64(b % 10) // right side digit
		left := uint64(b / 10)  // left side digit

		result += digits * right
		digits *= 10
		result += digits * left
		digits *= 10
	}
	if isNoData {
		return 0, ErrValueNoData
	}
	return result, nil
}

func (d *RawData) DecodeFloat(bitOffset uint16, bitLength uint16) (float64, error) {
	if bitLength != 32 {
		return 0.0, fmt.Errorf("can only decode float with 32 bits")
	}
	rawBytes, _, err := d.DecodeBytes(bitOffset, bitLength, false)
	if err != nil {
		return 0., err
	}
	asUint32 := binary.LittleEndian.Uint32(rawBytes)
	frombits := math.Float32frombits(asUint32)

	if asUint32 == math.MaxUint32 { // NaN as float32
		return 0., ErrValueNoData
	} else if asUint32 == (math.MaxUint32 - 1) { // NaN as float32
		return 0., ErrValueOutOfRange
	} else if asUint32 == (math.MaxUint32 - 2) {
		return 0., ErrValueReserved
	}

	return float64(frombits), nil
}

func (d *RawData) AsHex() string {
	if d == nil {
		return ""
	}
	return hex.EncodeToString(*d)
}

type EnumValue struct {
	Value uint32
	Code  string
}
