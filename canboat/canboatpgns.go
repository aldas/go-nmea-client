package canboat

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strconv"
)

// typedef struct
//{
//  char *   name;
//  uint32_t size; /* Size in bits. All fields are contiguous in message; use 'reserved' fields to fill in empty bits. */
//#define LEN_VARIABLE (0)
//  double resolution; /* Either a positive real value or one of the following RES_ special values */
//#define RES_NOTUSED (0)
//#define RES_RADIANS (1e-4)
//#define RES_ROTATION (1e-3 / 32.0)
//#define RES_HIRES_ROTATION (1e-6 / 32.0)
//#define RES_ASCII (-1.0)
//#define RES_LATITUDE (-2.0)
//#define RES_LONGITUDE (-3.0)
//#define RES_DATE (-4.0)
//#define RES_TIME (-5.0)
//#define RES_TEMPERATURE (-6.0)
//#define RES_6BITASCII (-7.0) /* Actually not used in N2K, only in N183 AIS */
//#define RES_INTEGER (-8.0)
//#define RES_LOOKUP (-9.0)
//#define RES_BINARY (-10.0)
//#define RES_MANUFACTURER (-11.0)
//#define RES_STRING (-12.0)
//#define RES_FLOAT (-13.0)
//#define RES_PRESSURE (-14.0)
//#define RES_STRINGLZ (-15.0) /* ASCII string starting with length byte and terminated by zero byte */
//#define RES_STRINGLAU (-16.0) /* ASCII or UNICODE string starting with length byte and ASCII/Unicode byte */
//#define RES_DECIMAL (-17.0)
//#define RES_BITFIELD (-18.0)
//#define RES_TEMPERATURE_HIGH (-19.0)
//#define RES_TEMPERATURE_HIRES (-20.0)
//#define RES_PRESSURE_HIRES (-21.0)
//#define RES_VARIABLE (-22.0)
//#define MAX_RESOLUTION_LOOKUP 22
//
//  bool  hasSign; /* Is the value signed, e.g. has both positive and negative values? */
//  char *units;   /* String containing the 'Dimension' (e.g. s, h, m/s, etc.) unless it starts with , in which
//                  * case it contains a set of lookup values.
//                  */
//  char *  description;
//  int32_t offset;  /* Only used for SAE J1939 values with sign; these are in Offset/Excess-K notation instead
//                    * of two's complement as used by NMEA 2000.
//                    * See http://en.wikipedia.org/wiki/Offset_binary
//                    */
//  char *camelName; /* Filled by C, no need to set in initializers. */
//} Field;

// typedef struct
//{
//  char *     description;
//  uint32_t   pgn;
//  uint16_t   complete;        /* Either PACKET_COMPLETE or bit values set for various unknown items */
//  PacketType type;            /* Single, Fast or ISO11783 */
//  uint32_t   size;            /* (Minimal) size of this PGN. Helps to determine initial malloc */
//  uint32_t   repeatingFields; /* How many fields at the end repeat until the PGN is exhausted? */
//  Field      fieldList[30]; /* Note fixed # of fields; increase if needed. RepeatingFields support means this is enough for now. */
//  uint32_t   fieldCount;    /* Filled by C, no need to set in initializers. */
//  char *     camelDescription; /* Filled by C, no need to set in initializers. */
//  bool       unknownPgn;       /* true = this is a catch-all for unknown PGNs */
//} Pgn;

// FieldType is type Canboat type field values
type FieldType uint8

const (
	// FieldTypeUnknownReal denotes Canboat unknown type that is parsed as real/float.
	FieldTypeUnknownReal FieldType = 0 // canboat: "" empty
	// FieldTypeBitValues denotes Canboat Bit Enum type (bit to enum)
	FieldTypeBitValues FieldType = 1 // canboat: "Bitfield"
	// FieldTypeManufacturerCode denotes Canboat enum type for manufacturer codes
	FieldTypeManufacturerCode FieldType = 2 // canboat: "Manufacturer code"
	// FieldTypeEnumValue denotes Canboat enum type (int to enum)
	FieldTypeEnumValue FieldType = 3 // canboat: "Lookup table"
	// FieldTypeInteger denotes Canboat integer type
	FieldTypeInteger FieldType = 4 // canboat: "Integer"
	// FieldTypeBinaryData denotes Canboat binary data (is raw binary or we do not know actual schema for this block of bytes)
	FieldTypeBinaryData FieldType = 5 // canboat: "Binary data"

	// FieldTypeString is string start/stop byte or starting with len
	// STRING format is <start=0x02> [ <data> ... ] <stop=0x01>
	//                  <len> [ <data> ... ] (with len > 2)
	//                  <stop>                                 zero length data
	//                  <#00>  ???
	FieldTypeString FieldType = 6 // canboat: "String with start/stop byte"

	// FieldTypeStringLAU is ASCII or UNICODE string starting with length and control byte
	// Format is <len> <control> [ <data> ... ]
	// where <control> == 0 = UNICODE, but we don't know whether it is UTF16, UTF8, etc. Not seen in the wild yet!
	//       <control> == 1 = ASCII(?) or maybe UTF8?
	FieldTypeStringLAU FieldType = 7 // canboat: "ASCII or UNICODE string starting with length and control byte"

	// FieldTypeStringLZ is "ASCII string starting with length byte"
	// Format is <len> [ <data> ... ]
	FieldTypeStringLZ FieldType = 8 // canboat: "ASCII string starting with length byte"

	// FieldTypeASCII is a fixed length string (field->size)
	FieldTypeASCII FieldType = 9 // canboat: "ASCII text"

	// FieldTypeNibbleDecimal is number encoded as nibble decimal values.
	// Decimal: 123456789 shall be encoded as 0x0C (=12), 0x22 (=34), 0x38 (=56), 0x4E (=78) and 0x5A (=90).
	// See "PGN 129808 DSC Call Information" https://www.nmea.org/Assets/20130720%20%20dsc%20technical%20corrigendum%20v1..pdf
	// Note: When displaying the MMSI the trailing zero should be removed.
	FieldTypeNibbleDecimal FieldType = 10 // canboat: "Decimal encoded number"
)

// CanboatSchema is root element for Canboat Json schema
type CanboatSchema struct {
	Comment     string `json:"Comment"`
	CreatorCode string `json:"CreatorCode"`
	License     string `json:"License"`
	Version     string `json:"Version"`
	PGNs        PGNs   `json:"PGNs"`
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
	PGN              uint32   `json:"PGN"` // Note: PGN is not unique. Some PGNs have multiple different packets (field sets). pgn+first-field-value is sometimes unique
	ID               string   `json:"Id"`
	Description      string   `json:"Description"`
	Type             string   `json:"Type"`     // ISO, Fast, Single
	Complete         bool     `json:"Complete"` // false if Canboat schema is incomplete
	MissingAttribute []string `json:"Missing"`  // Fields, FieldLengths, Precision, Lookups, SampleData
	// RepeatingFields is number of fields that may or may not exist at the end of fields list.
	RepeatingFields uint32 `json:"RepeatingFields"` // pgn.repeatingFields
	// FIXME: investigate `The last %u and %u fields repeat until the data is exhausted`. Mostly related to PGN=126208, png type of function is defined by first field
	RepeatingFields1 uint32 `json:"RepeatingFieldSet1"` // pgn.repeatingFields % 100
	RepeatingFields2 uint32 `json:"RepeatingFieldSet2"` // pgn.repeatingFields / 100
	// Length is (minimal) size of PGN message.
	Length uint32  `json:"Length"`
	Fields []Field `json:"Fields"`
}

// Field is (possibly) one of many values packed into PGN packet data
type Field struct {
	ID            string         `json:"Id"`
	Order         uint8          `json:"Order"`
	Name          string         `json:"Name"`
	Description   string         `json:"Description"`
	BitLength     uint16         `json:"BitLength"`
	BitOffset     uint16         `json:"BitOffset"`
	BitStart      uint16         `json:"BitStart"`
	Match         uint16         `json:"Match"`
	Units         string         `json:"Units"`
	Type          FieldType      `json:"Type"`
	Resolution    float64        `json:"Resolution"` // 0 do nothing, negative = special cases (not in pgn.json), pos = is scale factor (value * res)
	Signed        bool           `json:"Signed"`
	Offset        int32          `json:"Offset"`
	EnumValues    []EnumValue    `json:"EnumValues"`
	EnumBitValues []EnumBitValue `json:"EnumBitValues"`
	Reserved      bool
}

// UnmarshalJSON custom unmarshalling function for Field.
func (f *Field) UnmarshalJSON(b []byte) error {
	type tmpField Field
	if err := json.Unmarshal(b, (*tmpField)(f)); err != nil {
		return err
	}
	if f.ID == "reserved" {
		f.Reserved = true
	}
	return nil
}

// UnmarshalJSON custom unmarshalling function for FieldType.
func (bv *FieldType) UnmarshalJSON(b []byte) error {
	if b[0] == '"' && b[len(b)-1] == '"' {
		b = b[1 : len(b)-1]
	}
	t := string(b)

	var tmp FieldType
	switch t {
	case "Bitfield":
		tmp = FieldTypeBitValues
	case "Manufacturer code":
		tmp = FieldTypeManufacturerCode
	case "Lookup table":
		tmp = FieldTypeEnumValue
	case "Integer":
		tmp = FieldTypeInteger
	case "Binary data":
		tmp = FieldTypeBinaryData
	case "String with start/stop byte":
		tmp = FieldTypeString
	case "ASCII or UNICODE string starting with length and control byte":
		tmp = FieldTypeStringLAU
	case "ASCII string starting with length byte":
		tmp = FieldTypeStringLZ
	case "ASCII text":
		tmp = FieldTypeASCII
	case "Decimal encoded number":
		tmp = FieldTypeNibbleDecimal

	case "",
		"Latitude",
		"Longitude",
		"Date",
		"Time",
		"Temperature",
		"Temperature (hires)",
		"Pressure",
		"Pressure (hires)",
		"IEEE Float":
		tmp = FieldTypeUnknownReal
	default:
		return errors.New("unknown FieldType value")
	}
	*bv = tmp
	return nil
}

// EnumValue is Enum type for CANbus schema.
type EnumValue struct {
	Value int64  `json:"value"`
	Name  string `json:"name"`
}

// UnmarshalJSON custom unmarshalling function for EnumValue.
func (ev *EnumValue) UnmarshalJSON(b []byte) error {
	tmp := make(map[string]string)
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}
	name, ok := tmp["name"]
	if !ok {
		return fmt.Errorf("missing name field for EnumValue")
	}
	value, ok := tmp["value"]
	if !ok {
		return fmt.Errorf("missing value field for EnumValue")
	}
	tmpBit, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to convert EnumValue value to int64: %w", err)
	}
	ev.Value = tmpBit
	ev.Name = name
	return nil
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

// FindByPGN returns PGN object by its PGN value
func (pgns *PGNs) FindByPGN(pgn uint32) (PGN, bool) {
	for _, p := range *pgns {
		if p.PGN == pgn {
			return p, true
		}
	}
	return PGN{}, false
}
