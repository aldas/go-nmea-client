package nmea

//
//func isCustomPGNEqual(t *testing.T, expect CustomPGN, when CustomPGN, pgnConfig canboat.PGN) {
//	expectFields := expect.Fields
//	whenFields := when.Fields
//
//	expect.Fields = nil
//	when.Fields = nil
//
//	assert.Equal(t, expect, when)
//
//	assert.Len(t, whenFields, len(expectFields))
//	for i, expectField := range expectFields {
//		whenField := whenFields[i]
//
//		switch expectField.Value.(type) {
//		case float64:
//			assert.InDelta(t, expectField.Value, whenField.Value, float64(pgnConfig.Fields[i].Resolution))
//		default:
//			assert.Equal(t, expectField, whenField)
//		}
//		expectField.Value = nil
//		whenField.Value = nil
//		assert.Equal(t, expectField, whenField)
//	}
//}
//
//func TestParsePGN_127250(t *testing.T) {
//	pgnConf := canboat.PGN{
//		PGN:              127250, // 0x1f112,
//		ID:               "vesselHeading",
//		Description:      "Vessel Heading",
//		Type:             "Single",
//		Complete:         true,
//		MissingAttribute: []string(nil),
//		RepeatingFields:  0,
//		RepeatingFields1: 0,
//		RepeatingFields2: 0,
//		Length:           8,
//		Fields: []canboat.Field{
//			{
//				ID:            "sid",
//				Order:         0x1,
//				Name:          "SID",
//				Description:   "",
//				BitLength:     0x8,
//				BitOffset:     0x0,
//				BitStart:      0x0,
//				Match:         0x0,
//				Units:         "",
//				Type:          canboat.FieldTypeUnknownReal,
//				Resolution:    0,
//				Signed:        false,
//				Offset:        0,
//				EnumValues:    []canboat.EnumValue(nil),
//				EnumBitValues: []canboat.EnumBitValue(nil),
//			},
//			{
//				ID:            "heading",
//				Order:         0x2,
//				Name:          "Heading",
//				Description:   "",
//				BitLength:     16,
//				BitOffset:     8,
//				BitStart:      0,
//				Match:         0,
//				Units:         "rad",
//				Type:          canboat.FieldTypeUnknownReal, // RES_RADIANS
//				Resolution:    0,
//				Signed:        false,
//				Offset:        0,
//				EnumValues:    []canboat.EnumValue(nil),
//				EnumBitValues: []canboat.EnumBitValue(nil),
//			},
//			{
//				ID:            "deviation",
//				Order:         0x3,
//				Name:          "Deviation",
//				Description:   "",
//				BitLength:     16,
//				BitOffset:     24,
//				BitStart:      0,
//				Match:         0,
//				Units:         "rad",
//				Type:          canboat.FieldTypeUnknownReal, // RES_RADIANS
//				Resolution:    0,
//				Signed:        true,
//				Offset:        0,
//				EnumValues:    []canboat.EnumValue(nil),
//				EnumBitValues: []canboat.EnumBitValue(nil),
//			},
//			{
//				ID:            "variation",
//				Order:         0x4,
//				Name:          "Variation",
//				Description:   "",
//				BitLength:     16,
//				BitOffset:     40,
//				BitStart:      0,
//				Match:         0,
//				Units:         "rad",
//				Type:          canboat.FieldTypeUnknownReal, // RES_RADIANS
//				Resolution:    0,
//				Signed:        true,
//				Offset:        0,
//				EnumValues:    []canboat.EnumValue(nil),
//				EnumBitValues: []canboat.EnumBitValue(nil),
//			},
//			{
//				ID:          "reference",
//				Order:       0x5,
//				Name:        "Reference",
//				Description: "",
//				BitLength:   2,
//				BitOffset:   56,
//				BitStart:    0,
//				Match:       0,
//				Units:       "",
//				Type:        canboat.FieldTypeUnknownReal,
//				Resolution:  0,
//				Signed:      false,
//				Offset:      0,
//				EnumValues: []canboat.EnumValue{
//					{Value: 0, Name: "True"},
//					{Value: 1, Name: "Magnetic"},
//					{Value: 2, Name: "Error"},
//					{Value: 3, Name: "Null"}},
//				EnumBitValues: []canboat.EnumBitValue(nil),
//			},
//			{
//				ID:            "reserved",
//				Order:         0x6,
//				Name:          "Reserved",
//				Description:   "Reserved",
//				BitLength:     6,
//				BitOffset:     58,
//				BitStart:      2,
//				Match:         0,
//				Units:         "",
//				Type:          canboat.FieldTypeBinaryData,
//				Resolution:    0,
//				Signed:        false,
//				Offset:        0,
//				EnumValues:    []canboat.EnumValue(nil),
//				EnumBitValues: []canboat.EnumBitValue(nil),
//				Reserved:      true,
//			},
//		},
//	}
//
//	raw := RawMessage{
//		Priority:    0x2,       // 2
//		PGN:         0x1f112,   // 127250
//		Destination: 0xff,      // 255
//		Source:      0x80,      // 128
//		Timestamp:   0x90a3aaf, // 151665327
//		Length:      0x8,       // 8
//		Data: []byte{
//			0x0, 0xfd, 0xe3, 0xff, 0x7f, 0x30, 0x5, 0xfd, 0x41, 0x30,
//			0x39, 0x30, 0x38, 0x30, 0x30, 0x66, 0x64, 0x65, 0x33, 0x66,
//		},
//	}
//	expected := CustomPGN{
//		PGN: 127250, // 0x1f112,
//		Fields: []FieldValue{
//			{ID: "sid", Type: "uint64", Value: uint64(0)},
//			{ID: "heading", Type: "uint64", Value: uint64(58365)},
//			{ID: "variation", Type: "int64", Value: int64(1328)},
//			{ID: "reference", Type: "uint64", Value: uint64(1)},
//		},
//	}
//
//	result, err := ParsePGN(pgnConf, raw)
//
//	assert.NoError(t, err)
//	assert.Equal(t, expected, result)
//}
//
//func TestParsePGN_126992(t *testing.T) {
//	// actisense + canbusjs analyzer output:
//	//2021-05-21T06:45:05.572Z 3 127 255 126992 System Time:  00 F0 50 49 E8 82 7C 0E
//	//{"timestamp":"2021-05-21T06:45:05.572Z","prio":3,"src":127,"dst":255,"pgn":126992,"description":"System Time",
//	// "fields":{"SID":0,"Source":"GPS","Date":"2021.05.21", "Time": "06:45:04.01000"}}
//
//	rawBytes := test_test.LoadBytes(t, "pgn_126992.json")
//	pgnConf := canboat.PGN{}
//	err := json.Unmarshal(rawBytes, &pgnConf)
//	assert.NoError(t, err)
//
//	raw := RawMessage{
//		Priority:    3,
//		PGN:         126992,
//		Destination: 255,
//		Source:      127,
//		Timestamp:   1621579505,
//		Length:      8,
//		Data:        []byte{0x00, 0xF0, 0x50, 0x49, 0xE8, 0x82, 0x7C, 0x0E},
//	}
//	expected := CustomPGN{
//		PGN: 126992, // 0x1f010
//		Fields: []FieldValue{
//			{ID: "sid", Type: "uint64", Value: uint64(0)},    // 0x00
//			{ID: "source", Type: "uint64", Value: uint64(0)}, // 0xF0 4bits => 0 = GPS
//			{ID: "date", Type: "float64", Value: 18768.0},    // 0x49 0x50 => 18768 - days since 1970.01.01
//			{ID: "time", Type: "float64", Value: 24304.100},  // 0x0E 0x7C 0x82 0xE8 => 243041000 * 0.0001 - seconds since day start (06:45:04.1)
//		},
//	}
//
//	result, err := ParsePGN(pgnConf, raw)
//
//	assert.NoError(t, err)
//	isCustomPGNEqual(t, expected, result, pgnConf)
//}
//
//func TestParsePGN_129029(t *testing.T) {
//	// canboatjs equivalent:
//	// {"prio":3,"pgn":129029,"dst":255,"src":127,"timestamp":"2021-05-26T07:36:00.454Z",
//	// "input":["2021-05-26T07:36:00.454Z,3,129029,127,255,43,
//	//				00,55,49,b8,d9,4e,10,80,32,06,4a,71,41,14,08,00,9a,dd,56,f5,9a,1b,03,50,15,17,01,00,00,00,00,12,fc,00,0e,01,9a,01,ac,08,00,00,00"],
//	// "fields":{
//	//		"SID":0,"Date":"2021.05.26",
//	//		"Time":"07:36:00.30000",
//	//		"Latitude":58.21622066666666,
//	//		"Longitude":22.394298499999998,
//	//		"Altitude":18.29,
//	//		"GNSS type":"GPS+GLONASS",
//	//		"Method":"GNSS fix",
//	//		"Integrity":"No integrity checking",
//	//		"Number of SVs":0,
//	//		"HDOP":2.7,
//	//		"PDOP":4.1,
//	//		"Geoidal Separation":22.2,
//	//		"Reference Stations":0,
//	//		"list":[]
//	//		},"description":"GNSS Position Data"}
//
//	rawBytes := test_test.LoadBytes(t, "pgn_129029.json")
//	pgnConf := canboat.PGN{}
//	err := json.Unmarshal(rawBytes, &pgnConf)
//	assert.NoError(t, err)
//
//	raw := RawMessage{
//		Priority:    3,
//		PGN:         129029,
//		Destination: 255,
//		Source:      127,
//		Timestamp:   0x46f1bc0c,
//		Length:      43,
//		Data: []uint8{
//			0x0, 0x55, 0x49, 0xb8, 0xd9, 0x4e, 0x10, 0x80, 0x32, 0x6, // 10 (0-9)
//			0x4a, 0x71, 0x41, 0x14, 0x8, 0x0, 0x9a, 0xdd, 0x56, 0xf5, // 20 (10-19)
//			0x9a, 0x1b, 0x3, 0x50, 0x15, 0x17, 0x1, 0x0, 0x0, 0x0, // 30 (20-29)
//			0x0, 0x12, 0xfc, 0x0, 0xe, 0x1, 0x9a, 0x1, 0xac, 0x8, // 40 (30-39)
//			0x0, 0x0, 0x0, // 43 (40-42)
//		},
//	}
//	expected := CustomPGN{
//		PGN: 129029,
//		Fields: []FieldValue{
//			{ID: "sid", Type: "uint64", Value: uint64(0)},                 // 0x00
//			{ID: "date", Type: "float64", Value: 18773.0},                 // 0x55, 0x49 = days since 1970.01.01
//			{ID: "time", Type: "float64", Value: 27360.300},               // 0xb8, 0xd9, 0x4e, 0x10 = seconds since day start (06:45:04.1)
//			{ID: "latitude", Type: "float64", Value: 58.21622066666666},   // 0x80, 0x32, 0x6, 0x4a, 0x71, 0x41, 0x14, 0x8,
//			{ID: "longitude", Type: "float64", Value: 22.394298499999998}, // 0x0, 0x9a, 0xdd, 0x56, 0xf5, 0x9a, 0x1b, 0x3
//			{ID: "altitude", Type: "float64", Value: 18.29},               // 0x50, 0x15, 0x17, 0x1, 0x0, 0x0, 0x0, 0x0
//			{ID: "gnssType", Type: "uint64", Value: uint64(2)},            // 0x12 4bits
//			{ID: "method", Type: "uint64", Value: uint64(1)},              // 0x12 4bits
//			{ID: "integrity", Type: "uint64", Value: uint64(0)},           // 0xfc 2bits
//			{ID: "numberOfSvs", Type: "uint64", Value: uint64(0)},         // 0x0
//			{ID: "hdop", Type: "float64", Value: 2.7},                     // 0xe, 0x1
//			{ID: "pdop", Type: "float64", Value: 4.1},                     // 0x9a, 0x1
//			{ID: "geoidalSeparation", Type: "float64", Value: 22.2},       // 0xac, 0x8, 0x0, 0x0
//			{ID: "referenceStations", Type: "uint64", Value: uint64(0)},   // 0x0
//			// Note: last 3 fields are not parsed as length=43 and `"RepeatingFields": 3,` in this case
//		},
//	}
//
//	result, err := ParsePGN(pgnConf, raw)
//
//	assert.NoError(t, err)
//	isCustomPGNEqual(t, expected, result, pgnConf)
//}
//
//func TestParsePGN_130827(t *testing.T) {
//	rawBytes := test_test.LoadBytes(t, "pgn_130827.json")
//	pgnConf := canboat.PGN{}
//	err := json.Unmarshal(rawBytes, &pgnConf)
//	assert.NoError(t, err)
//
//	raw := RawMessage{
//		Priority:    0x7,      // 7
//		PGN:         0x1ff0b,  // 130827
//		Destination: 0xff,     // 255
//		Source:      0x8,      // 8
//		Timestamp:   0x2e17af, // 3020719 - actisense ngt1 "timestamp"
//		Length:      0x5,      // 5
//		Data:        []uint8{0x3f, 0x9f, 0x2, 0x0, 0x0},
//	}
//	expected := CustomPGN{
//		PGN: 130827,
//		Fields: []FieldValue{
//			{ID: "manufacturerCode", Type: "uint64", Value: uint64(1855)}, // 1855=furuno
//			{ID: "industryCode", Type: "uint64", Value: uint64(4)},
//			{ID: "a", Type: "uint64", Value: uint64(2)},
//			{ID: "b", Type: "uint64", Value: uint64(0)},
//			{ID: "c", Type: "uint64", Value: uint64(0)},
//		},
//	}
//
//	result, err := ParsePGN(pgnConf, raw)
//
//	assert.NoError(t, err)
//	isCustomPGNEqual(t, expected, result, pgnConf)
//}
//
//func TestParseField(t *testing.T) {
//	// actisense + canbusjs analyzer output:
//	// {"prio":2,"pgn":127250,"dst":255,"src":128,"timestamp":"2021-05-21T08:40:36.988Z",
//	//"fields":{
//	//	"SID":0,
//	//	"Heading":6.0995,
//	//	"Variation":0.1329,
//	//	"Reference":"Magnetic"},
//	//	"description":"Vessel Heading"
//	//	}
//	//"input":["2021-05-21T08:40:36.988Z,2,127250,128,255,8,00,43,ee,ff,7f,31,05,fd"],
//	exampleData := []byte{ // example 127250 message data
//		0x00, 0x43, 0xee, 0xff, 0x7f, 0x31, 0x05, 0xfd,
//	}
//	var testCases = []struct {
//		name        string
//		givenData   []byte
//		whenField   canboat.Field
//		expect      FieldValue
//		expectError string
//	}{
//		{
//			name: "ok, uint8 (sid)",
//			whenField: canboat.Field{
//				ID:        "sid",
//				Type:      canboat.FieldTypeUnknownReal,
//				BitLength: 8,
//				BitOffset: 0,
//				BitStart:  0,
//				Signed:    false,
//			},
//			expect: FieldValue{
//				ID:    "sid",
//				Type:  "uint64",
//				Value: uint64(0),
//			},
//		},
//		{
//			// "Heading":6.0995, <--- "BitLength":16, "BitOffset":8, "BitStart":0, "Units":"rad", "Resolution":"0.0001", "Signed":false
//			name: "ok, uint16",
//			whenField: canboat.Field{
//				ID:         "heading",
//				Type:       canboat.FieldTypeUnknownReal,
//				BitLength:  16,
//				BitOffset:  8,
//				BitStart:   0,
//				Resolution: 0.0001,
//				Signed:     false,
//			},
//			expect: FieldValue{
//				ID:    "heading",
//				Type:  "float64",
//				Value: 6.0995,
//			},
//		},
//		{
//			// "Variation":0.1329, <--- "BitLength":16, "BitOffset":40, "BitStart":0, "Units":"rad", "Resolution":"0.0001", "Signed":true
//			name: "ok, int16",
//			whenField: canboat.Field{
//				ID:         "variation",
//				Type:       canboat.FieldTypeUnknownReal,
//				BitLength:  16,
//				BitOffset:  40,
//				BitStart:   0,
//				Resolution: 0.0001,
//				Signed:     true,
//			},
//			expect: FieldValue{
//				ID:    "variation",
//				Type:  "float64",
//				Value: 0.1329,
//			},
//		},
//		{
//			// "Reference":"Magnetic", <--- "BitLength":2, "BitOffset":56, "BitStart":0, "Type":"Lookup table", "Signed":false,
//			name: "ok, enum",
//			whenField: canboat.Field{
//				ID:        "reference",
//				Type:      canboat.FieldTypeEnumValue,
//				BitLength: 2,
//				BitOffset: 56,
//				BitStart:  0,
//				Signed:    false,
//				EnumValues: []canboat.EnumValue{
//					{Value: 0, Name: "True"},
//					{Value: 1, Name: "Magnetic"},
//					{Value: 2, Name: "Error"},
//					{Value: 3, Name: "Null"},
//				},
//			},
//			expect: FieldValue{
//				ID:    "reference",
//				Type:  "uint64",
//				Value: uint64(1),
//			},
//		},
//	}
//
//	for _, tc := range testCases {
//		t.Run(tc.name, func(t *testing.T) {
//			raw := exampleData
//			if tc.givenData != nil {
//				raw = tc.givenData
//			}
//
//			result, err := ParseField(tc.whenField, raw)
//
//			if tc.expectError != "" {
//				assert.EqualError(t, err, tc.expectError)
//				assert.Equal(t, tc.expect, result)
//			} else {
//				assert.NoError(t, err)
//
//				switch tc.expect.Value.(type) {
//				case float64:
//					assert.InDelta(t, tc.expect.Value, result.Value, float64(tc.whenField.Resolution))
//				default:
//					assert.Equal(t, tc.expect, result)
//				}
//				assert.Equal(t, tc.expect.ID, result.ID)
//				assert.Equal(t, tc.expect.Type, result.Type)
//			}
//		})
//	}
//}
//
//func TestParseBytesValue(t *testing.T) {
//	var testCases = []struct {
//		name        string
//		givenField  canboat.Field
//		whenData    []byte
//		expect      uint64
//		expectError string
//	}{
//		{
//			name: "ok, multibyte+(start/end)offset",
//			givenField: canboat.Field{
//				BitLength: 11,
//				BitOffset: 1,
//				BitStart:  0,
//				Offset:    0,
//			},
//			whenData: []byte{0b11110001, 0b11001100},
//			// expect: 1111000+1100 = 0b1111000 0b100<<7 = 120 + 1536 = 0b00000110 0b01111000
//			expect: 1656,
//		},
//		{
//			name: "ok, single byte+endoffset",
//			givenField: canboat.Field{
//				BitLength: 2,
//				BitOffset: 56,
//				BitStart:  0,
//			},
//			whenData:    []byte{0x00, 0x43, 0xee, 0xff, 0x7f, 0x31, 0x05, 0xfd},
//			expect:      1,
//			expectError: "",
//		},
//	}
//
//	for _, tc := range testCases {
//		t.Run(tc.name, func(t *testing.T) {
//			result, err := parseBytesValue(tc.givenField, tc.whenData)
//
//			assert.Equal(t, tc.expect, result)
//			if tc.expectError != "" {
//				assert.EqualError(t, err, tc.expectError)
//			} else {
//				assert.NoError(t, err)
//			}
//		})
//	}
//}
