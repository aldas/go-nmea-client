package canboat

import (
	"encoding/json"
	"fmt"
	test_test "github.com/aldas/go-nmea-client/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPGNs_Unmarshal_CanBoatSchema(t *testing.T) {
	//t.SkipNow()

	examplePGNs := test_test.LoadBytes(t, "pgns.json")
	result := CanboatSchema{}

	err := json.Unmarshal(examplePGNs, &result)
	assert.NoError(t, err)

	//x, _ := result.PGNs.FindByPGN(127250)

	fmt.Printf("pgn;id;bitlen;type;\n")
	lens := map[uint16]struct{}{}
	for _, pgn := range result.PGNs {
		for _, f := range pgn.Fields {
			if f.Type == FieldTypeUnknownReal && f.BitLength == 0 {
				fmt.Printf("%v;%v;%v;%v;\n", pgn.PGN, f.ID, f.BitLength, f.Type)
				lens[f.BitLength] = struct{}{}
			}
		}
	}
	fmt.Printf("lens-------------\n")
	for k := range lens {
		fmt.Printf("%v\n", k)
	}
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
				PGN:              127489,
				ID:               "engineParametersDynamic",
				Description:      "Engine Parameters, Dynamic",
				Type:             "Fast",
				Complete:         true,
				MissingAttribute: nil,
				RepeatingFields:  0,
				RepeatingFields1: 0,
				RepeatingFields2: 0,
				Length:           26,
				Fields: []Field{
					{
						ID:          "instance",
						Order:       1,
						Name:        "Instance",
						Description: "",
						BitLength:   8,
						BitOffset:   0,
						BitStart:    0,
						Match:       0,
						Units:       "",
						Type:        FieldTypeEnumValue,
						Resolution:  0,
						Signed:      false,
						Offset:      0,
						EnumValues: []EnumValue{
							{Value: 0, Name: "Single Engine or Dual Engine Port"},
							{Value: 1, Name: "Dual Engine Starboard"},
						},
						EnumBitValues: nil,
					},
					{
						ID:            "oilPressure",
						Order:         2,
						Name:          "Oil pressure",
						Description:   "",
						BitLength:     16,
						BitOffset:     8,
						BitStart:      0,
						Match:         0,
						Units:         "hPa",
						Type:          FieldTypeUnknownReal, // "Pressure"
						Resolution:    0,
						Signed:        false,
						Offset:        0,
						EnumValues:    nil,
						EnumBitValues: nil,
					},
					{
						ID:          "discreteStatus2",
						Order:       12,
						Name:        "Discrete Status 2",
						Description: "",
						BitLength:   16,
						BitOffset:   176,
						BitStart:    0,
						Match:       0,
						Units:       "",
						Type:        FieldTypeBitValues,
						Resolution:  0,
						Signed:      false,
						Offset:      0,
						EnumValues:  nil,
						EnumBitValues: []EnumBitValue{
							{Bit: 0, Name: "Warning Level 1"},
							{Bit: 1, Name: "Warning Level 2"},
						},
					},
				},
			},
		},
		{
			name: "ok, with enumvalues",
			json: test_test.LoadBytes(t, "canboat_pgn_with_field_enumvalues.json"),
			expect: PGN{
				PGN:              127250,
				ID:               "vesselHeading",
				Description:      "Vessel Heading",
				Type:             "Single",
				Complete:         true,
				MissingAttribute: nil,
				RepeatingFields:  0,
				RepeatingFields1: 0,
				RepeatingFields2: 0,
				Length:           8,
				Fields: []Field{
					{
						ID:            "heading",
						Order:         2,
						Name:          "Heading",
						Description:   "",
						BitLength:     16,
						BitOffset:     8,
						BitStart:      0,
						Match:         0,
						Units:         "rad",
						Type:          FieldTypeUnknownReal,
						Resolution:    0.0001,
						Signed:        false,
						Offset:        0,
						EnumValues:    nil,
						EnumBitValues: nil,
					},
					{
						ID:          "reference",
						Order:       5,
						Name:        "Reference",
						Description: "",
						BitLength:   2,
						BitOffset:   56,
						BitStart:    0,
						Match:       0,
						Units:       "",
						Type:        FieldTypeEnumValue,
						Resolution:  0,
						Signed:      false,
						Offset:      0,
						EnumValues: []EnumValue{
							{Value: 0, Name: "True"},
							{Value: 1, Name: "Magnetic"},
							{Value: 2, Name: "Error"},
							{Value: 3, Name: "Null"},
						},
						EnumBitValues: nil,
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
