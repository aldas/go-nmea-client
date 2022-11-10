package canboat

import (
	"errors"
	"fmt"
	"github.com/aldas/go-nmea-client"
	"math"
)

var (
	ErrDecodeUnknownPGN = errors.New("decode failed, unknown PGN seen")
)

type DecoderConfig struct {
	// DecodeReservedFields instructs Decoder to include reserved type fields in output
	DecodeReservedFields bool
	// DecodeSpareFields instructs Decoder to include spare type fields in output
	DecodeSpareFields bool
	// DecodeLookupsToEnumType instructs Decoder to convert lookup number to actual enum text+value pair
	DecodeLookupsToEnumType bool
}

type Decoder struct {
	config DecoderConfig

	uniquePGNs  map[uint32]PGN
	nonUniqPGNs map[uint32]PGNs

	lookups         LookupEnumerations
	indirectLookups LookupIndirectEnumerations
	bitLookups      LookupBitEnumerations
}

// NewDecoderWithConfig creates new instance of Canboat PGN decoder with given config
func NewDecoderWithConfig(schema CanboatSchema, config DecoderConfig) *Decoder {
	d := NewDecoder(schema)
	d.config = config
	return d
}

// NewDecoder creates new instance of Canboat PGN decoder
func NewDecoder(schema CanboatSchema) *Decoder {
	uniq := map[uint32]PGN{}
	nonUniq := map[uint32]PGNs{}
	for _, pgn := range schema.PGNs {
		existing, ok := uniq[pgn.PGN]
		if !ok {
			uniq[pgn.PGN] = pgn
			continue
		}

		delete(uniq, pgn.PGN)
		group, ok := nonUniq[pgn.PGN]
		if !ok {
			group = PGNs{existing}
		}
		group = append(group, pgn)
		nonUniq[pgn.PGN] = group
	}
	return &Decoder{
		uniquePGNs:  uniq,
		nonUniqPGNs: nonUniq,

		lookups:         schema.Enums,
		indirectLookups: schema.IndirectEnums,
		bitLookups:      schema.BitEnums,
	}
}

type decoded struct {
	Field    Field
	Value    nmea.FieldValue
	ValueSet [][]decoded
}

func (d *Decoder) Decode(raw nmea.RawMessage) (nmea.Message, error) {
	pgn, err := d.findPGN(raw)
	if err != nil {
		return nmea.Message{}, err
	}
	var decodedFields []decoded
	if pgn.RepeatingFieldSet1StartField > 0 || pgn.RepeatingFieldSet2StartField > 0 {
		decodedFields, err = d.decodeWithRepeatedFields(pgn, raw)
	} else {
		decodedFields, err = d.decode(pgn, raw)
	}
	if err != nil {
		return nmea.Message{}, err
	}

	fields, err := d.postProcessFields(decodedFields)
	if err != nil {
		return nmea.Message{}, err
	}

	return nmea.Message{
		Header: raw.Header,
		Fields: fields,
	}, nil
}

var errValueIgnored = errors.New("field value ignored")

func (d *Decoder) decodeSingleField(raw nmea.RawMessage, f Field, bitOffset uint16) (decoded, uint16, error) {
	if (f.FieldType == FieldTypeReserved && !d.config.DecodeReservedFields) ||
		(f.FieldType == FieldTypeSpare && !d.config.DecodeSpareFields) {
		return decoded{}, f.BitLength, errValueIgnored
	}

	fv, readBits, err := f.Decode(raw.Data, bitOffset)
	if err != nil {
		if err == nmea.ErrValueNoData || err == nmea.ErrValueOutOfRange || err == nmea.ErrValueReserved {
			return decoded{}, readBits, errValueIgnored
		}
		return decoded{}, 0, fmt.Errorf("decoder failed to decode field: %v, err: %w", f.ID, err)
	}
	return decoded{
		Field: f,
		Value: fv,
	}, readBits, nil
}

// for the sake of simplicity decoding PGN with repeated fields has different decoding methods as simple PGN
func (d *Decoder) decode(pgn PGN, raw nmea.RawMessage) ([]decoded, error) {
	decodedFields := make([]decoded, 0, len(pgn.Fields))
	messageBitCount := uint16(len(raw.Data) * 8)
	bitOffset := pgn.Fields[0].BitOffset

	// we decode until we reach at the end of the message. This means that some fields may be left out (be optional)
	for i := 0; bitOffset < messageBitCount; i++ {
		if i >= len(pgn.Fields) {
			break
		}
		f := pgn.Fields[i]

		dfv, readBits, err := d.decodeSingleField(raw, f, bitOffset)
		bitOffset += readBits

		if err == errValueIgnored {
			continue
		}
		if err != nil {
			return nil, err
		}
		decodedFields = append(decodedFields, dfv)
	}
	return decodedFields, nil
}

func (d *Decoder) decodeWithRepeatedFields(pgn PGN, raw nmea.RawMessage) ([]decoded, error) {
	decodedFields := make([]decoded, 0, len(pgn.Fields))
	messageBitCount := uint16(len(raw.Data) * 8)
	bitOffset := pgn.Fields[0].BitOffset

	neededRepetitionCountFields := 0
	currentFieldOrder := 1
	currentRepFieldOrder := 0
	currentRepGroupIndex := 0

	var rep1Values [][]decoded
	rep1StartIndex := math.MaxInt // index of first decoded field over all rep groups
	if pgn.RepeatingFieldSet1StartField > 0 {
		rep1StartIndex = int(pgn.RepeatingFieldSet1StartField)
	}
	rep1EndIndex := 0 // index of last decoded field over all rep groups
	if pgn.RepeatingFieldSet1CountField == 0 {
		// Not all PGNs have `RepeatingFieldSet1CountField`. In that case field group repeats till the end of the message (PGN 126464)
		rep1EndIndex = math.MaxInt
		rep1Values = make([][]decoded, 0, 1)
	} else {
		neededRepetitionCountFields++
	}

	var rep2Values [][]decoded
	rep2StartIndex := math.MaxInt // index of first decoded field over all rep groups
	if pgn.RepeatingFieldSet2StartField > 0 {
		rep2StartIndex = int(pgn.RepeatingFieldSet2StartField)
	}
	rep2EndIndex := 0 // index of last decoded field over all rep groups
	if pgn.RepeatingFieldSet2CountField == 0 {
		rep2EndIndex = math.MaxInt
		rep2Values = make([][]decoded, 0, 1)
	} else {
		neededRepetitionCountFields++
	}

	// due to the repeating fields we can not just range over fields. Repeating fields are group of fields that can repeat
	// multiple times in message and the amount of repetitions is determined from specific field value.
	// Note:
	// * Repeating fields are optional, so we break out of decoding loop when we reach at the end of data with our bitOffset
	// * Not all PGNs have `RepeatingFieldSet1CountField`. In that case field group repeats till the end of the message (PGN 126464).
	for i := 0; bitOffset < messageBitCount; i++ {
		if currentFieldOrder > len(pgn.Fields) {
			break
		}
		f := pgn.Fields[currentFieldOrder-1]

		isWithinRep1 := currentFieldOrder >= rep1StartIndex && currentFieldOrder <= rep1EndIndex
		isWithinRep2 := !isWithinRep1 && currentFieldOrder >= rep2StartIndex && currentFieldOrder <= rep2EndIndex
		if isWithinRep1 {
			if currentFieldOrder == rep1StartIndex {
				currentRepFieldOrder = 1
			} else {
				currentRepFieldOrder++
			}
			currentFieldOrder = rep1StartIndex + (currentRepFieldOrder % int(pgn.RepeatingFieldSet1Size))
			currentRepGroupIndex = (currentRepFieldOrder - 1) / int(pgn.RepeatingFieldSet1Size)
		} else if isWithinRep2 {
			if currentFieldOrder == rep2StartIndex {
				currentRepFieldOrder = 1
			} else {
				currentRepFieldOrder++
			}
			currentFieldOrder = rep2StartIndex + (currentRepFieldOrder % int(pgn.RepeatingFieldSet2Size))
			currentRepGroupIndex = (currentRepFieldOrder - 1) / int(pgn.RepeatingFieldSet2Size)
		} else {
			currentFieldOrder++
		}

		dfv, readBits, err := d.decodeSingleField(raw, f, bitOffset)
		bitOffset += readBits

		if err == errValueIgnored {
			continue
		}
		if err != nil {
			return nil, err
		}

		if neededRepetitionCountFields > 0 {
			// when we reach field count field we can calculate end index for that repetition group
			if currentFieldOrder-1 == int(pgn.RepeatingFieldSet1CountField) {
				rep1Count := int(dfv.Value.Value.(uint64))
				rep1Values = make([][]decoded, 0, rep1Count)

				rep1EndIndex = rep1Count*int(pgn.RepeatingFieldSet1Size) + int(pgn.RepeatingFieldSet1StartField)
				neededRepetitionCountFields--
			} else if currentFieldOrder-1 == int(pgn.RepeatingFieldSet2CountField) {
				rep2Count := int(dfv.Value.Value.(uint64))
				rep2Values = make([][]decoded, 0, rep2Count)

				rep2EndIndex = rep2Count*int(pgn.RepeatingFieldSet2Size) + int(pgn.RepeatingFieldSet2StartField)
				neededRepetitionCountFields--
			}
		}

		if isWithinRep1 {
			if currentRepGroupIndex+1 != len(rep1Values) {
				rep1Values = append(rep1Values, make([]decoded, 0, pgn.RepeatingFieldSet1Size))
			}
			grp := rep1Values[currentRepGroupIndex]
			grp = append(grp, dfv)
			rep1Values[currentRepGroupIndex] = grp
		} else if isWithinRep2 {
			if currentRepGroupIndex+1 != len(rep2Values) {
				rep2Values = append(rep2Values, make([]decoded, 0, pgn.RepeatingFieldSet2Size))
			}
			grp := rep2Values[currentRepGroupIndex]
			grp = append(grp, dfv)
			rep2Values[currentRepGroupIndex] = grp
		} else {
			decodedFields = append(decodedFields, dfv)
		}
	}
	if len(rep1Values) > 0 {
		decodedFields = append(decodedFields, decoded{
			Field:    Field{ID: "FIELDSET_1"},
			ValueSet: rep1Values,
		})
	}
	if len(rep2Values) > 0 {
		decodedFields = append(decodedFields, decoded{
			Field:    Field{ID: "FIELDSET_2"},
			ValueSet: rep2Values,
		})
	}

	return decodedFields, nil
}

func (d *Decoder) postProcessFields(decodedFields []decoded) (nmea.FieldValues, error) {
	fields := make([]nmea.FieldValue, 0)
	for _, f := range decodedFields {
		if f.ValueSet != nil {
			fieldsets := make([][]nmea.FieldValue, 0, len(f.ValueSet))
			for _, fs := range f.ValueSet {
				tmp, err := d.postProcessFields(fs)
				if err != nil {
					return nil, err
				}
				fieldsets = append(fieldsets, tmp)
			}
			fields = append(fields, nmea.FieldValue{
				ID:    f.Field.ID,
				Type:  "FIELDSET",
				Value: fieldsets,
			})
			continue
		}
		fv := f.Value
		if d.config.DecodeLookupsToEnumType && (f.Field.FieldType == FieldTypeLookup ||
			f.Field.FieldType == FieldTypeIndirectLookup || f.Field.FieldType == FieldTypeBitLookup) {
			tmpFv, err := d.decodeToEnum(f, decodedFields)
			if err != nil {
				return nil, err
			}
			fv = tmpFv
		}
		fields = append(fields, fv)
	}
	return fields, nil
}

func (d *Decoder) decodeToEnum(df decoded, decodedFields []decoded) (nmea.FieldValue, error) {
	val, ok := df.Value.Value.(uint64)
	if !ok {
		return nmea.FieldValue{}, fmt.Errorf("decoder failed to convert enum value to uint64. field: %v", df.Field.ID)
	}
	f := df.Field
	fv := df.Value
	val32 := uint32(val)

	switch f.FieldType {
	case FieldTypeLookup:
		ev, err := d.lookups.FindValue(f.LookupEnumeration, val32)
		if err == nil {
			fv.Value = nmea.EnumValue{
				Value: ev.Value,
				Code:  ev.Name,
			}
		} else if err == ErrUnknownEnumValue {
			fv.Value = nmea.EnumValue{Value: val32, Code: "UNKNOWN ENUM VALUE"}
		} else {
			return nmea.FieldValue{}, fmt.Errorf("enum field decoding failure, field: %v, err: %w", f.ID, err)
		}
	case FieldTypeBitLookup:
		evBits, err := d.bitLookups.FindValue(f.LookupBitEnumeration, val32)
		if err == nil {
			evs := make([]nmea.EnumValue, 0, len(evBits))
			for _, ev := range evBits {
				evs = append(evs, nmea.EnumValue{
					Value: ev.Bit,
					Code:  ev.Name,
				})
			}
			fv.Value = evs
		} else if err == ErrUnknownEnumValue {
			fv.Value = []nmea.EnumValue{{Value: val32, Code: "UNKNOWN BIT ENUM VALUE"}}
		} else {
			return nmea.FieldValue{}, fmt.Errorf("bit enum field decoding failure, field: %v, err: %w", f.ID, err)
		}

	case FieldTypeIndirectLookup:
		var indirectField decoded
		found := false
		for _, tmpD := range decodedFields {
			if df.Field.LookupIndirectEnumerationFieldOrder == tmpD.Field.Order {
				found = true
				indirectField = tmpD
				break
			}
		}
		if !found {
			return nmea.FieldValue{}, fmt.Errorf("enum field decoding failure, field: %v, could not find indirect field with order: %v", f.ID, df.Field.LookupIndirectEnumerationFieldOrder)
		}
		indirectValue, ok := indirectField.Value.Value.(uint64)
		if !ok {
			return nmea.FieldValue{}, fmt.Errorf("decoder failed to convert indirect enum value to uint64. field: %v", indirectField.Field.ID)
		}

		ev, err := d.indirectLookups.FindValue(f.LookupIndirectEnumeration, val32, uint32(indirectValue))
		if err == nil {
			fv.Value = nmea.EnumValue{
				Value: val32,
				Code:  ev.Name,
			}
		} else if err == ErrUnknownEnumValue {
			fv.Value = nmea.EnumValue{Value: val32, Code: "UNKNOWN INDIRECT ENUM VALUE"}
		} else {
			return nmea.FieldValue{}, fmt.Errorf("indirect enum field decoding failure, field: %v, err: %w", f.ID, err)
		}
	}

	return fv, nil
}

func (d *Decoder) findPGN(raw nmea.RawMessage) (PGN, error) {
	pgn, ok := d.uniquePGNs[raw.Header.PGN]
	if ok {
		return pgn, nil
	}

	pgns, ok := d.nonUniqPGNs[raw.Header.PGN]
	if !ok || len(pgns) == 0 {
		return PGN{}, ErrDecodeUnknownPGN
	}
	pgn, ok = pgns.Match(raw.Data)
	if !ok {
		return PGN{}, ErrDecodeUnknownPGN
	}
	return pgn, nil
}
