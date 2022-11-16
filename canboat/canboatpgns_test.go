package canboat

import (
	"encoding/json"
	"github.com/aldas/go-nmea-client"
	test_test "github.com/aldas/go-nmea-client/test"
	"github.com/aldas/go-nmea-client/test/message_test"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPGNs_validate(t *testing.T) {
	//t.SkipNow()

	examplePGNs := test_test.LoadBytes(t, "canboat.json")
	result := CanboatSchema{}
	if err := json.Unmarshal(examplePGNs, &result); err != nil {
		t.Fatal(err)
	}

	errs := result.PGNs.Validate()
	if errs != nil {
		for _, err := range errs {
			assert.NoError(t, err)
		}
	}

	//for _, pgn := range result.PGNs {
	//	nMap := map[string]int8{}
	//	for _, f := range pgn.Fields {
	//		firstSeenOrder, isDuplicate := nMap[f.Name]
	//		if isDuplicate {
	//			fmt.Printf("%v, %v, %v, %v\n", pgn.PGN, f.Name, f.Order, firstSeenOrder)
	//			continue
	//		}
	//		nMap[f.Name] = f.Order
	//	}
	//}
}

func TestPGN_Unmarshal(t *testing.T) {
	var testCases = []struct {
		name        string
		json        []byte
		expect      PGN
		expectError string
	}{
		{
			name: "ok, with EnumBitValues",
			json: test_test.LoadBytes(t, "canboat_pgn_with_field_enumbitvalues.json"),
			expect: PGN{
				PGN:                          0x1f201,
				ID:                           "engineParametersDynamic",
				Description:                  "Engine Parameters, Dynamic",
				Explanation:                  "",
				URL:                          "",
				Type:                         "Fast",
				Complete:                     true,
				FieldCount:                   14,
				MinLength:                    0,
				Length:                       26,
				MissingAttribute:             []string(nil),
				RepeatingFieldSet1Size:       0,
				RepeatingFieldSet1StartField: 0,
				RepeatingFieldSet1CountField: 0,
				RepeatingFieldSet2Size:       0,
				RepeatingFieldSet2StartField: 0,
				RepeatingFieldSet2CountField: 0,
				TransmissionInterval:         500,
				TransmissionIrregular:        false,
				Fields: []Field{
					{
						ID:                   "instance",
						Order:                1,
						Name:                 "Instance",
						Description:          "",
						Condition:            "",
						Match:                0,
						Unit:                 "",
						Format:               "",
						PhysicalQuantity:     "",
						BitLength:            8,
						BitOffset:            0,
						BitLengthVariable:    false,
						Signed:               false,
						Offset:               0,
						Resolution:           1,
						RangeMin:             0,
						RangeMax:             253,
						FieldType:            FieldTypeLookup,
						LookupEnumeration:    "ENGINE_INSTANCE",
						LookupBitEnumeration: "",
					},
					{
						ID:                   "oilPressure",
						Order:                2,
						Name:                 "Oil pressure",
						Description:          "",
						Condition:            "",
						Match:                0,
						Unit:                 "Pa",
						Format:               "",
						PhysicalQuantity:     "PRESSURE",
						BitLength:            16,
						BitOffset:            8,
						BitLengthVariable:    false,
						Signed:               false,
						Offset:               0,
						Resolution:           100,
						RangeMin:             0,
						RangeMax:             6.5533e+06,
						FieldType:            FieldTypeNumber,
						LookupEnumeration:    "",
						LookupBitEnumeration: "",
					},
				},
			},
		},
		{
			name: "ok, with enumvalues",
			json: test_test.LoadBytes(t, "canboat_pgn_with_field_enumvalues.json"),
			expect: PGN{
				PGN:                          0x1f112,
				ID:                           "vesselHeading",
				Description:                  "Vessel Heading",
				Explanation:                  "",
				URL:                          "",
				Type:                         "Single",
				Complete:                     true,
				FieldCount:                   6,
				MinLength:                    0,
				Length:                       8,
				MissingAttribute:             []string(nil),
				RepeatingFieldSet1Size:       0,
				RepeatingFieldSet1StartField: 0,
				RepeatingFieldSet1CountField: 0,
				RepeatingFieldSet2Size:       0,
				RepeatingFieldSet2StartField: 0,
				RepeatingFieldSet2CountField: 0,
				TransmissionInterval:         100,
				TransmissionIrregular:        false,
				Fields: []Field{
					{
						ID:                   "heading",
						Order:                2,
						Name:                 "Heading",
						Description:          "",
						Condition:            "",
						Match:                0,
						Unit:                 "rad",
						Format:               "",
						PhysicalQuantity:     "ANGLE",
						BitLength:            16,
						BitOffset:            8,
						BitLengthVariable:    false,
						Signed:               false,
						Offset:               0,
						Resolution:           0.0001,
						RangeMin:             0,
						RangeMax:             6.5533,
						FieldType:            FieldTypeNumber,
						LookupEnumeration:    "",
						LookupBitEnumeration: "",
					},
					{
						ID:                   "reference",
						Order:                5,
						Name:                 "Reference",
						Description:          "",
						Condition:            "",
						Match:                0,
						Unit:                 "",
						Format:               "",
						PhysicalQuantity:     "",
						BitLength:            2,
						BitOffset:            56,
						BitLengthVariable:    false,
						Signed:               false,
						Offset:               0,
						Resolution:           1,
						RangeMin:             0,
						RangeMax:             2,
						FieldType:            FieldTypeLookup,
						LookupEnumeration:    "DIRECTION_REFERENCE",
						LookupBitEnumeration: "",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := PGN{}
			err := json.Unmarshal(tc.json, &result)

			assert.Equal(t, tc.expect, result)
			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestField_Decode(t *testing.T) {
	var testCases = []struct {
		name           string
		givenRawData   []byte
		when           Field
		expect         nmea.FieldValue
		expectReadBits uint16
		expectError    string
	}{
		{
			name:         "number type decodes to UINT64",
			givenRawData: []uint8{0x3f, 0x9f, 0x14, 0x22, 0xff},
			when: Field{
				ID:         "manufacturerCode",
				Name:       "Manufacturer Code",
				BitLength:  11,
				BitOffset:  0,
				Signed:     false,
				Resolution: 1,
				FieldType:  FieldTypeNumber,
			},
			expect:         nmea.FieldValue{ID: "manufacturerCode", Type: "UINT64", Value: uint64(1855)},
			expectReadBits: 11,
		},
		{
			name:         "number type decodes to INT64",
			givenRawData: []uint8{0x0, 0xff, 0x7f, 0x77, 0xfc, 0xec, 0xf9, 0xff},
			when: Field{
				ID:         "pitch",
				Name:       "Pitch",
				BitLength:  16,
				BitOffset:  24,
				Signed:     true,
				Resolution: 1,
				FieldType:  FieldTypeNumber,
			},
			expect:         nmea.FieldValue{ID: "pitch", Type: "INT64", Value: int64(-905)},
			expectReadBits: 16,
		},
		{
			name:         "number type decodes to FLOAT64",
			givenRawData: []uint8{0x0, 0xff, 0x7f, 0x77, 0xfc, 0xec, 0xf9, 0xff},
			when: Field{
				ID:         "pitch",
				Name:       "Pitch",
				BitLength:  16,
				BitOffset:  24,
				Signed:     true,
				Resolution: 0.0001,
				FieldType:  FieldTypeNumber,
			},
			expect:         nmea.FieldValue{ID: "pitch", Type: "FLOAT64", Value: -0.0905},
			expectReadBits: 16,
		},
		{
			name:         "lookup type decodes to UINT64",
			givenRawData: []uint8{0x3f, 0x9f, 0x14, 0x22, 0xff},
			when: Field{
				ID:         "manufacturerCode",
				Name:       "Manufacturer Code",
				BitLength:  11,
				BitOffset:  0,
				Signed:     false,
				Resolution: 1,
				FieldType:  FieldTypeLookup,
			},
			expect:         nmea.FieldValue{ID: "manufacturerCode", Type: "UINT64", Value: uint64(1855)},
			expectReadBits: 11,
		},
		{
			name:         "reserved type",
			givenRawData: []uint8{0x3f, 0x9f, 0x14, 0x22, 0xff},
			when: Field{
				ID:         "reserved",
				Name:       "Reserved",
				BitLength:  2,
				BitOffset:  11,
				Signed:     false,
				Resolution: 1,
				FieldType:  FieldTypeReserved,
			},
			expect:         nmea.FieldValue{ID: "reserved", Type: "BYTES", Value: []byte{3}},
			expectReadBits: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, readBits, err := tc.when.Decode(tc.givenRawData, tc.when.BitOffset)

			assert.Equal(t, tc.expectReadBits, readBits)
			message_test.AssertFieldValue(t, tc.expect, result, 0.00000_00001)
			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
