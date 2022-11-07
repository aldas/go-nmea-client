package canboat

import (
	"errors"
	"fmt"
	"github.com/aldas/go-nmea-client"
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
	Field Field
	Value nmea.FieldValue
}

func (d *Decoder) Decode(raw nmea.RawMessage) (nmea.Message, error) {
	pgn, err := d.findPGN(raw)
	if err != nil {
		return nmea.Message{}, err
	}

	decodedFields := make([]decoded, 0, len(pgn.Fields))
	bitOffset := pgn.Fields[0].BitOffset
	for _, f := range pgn.Fields {

		if f.FieldType == FieldTypeReserved && !d.config.DecodeReservedFields {
			bitOffset += f.BitLength
			continue
		} else if f.FieldType == FieldTypeSpare && !d.config.DecodeSpareFields {
			bitOffset += f.BitLength
			continue
		}

		fv, readBits, err := f.Decode(raw.Data, bitOffset)
		bitOffset += readBits
		if err != nil {
			if err == nmea.ErrValueNoData || err == nmea.ErrValueOutOfRange || err == nmea.ErrValueReserved {
				continue
			}
			return nmea.Message{}, fmt.Errorf("decoder failed to decode field: %v, err: %w", f.ID, err)
		}

		decodedFields = append(decodedFields, decoded{
			Field: f,
			Value: fv,
		})
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

func (d *Decoder) postProcessFields(decodedFields []decoded) (nmea.FieldValues, error) {
	fields := make([]nmea.FieldValue, 0)
	for _, f := range decodedFields {
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
