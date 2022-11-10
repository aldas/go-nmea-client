package canboat

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aldas/go-nmea-client"
	"io/fs"
	"strconv"
)

// FieldType is type Canboat type field values
type FieldType string

const (
	// FieldTypeNumber - Binary numbers are little endian. Number fields that use two or three bits use one special
	// encoding, for the maximum value.  When present, this means that the field is not present. Number fields that
	// use four bits or more use two special encodings. The maximum positive value means that the field is not present.
	// The maximum positive value minus 1 means that the field has an error. For instance, a broken sensor.
	// For signed numbers the maximum values are the maximum positive value and that minus 1, not the all-ones bit
	// encoding which is the maximum negative value. https://en.wikipedia.org/wiki/Binary_number
	FieldTypeNumber FieldType = "NUMBER"
	// FieldTypeFloat - 32 bit IEEE-754 floating point number. https://en.wikipedia.org/wiki/IEEE_754
	FieldTypeFloat FieldType = "FLOAT"
	// FieldTypeDecimal - A unsigned numeric value represented with 2 decimal digits per. Each byte represent 2 digits,
	// so 1234 is represented by 2 bytes containing 0x12 and 0x34. A number with an odd number of digits will have 0
	// as the first digit in the first byte. https://en.wikipedia.org/wiki/Binary-coded_decimal
	FieldTypeDecimal FieldType = "DECIMAL"
	// FieldTypeLookup - Number value where each value encodes for a distinct meaning. Each lookup has a
	// LookupEnumeration defining what the possible values mean
	FieldTypeLookup FieldType = "LOOKUP"
	// FieldTypeIndirectLookup - Number value where each value encodes for a distinct meaning but the meaning also
	// depends on the value in another field. Each lookup has a LookupIndirectEnumeration defining what the possible values mean.
	FieldTypeIndirectLookup FieldType = "INDIRECT_LOOKUP"
	// FieldTypeBitLookup - Number value where each bit value encodes for a distinct meaning. Each LookupBit has a
	// LookupBitEnumeration defining what the possible values mean. A bitfield can have any combination of bits set.
	FieldTypeBitLookup FieldType = "BITLOOKUP"
	// FieldTypeTime - time https://en.wikipedia.org/wiki/Time
	FieldTypeTime FieldType = "TIME"
	// FieldTypeDate - The date, in days since 1 January 1970. https://en.wikipedia.org/wiki/Calendar_date
	FieldTypeDate FieldType = "DATE"
	// FieldTypeStringFix - A fixed length string containing single byte codepoints. The length of the string is
	// determined by the PGN field definition. Trailing bytes have been observed as '@', ' ', 0x0 or 0xff.
	FieldTypeStringFix FieldType = "STRING_FIX"
	// FieldTypeStringVar - A varying length string containing single byte codepoints. The length of the string is
	// determined either with a start (0x02) and stop (0x01) byte, or with a starting length byte (> 0x02), or an
	// indication that the string is empty which is encoded by either 0x01 or 0x00 as the first byte.
	FieldTypeStringVar FieldType = "STRING_VAR"
	// FieldTypeStringLz - A varying length string containing single byte codepoints encoded with a length byte and
	// terminating zero. The length of the string is determined by a starting length byte. It also contains a
	// terminating zero byte. The length byte includes the zero byte but not itself.
	FieldTypeStringLz FieldType = "STRING_LZ"
	// FieldTypeStringLAU - A varying length string containing double or single byte codepoints encoded with a length
	// byte and terminating zero. The length of the string is determined by a starting length byte. The 2nd byte
	// contains 0 for UNICODE or 1 for ASCII.
	FieldTypeStringLAU FieldType = "STRING_LAU"
	// FieldTypeBinary - Unspecified content consisting of any number of bits.
	FieldTypeBinary FieldType = "BINARY"
	// FieldTypeReserved - Reserved field. All reserved bits shall be 1
	FieldTypeReserved FieldType = "RESERVED"
	// FieldTypeSpare - Spare field. All spare bits shall be 0
	FieldTypeSpare FieldType = "SPARE"
	// FieldTypeMMSI - The MMSI is encoded as a 32 bit number, but is always printed as a 9 digit number and
	// should be considered as a string. The first three or four digits are special, see the USCG link for a detailed
	// explanation.
	FieldTypeMMSI FieldType = "MMSI"
	// FieldTypeVariable - Variable. The definition of the field is that of the reference PGN and reference field,
	// this is totally variable.
	FieldTypeVariable FieldType = "VARIABLE"
)

var (
	ErrUnsupportedFieldType = errors.New("unsupported field type")
)

// UnmarshalJSON custom unmarshalling function for FieldType.
func (bv *FieldType) UnmarshalJSON(b []byte) error {
	if b[0] == '"' && b[len(b)-1] == '"' {
		b = b[1 : len(b)-1]
	}
	t := string(b)

	var tmp FieldType
	switch t {
	case string(FieldTypeNumber), string(FieldTypeFloat), string(FieldTypeDecimal), string(FieldTypeLookup),
		string(FieldTypeIndirectLookup),
		string(FieldTypeBitLookup), string(FieldTypeTime), string(FieldTypeDate), string(FieldTypeStringFix),
		string(FieldTypeStringVar), string(FieldTypeStringLz), string(FieldTypeStringLAU), string(FieldTypeBinary),
		string(FieldTypeReserved), string(FieldTypeSpare), string(FieldTypeMMSI),
		string(FieldTypeVariable):
		tmp = FieldType(t)
	default:
		return fmt.Errorf("unknown FieldType value: `%v`", t)
	}
	*bv = tmp
	return nil
}

type PacketType string

const (
	PacketTypeISO    PacketType = "ISO"  // including multi-packet messages send with ISO 11783-3 Transport Protocol
	PacketTypeFast   PacketType = "Fast" // can have up to 223 bytes of payload data
	PacketTypeSingle PacketType = "Single"
)

// UnmarshalJSON custom unmarshalling function for PacketType.
func (pt *PacketType) UnmarshalJSON(b []byte) error {
	if b[0] == '"' && b[len(b)-1] == '"' {
		b = b[1 : len(b)-1]
	}
	t := string(b)

	var tmp PacketType
	switch t {
	case string(PacketTypeISO), string(PacketTypeFast), string(PacketTypeSingle):
		tmp = PacketType(t)
	default:
		return fmt.Errorf("unknown PacketType value: `%v`", t)
	}
	*pt = tmp
	return nil
}

// CanboatSchema is root element for Canboat Json schema
type CanboatSchema struct {
	Comment       string                     `json:"Comment"`
	CreatorCode   string                     `json:"CreatorCode"`
	License       string                     `json:"License"`
	Version       string                     `json:"Version"`
	PGNs          PGNs                       `json:"PGNs"`
	Enums         LookupEnumerations         `json:"LookupEnumerations"`
	IndirectEnums LookupIndirectEnumerations `json:"LookupIndirectEnumerations"`
	BitEnums      LookupBitEnumerations      `json:"LookupBitEnumerations"`
}

// LoadCANBoatSchema loads CANBoat PGN schema from JSON file
func LoadCANBoatSchema(filesystem fs.FS, path string) (CanboatSchema, error) {
	f, err := filesystem.Open(path)
	if err != nil {
		return CanboatSchema{}, err
	}
	defer func() {
		err = f.Close()
	}()

	schema := CanboatSchema{}
	if err := json.NewDecoder(f).Decode(&schema); err != nil {
		return CanboatSchema{}, err
	}
	return schema, err
}

// PGNs is list of PNG instances
type PGNs []PGN

// PGN is Parameter Group Number. A PGN identifies a message's function and how its data is structured.
type PGN struct {
	// Note: PGN is not unique. Some PGNs have multiple different packets (field sets). pgn+first-field-value is sometimes unique
	PGN              uint32     `json:"PGN"`
	ID               string     `json:"Id"`
	Description      string     `json:"Description"`
	Explanation      string     `json:"Explanation"`
	URL              string     `json:"URL"`
	Type             PacketType `json:"Type"`     // ISO, Fast, Single
	Complete         bool       `json:"Complete"` // false if Canboat schema is incomplete
	FieldCount       int16      `json:"FieldCount"`
	MinLength        int16      `json:"MinLength"`
	Length           int16      `json:"Length"`
	MissingAttribute []string   `json:"Missing"` // Fields, FieldLengths, Precision, Lookups, SampleData

	// RepeatingFields is number of fields that may or may not exist at the end of fields list.
	RepeatingFieldSet1Size       int8 `json:"RepeatingFieldSet1Size"`
	RepeatingFieldSet1StartField int8 `json:"RepeatingFieldSet1StartField"`
	RepeatingFieldSet1CountField int8 `json:"RepeatingFieldSet1CountField"`

	RepeatingFieldSet2Size       int8 `json:"RepeatingFieldSet2Size"`
	RepeatingFieldSet2StartField int8 `json:"RepeatingFieldSet2StartField"`
	RepeatingFieldSet2CountField int8 `json:"RepeatingFieldSet2CountField"`

	TransmissionInterval  int16 `json:"TransmissionInterval"`
	TransmissionIrregular bool  `json:"TransmissionIrregular"`

	Fields []Field `json:"Fields"`

	// synthetic fields

	// IsMatchable denotes that PGNs contains Fields that are matchable.
	IsMatchable bool
}

// UnmarshalJSON custom unmarshalling function for Field.
func (p *PGN) UnmarshalJSON(b []byte) error {
	type tmpPGN PGN
	if err := json.Unmarshal(b, (*tmpPGN)(p)); err != nil {
		return err
	}
	for _, f := range p.Fields {
		if f.Match != 0 {
			p.IsMatchable = true
			break
		}
	}
	return nil
}

func (p *PGN) IsMatch(rawData []byte) bool {
	if !p.IsMatchable {
		return false
	}
	for _, f := range p.Fields {
		if f.Match == 0 {
			continue
		}
		if ok := f.IsMatch(rawData); !ok {
			return false
		}
	}
	return true
}

// Field is (possibly) one of many values packed into PGN packet data
type Field struct {
	ID          string `json:"Id"`
	Order       int8   `json:"Order"`
	Name        string `json:"Name"`
	Description string `json:"Description"`

	Condition        string `json:"Condition"`
	Match            int32  `json:"Match"`
	Unit             string `json:"Unit"`
	Format           string `json:"Format"`
	PhysicalQuantity string `json:"PhysicalQuantity"`

	BitLength         uint16  `json:"BitLength"`
	BitOffset         uint16  `json:"BitOffset"`
	BitLengthVariable bool    `json:"BitLengthVariable"`
	Signed            bool    `json:"Signed"`
	Offset            int32   `json:"Offset"`
	Resolution        float64 `json:"Resolution"` // scale factor for parsed value. result = Offset + (parsedValue * Resolution)
	RangeMin          float64 `json:"RangeMin"`
	RangeMax          float64 `json:"RangeMax"`

	FieldType                           FieldType `json:"FieldType"`
	LookupEnumeration                   string    `json:"LookupEnumeration"`
	LookupBitEnumeration                string    `json:"LookupBitEnumeration"`
	LookupIndirectEnumeration           string    `json:"LookupIndirectEnumeration"`
	LookupIndirectEnumerationFieldOrder int8      `json:"LookupIndirectEnumerationFieldOrder"`
}

func (f *Field) Validate() error {
	switch f.FieldType {
	case FieldTypeStringLAU:
		if !f.BitLengthVariable {
			return fmt.Errorf("field id: %v of type STRING_LAU is not BitLengthVariable", f.ID)
		}
		if f.BitLength != 0 || f.BitOffset != 0 {
			return fmt.Errorf("field id: %v should have BitLength=0 and BitOffset=0", f.ID)
		}
	case FieldTypeMMSI:
		if f.BitLength != 32 {
			return fmt.Errorf("field id: %v of type MSSI bit length is not 32 is %v", f.ID, f.BitLength)
		}
	case FieldTypeDate:
		if f.BitLength != 16 {
			return fmt.Errorf("field id: %v of type DATE bit length is not 16 is %v", f.ID, f.BitLength)
		}
	case FieldTypeLookup:
		if f.LookupEnumeration == "" {
			return fmt.Errorf("field id: %v of type %v has empty LookupEnumeration field", f.ID, FieldTypeLookup)
		}
	case FieldTypeIndirectLookup:
		if f.LookupIndirectEnumeration == "" {
			return fmt.Errorf("field id: %v of type %v has empty LookupIndirectEnumeration field", f.ID, FieldTypeIndirectLookup)
		}
	case FieldTypeBitLookup:
		if f.LookupBitEnumeration == "" {
			return fmt.Errorf("field id: %v of type %v has empty LookupBitEnumeration field", f.ID, FieldTypeBitLookup)
		}
		// FIXME: check if enum exists
	}
	return nil
}

func (f *Field) IsMatch(rawData nmea.RawData) bool {
	// we deliberately consider errors here as no match
	value, err := rawData.DecodeVariableUint(f.BitOffset, f.BitLength)
	return err == nil && uint64(f.Match) == value
}

func (f *Field) Decode(rawData nmea.RawData, bitOffset uint16) (nmea.FieldValue, uint16, error) {
	switch f.FieldType {
	case FieldTypeNumber:
		value, err := f.decodeNumber(rawData, bitOffset)
		return value, f.BitLength, err
	case FieldTypeLookup, FieldTypeIndirectLookup, FieldTypeBitLookup:
		// Decoder will convert them to other Enum types if needed
		value, err := f.decodeNumber(rawData, bitOffset)
		return value, f.BitLength, err
	case FieldTypeReserved, FieldTypeSpare, FieldTypeBinary:
		return f.decodeBytes(rawData, bitOffset)
	case FieldTypeTime:
		value, err := f.decodeTime(rawData, bitOffset)
		return value, f.BitLength, err
	case FieldTypeMMSI:
		value, err := f.decodeMMSI(rawData, bitOffset)
		return value, f.BitLength, err
	case FieldTypeStringFix:
		value, err := f.decodeStringFIX(rawData, bitOffset)
		return value, f.BitLength, err
	case FieldTypeStringLz:
		return f.decodeStringLZ(rawData, bitOffset)
	case FieldTypeStringLAU:
		return f.decodeStringLAU(rawData, bitOffset)
	case FieldTypeDate:
		value, err := f.decodeDate(rawData, bitOffset)
		return value, f.BitLength, err
	case FieldTypeDecimal:
		value, err := f.decodeDecimal(rawData, bitOffset)
		return value, f.BitLength, err
	case FieldTypeFloat:
		value, err := f.decodeFloat(rawData, bitOffset)
		return value, f.BitLength, err
	}
	return nmea.FieldValue{}, 0, fmt.Errorf("field type: %v, err: %w", f.FieldType, ErrUnsupportedFieldType)
}

func (f *Field) decodeNumber(rawData nmea.RawData, bitOffset uint16) (nmea.FieldValue, error) {
	var tmpIntValue int64
	var tmpUIntValue uint64
	var err error
	if f.Signed {
		tmpIntValue, err = rawData.DecodeVariableInt(bitOffset, f.BitLength)
	} else {
		tmpUIntValue, err = rawData.DecodeVariableUint(bitOffset, f.BitLength)
	}
	if err != nil {
		return nmea.FieldValue{}, err
	}

	var value interface{}
	if f.Signed {
		tmpIntValue += int64(f.Offset)
		if f.Resolution == 1 {
			return nmea.FieldValue{ID: f.ID, Type: "INT64", Value: tmpIntValue}, nil
		}
		value = float64(tmpIntValue) * f.Resolution
	} else {
		tmpUIntValue += uint64(f.Offset)
		if f.Resolution == 1 {
			return nmea.FieldValue{ID: f.ID, Type: "UINT64", Value: tmpUIntValue}, nil
		}
		value = float64(tmpUIntValue) * f.Resolution
	}
	return nmea.FieldValue{ID: f.ID, Type: "FLOAT64", Value: value}, nil
}

func (f *Field) decodeBytes(rawData nmea.RawData, bitOffset uint16) (nmea.FieldValue, uint16, error) {
	value, bits, err := rawData.DecodeBytes(bitOffset, f.BitLength, f.BitLengthVariable)
	if err != nil {
		return nmea.FieldValue{}, 0, err
	}
	return nmea.FieldValue{
		ID:    f.ID,
		Type:  "BYTES",
		Value: value,
	}, bits, nil
}

func (f *Field) decodeTime(rawData nmea.RawData, bitOffset uint16) (nmea.FieldValue, error) {
	value, err := rawData.DecodeTime(bitOffset, f.BitLength, f.Resolution)
	if err != nil {
		return nmea.FieldValue{}, err
	}
	return nmea.FieldValue{
		ID:    f.ID,
		Type:  "DURATION",
		Value: value,
	}, nil
}

func (f *Field) decodeDate(rawData nmea.RawData, bitOffset uint16) (nmea.FieldValue, error) {
	str, err := rawData.DecodeDate(bitOffset, f.BitLength)
	if err != nil {
		return nmea.FieldValue{}, err
	}
	return nmea.FieldValue{
		ID:    f.ID,
		Type:  "DATE",
		Value: str,
	}, nil
}

func (f *Field) decodeMMSI(rawData nmea.RawData, bitOffset uint16) (nmea.FieldValue, error) {
	mmsi, err := rawData.DecodeVariableUint(bitOffset, f.BitLength)
	if err != nil {
		return nmea.FieldValue{}, err
	}
	// FIXME: should we validate that MMSI is in range of 0 to 999_999_999
	return nmea.FieldValue{
		ID:    f.ID,
		Type:  "UINT64",
		Value: mmsi,
	}, nil
}

func (f *Field) decodeStringFIX(rawData nmea.RawData, bitOffset uint16) (nmea.FieldValue, error) {
	str, err := rawData.DecodeStringFix(bitOffset, f.BitLength)
	if err != nil {
		return nmea.FieldValue{}, err
	}
	return nmea.FieldValue{
		ID:    f.ID,
		Type:  "STRING",
		Value: str,
	}, nil
}

func (f *Field) decodeStringLZ(rawData nmea.RawData, bitOffset uint16) (nmea.FieldValue, uint16, error) {
	str, readBits, err := rawData.DecodeStringLZ(bitOffset, f.BitLength)
	if err != nil {
		return nmea.FieldValue{}, 0, err
	}
	return nmea.FieldValue{
		ID:    f.ID,
		Type:  "STRING",
		Value: str,
	}, readBits, nil
}

func (f *Field) decodeStringLAU(rawData nmea.RawData, bitOffset uint16) (nmea.FieldValue, uint16, error) {
	str, readBits, err := rawData.DecodeStringLAU(bitOffset)
	if err != nil {
		return nmea.FieldValue{}, 0, err
	}
	return nmea.FieldValue{
		ID:    f.ID,
		Type:  "STRING",
		Value: str,
	}, readBits, nil
}

func (f *Field) decodeDecimal(rawData nmea.RawData, bitOffset uint16) (nmea.FieldValue, error) {
	decimal, err := rawData.DecodeDecimal(bitOffset, f.BitLength)
	if err != nil {
		return nmea.FieldValue{}, err
	}
	return nmea.FieldValue{
		ID:    f.ID,
		Type:  "UINT64",
		Value: decimal,
	}, nil
}

func (f *Field) decodeFloat(rawData nmea.RawData, bitOffset uint16) (nmea.FieldValue, error) {
	float, err := rawData.DecodeFloat(bitOffset, f.BitLength)
	if err != nil {
		return nmea.FieldValue{}, err
	}
	return nmea.FieldValue{
		ID:    f.ID,
		Type:  "FLOAT64",
		Value: float,
	}, nil
}

// EnumBitValue is Enum type for Canbus schema.
type EnumBitValue struct {
	Bit  uint8
	Name string
}

// UnmarshalJSON custom unmarshalling function for EnumBitValue.
func (bv *EnumBitValue) UnmarshalJSON(b []byte) error {
	tmp := make(map[string]string)
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}
	for k, v := range tmp {
		tmpBit, err := strconv.ParseUint(k, 10, 16)
		if err != nil {
			return fmt.Errorf("failed to convert EnumBitValue bit value to uint8: %w", err)
		}
		bv.Bit = uint8(tmpBit)
		bv.Name = v
		break
	}
	return nil
}

// FilterByPGN returns list of matching PGN objects that match by PGN value
func (pgns *PGNs) FilterByPGN(pgn uint32) PGNs {
	result := PGNs{}
	for _, p := range *pgns {
		if p.PGN == pgn {
			result = append(result, p)
		}
	}
	return result
}

func (pgns *PGNs) Match(rawData []byte) (PGN, bool) {
	for _, pgn := range *pgns {
		if !pgn.IsMatchable {
			continue
		}
		if ok := pgn.IsMatch(rawData); ok {
			return pgn, true
		}
	}
	return PGN{}, false
}

func (pgns *PGNs) Validate() []error {
	result := make([]error, 0)
	for _, pgn := range *pgns {
		// RULE: field.ID must be unique within PGN
		fields := map[string]Field{}
		for i, f := range pgn.Fields {
			_, ok := fields[f.ID]
			if ok {
				result = append(result, fmt.Errorf("PGN %v has duplicate field ID: %v", pgn.PGN, f.ID))
			}

			if int(pgn.RepeatingFieldSet1CountField) == i+1 && f.FieldType != FieldTypeNumber {
				result = append(result, fmt.Errorf("PGN %v Field ID: %v with non NUMBER type as RepeatingFieldSet1CountField", pgn.PGN, f.ID))
			} else if int(pgn.RepeatingFieldSet2CountField) == i+1 && f.FieldType != FieldTypeNumber {
				result = append(result, fmt.Errorf("PGN %v Field ID: %v with non NUMBER type as RepeatingFieldSet2CountField", pgn.PGN, f.ID))
			}
		}

		if pgn.RepeatingFieldSet1StartField > 0 && pgn.RepeatingFieldSet1StartField <= pgn.RepeatingFieldSet1CountField {
			result = append(result, fmt.Errorf("PGN %v RepeatingFieldSet1StartField is before RepeatingFieldSet1CountField", pgn.PGN))
		} else if pgn.RepeatingFieldSet2StartField > 0 && pgn.RepeatingFieldSet2StartField <= pgn.RepeatingFieldSet2CountField {
			result = append(result, fmt.Errorf("PGN %v RepeatingFieldSet2StartField is before RepeatingFieldSet2CountField", pgn.PGN))
		}

		// Usual field validations
		for _, f := range pgn.Fields {
			if err := f.Validate(); err != nil {
				result = append(result, err)
			}
		}
	}
	if len(result) > 0 {
		return result
	}
	return nil
}
