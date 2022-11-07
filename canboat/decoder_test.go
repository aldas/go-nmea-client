package canboat

import (
	"github.com/aldas/go-nmea-client"
	test_test "github.com/aldas/go-nmea-client/test"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func loadPGN(t *testing.T, filename string) *PGN {
	pgn := PGN{}
	test_test.LoadJSON(t, filename, &pgn)
	return &pgn
}

func TestDecoder_Decode(t *testing.T) {
	now := test_test.UTCTime(1665488842) // Tue Oct 11 2022 11:47:22 GMT+0000

	enums := LookupEnumerations{
		{
			Name: "MANUFACTURER_CODE",
			Values: []EnumValue{
				{Name: "Groco", Value: 272},
				{Name: "Actisense", Value: 273},
			},
		},
		{
			Name: "DEVICE_CLASS",
			Values: []EnumValue{
				{Name: "Propulsion", Value: 50},
			},
		},
		{
			Name: "INDUSTRY_CODE",
			Values: []EnumValue{
				{Name: "Marine", Value: 4},
			},
		},
		{
			Name: "ENGINE_INSTANCE",
			Values: []EnumValue{
				{Name: "Single Engine or Dual Engine Port", Value: 0},
				{Name: "Dual Engine Starboard", Value: 1},
			},
		},
	}
	bitEnums := LookupBitEnumerations{
		{
			Name: "ENGINE_STATUS_2",
			Values: []BitEnumValue{
				{Name: "Warning Level 1", Bit: 0},
				{Name: "Warning Level 2", Bit: 1},
			},
		},
		{
			Name: "ENGINE_STATUS_1",
			Values: []BitEnumValue{
				{Name: "Check Engine", Bit: 0},
				{Name: "Over Temperature", Bit: 1},
				{Name: "Low System Voltage", Bit: 5}, //
				{Name: "Low Coolant Level", Bit: 6},
			},
		},
	}
	indirectEnums := LookupIndirectEnumerations{
		{
			Name: "DEVICE_FUNCTION",
			Values: []IndirectEnumValue{
				{Name: "Engine Gateway", IndirectValue: 35, Value: 180},
				{Name: "Engine Gateway", IndirectValue: 50, Value: 160},
			},
		},
	}

	pgn60928 := loadPGN(t, "canboat_pgn_60928.json")
	pgn126998 := loadPGN(t, "canboat_pgn_126998.json")
	pgn130820 := loadPGN(t, "canboat_pgn_130820.json")
	pgn129809 := loadPGN(t, "canboat_pgn_129809.json")
	pgn127506 := loadPGN(t, "canboat_pgn_127506.json")
	pgn127257 := loadPGN(t, "canboat_pgn_127257.json")
	pgns130845 := PGNs{}
	test_test.LoadJSON(t, "canboat_nonuniqpgn_130845.json", &pgns130845)

	var testCases = []struct {
		name        string
		givenPGN    *PGN
		givenConfig DecoderConfig
		whenRaw     nmea.RawMessage
		expect      nmea.Message
		expectError string
	}{
		{
			name: "ok, 127257, Attitude",
			// canboatjs equivalent:
			// echo "2020-08-22T13:52:36.950Z,3,127257,24,255,8,00,fd,7f,44,00,3d,00,ff" | ./bin/analyzerjs
			//{"prio":3,"pgn":127257,"dst":255,"src":24,"timestamp":"2020-08-22T13:52:36.950Z",
			//  "input":["2020-08-22T13:52:36.950Z,3,127257,24,255,8,00,fd,7f,44,00,3d,00,ff"],
			//   "fields":{"SID":0,"Yaw":3.2765,"Pitch":0.0068,"Roll":0.0061},"description":"Attitude"}
			//
			// canboat equivalent:
			// echo "2020-08-22T13:52:36.950Z,3,127257,24,255,8,00,fd,7f,44,00,3d,00,ff" | ./analyzer -json -debug -raw -si
			// {"version":"4.5.2","units":"si","showLookupValues":false}
			//{"timestamp":"2020-08-22T13:52:36.950Z","prio":3,"src":24,"dst":255,"pgn":127257,"description":"Attitude",
			// "fields":{{"SID":0,"bytes"="00"},{"Yaw":3.2765,"bytes"="FD 7F"},{"Pitch":0.0068,"bytes"="44 00"},
			// {"Roll":0.0061,"bytes"="3D 00"}}}
			whenRaw: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    3,
					PGN:         127257,
					Destination: 255,
					Source:      128,
				},
				Data: []uint8{0x0, 0xff, 0x7f, 0x77, 0xfc, 0xec, 0xf9, 0xff},
			},
			expect: nmea.Message{
				Header: nmea.CanBusHeader{
					Priority:    3,
					PGN:         127257,
					Destination: 255,
					Source:      128,
				},
				Fields: nmea.FieldValues{
					{ID: "sid", Type: "UINT64", Value: uint64(0)},  // 0x0
					{ID: "pitch", Type: "FLOAT64", Value: -0.0905}, // 0x77fc => -905 * 0.0001 = -0.0905
					{ID: "roll", Type: "FLOAT64", Value: -0.1556},  // 0xecf9 => -1556 * 0.0001 = -0.1556
				},
			},
		},
		{
			name:        "ok, match from multiple, 130845, Furuno: Multi Sats In View Extended",
			givenConfig: DecoderConfig{DecodeReservedFields: true},
			// canboat equivalent:
			// echo "2021-01-27T23:33:54.347Z,7,130845,1,255,115,3f,9f,14,22,ff,13,1b,57,00,21,57,ab,f8,11,ff,ff,ff,7f,03,58,bc,0a,ca,99,a0,0f,ff,ff,ff,7f,01,84,8b,06,37,dc,cc,10,ff,ff,ff,7f,21,90,60,25,71,10,94,11,ff,ff,ff,7f,23,91,ab,1e,e4,00,00,00,ff,ff,ff,7f,00,98,79,20,40,f5,04,10,ff,ff,ff,7f,00,9d,2d,2c,82,9e,94,11,ff,ff,ff,7f,23,9e,a4,1f,2a,3a,f8,11,ff,ff,ff,7f,01,a4,8c,0a,63,95,74,0e,ff,ff,ff,7f,00" | ./analyzer -json -debug -raw -si
			//{"version":"4.5.2","units":"si","showLookupValues":false}
			//{"timestamp":"2021-01-27T23:33:54.347Z","prio":7,"src":1,"dst":255,"pgn":130845,"description":"Furuno: Multi Sats In View Extended",
			//"fields":{
			//  {"Manufacturer Code":"Furuno","bytes"="3F 07","bits"="11100111111"},
			//  {{"Industry Code":"Marine Industry","bytes"="80","bits"="100"}}}
			whenRaw: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    7,
					PGN:         130845,
					Destination: 255,
					Source:      1,
				},
				Data: []uint8{
					0x3f, 0x9f, 0x14, 0x22, 0xff, 0x13, 0x1b, 0x57, 0x00, 0x21,
					0x57, 0xab, 0xf8, 0x11, 0xff, 0xff, 0xff, 0x7f, 0x03, 0x58,
					0xbc, 0x0a, 0xca, 0x99, 0xa0, 0x0f, 0xff, 0xff, 0xff, 0x7f,
					0x01, 0x84, 0x8b, 0x06, 0x37, 0xdc, 0xcc, 0x10, 0xff, 0xff,
					0xff, 0x7f, 0x21, 0x90, 0x60, 0x25, 0x71, 0x10, 0x94, 0x11,
					0xff, 0xff, 0xff, 0x7f, 0x23, 0x91, 0xab, 0x1e, 0xe4, 0x00,
					0x00, 0x00, 0xff, 0xff, 0xff, 0x7f, 0x00, 0x98, 0x79, 0x20,
					0x40, 0xf5, 0x04, 0x10, 0xff, 0xff, 0xff, 0x7f, 0x00, 0x9d,
					0x2d, 0x2c, 0x82, 0x9e, 0x94, 0x11, 0xff, 0xff, 0xff, 0x7f,
					0x23, 0x9e, 0xa4, 0x1f, 0x2a, 0x3a, 0xf8, 0x11, 0xff, 0xff,
					0xff, 0x7f, 0x01, 0xa4, 0x8c, 0x0a, 0x63, 0x95, 0x74, 0x0e,
					0xff, 0xff, 0xff, 0x7f, 0x00,
				},
			},
			expect: nmea.Message{
				Header: nmea.CanBusHeader{
					Priority:    7,
					PGN:         130845,
					Destination: 255,
					Source:      1,
				},
				Fields: nmea.FieldValues{
					{ID: "manufacturerCode", Type: "UINT64", Value: uint64(1855)},
					{ID: "reserved", Type: "BYTES", Value: []byte{3}},
					{ID: "industryCode", Type: "UINT64", Value: uint64(4)},
				},
			},
		},
		{
			name: "ok, 127506, DC Detailed Status",
			// canboat equivalent:
			// echo "2016-02-28T19:57:03.282Z,6,127506,176,255,9,cd,01,00,64,ff,2e,2b,a9,00" | ./analyzer -json -debug -raw -si
			// {"timestamp":"2016-02-28T19:57:03.282Z","prio":6,"src":176,"dst":255,"pgn":127506,"description":"DC Detailed Status",
			// "fields":{
			//   {"SID":205,"bytes"="CD"},
			//   {"Instance":1,"bytes"="01"},
			//   {"DC Type":"Battery","bytes"="00"},
			//   {"State of Charge":100,"bytes"="64"},
			//   {"State of Health":null,"bytes"="FF"},
			//   { "Time Remaining": "184:14:00","bytes"="2E 2B"},
			//   {"Ripple Voltage":1.69,"bytes"="A9 00"}}}
			whenRaw: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    6,
					PGN:         127506,
					Destination: 255,
					Source:      176,
				},
				Data: []uint8{0xcd, 0x01, 0x00, 0x64, 0xff, 0x2e, 0x2b, 0xa9, 0x00, 0x00, 0x01},
			},
			expect: nmea.Message{
				Header: nmea.CanBusHeader{
					Priority:    6,
					PGN:         127506,
					Destination: 255,
					Source:      176,
				},
				Fields: nmea.FieldValues{
					{ID: "sid", Type: "UINT64", Value: uint64(205)},
					{ID: "instance", Type: "UINT64", Value: uint64(1)},
					{ID: "dcType", Type: "UINT64", Value: uint64(0)},
					{ID: "stateOfCharge", Type: "UINT64", Value: uint64(100)},
					{ID: "timeRemaining", Type: "DURATION", Value: 184*time.Hour + 14*time.Minute},
					{ID: "rippleVoltage", Type: "FLOAT64", Value: 1.69},
					{ID: "remainingCapacity", Type: "UINT64", Value: uint64(256)},
				},
			},
		},
		{
			name: "ok, pgn with bytes fields",
			givenPGN: &PGN{
				PGN:        65288,
				ID:         "BEPMarine",
				Type:       "fast",
				FieldCount: 0,
				Fields: []Field{
					{
						ID:         "manufacturerCode",
						Name:       "Manufacturer Code",
						BitLength:  11,
						BitOffset:  0,
						Signed:     false,
						Resolution: 1,
						FieldType:  FieldTypeNumber,
					},
					{
						ID:         "reserved",
						BitLength:  5,
						BitOffset:  11,
						Signed:     false,
						Resolution: 1,
						FieldType:  FieldTypeReserved,
					},
					{
						ID:        "data",
						Name:      "binary data",
						BitLength: 16,
						BitOffset: 16,
						Signed:    false,
						FieldType: FieldTypeBinary,
					},
				},
			},
			whenRaw: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    7,
					PGN:         65288,
					Destination: 255,
					Source:      29,
				},
				Data: []uint8{0xcd, 0x01, 0x00, 0x64, 0xff, 0x2e, 0x2b, 0xa9, 0x00, 0x00, 0x01},
			},
			expect: nmea.Message{
				Header: nmea.CanBusHeader{
					Priority:    7,
					PGN:         65288,
					Destination: 255,
					Source:      29,
				},
				Fields: nmea.FieldValues{
					{ID: "manufacturerCode", Type: "UINT64", Value: uint64(461)},
					{ID: "data", Type: "BYTES", Value: []byte{0x0, 0x64}},
				},
			},
		},
		{
			// echo "2022-09-10T12:07:03.527Z,6,129809,23,255,27,18,55,81,97,0e,57,49,54,54,45,20,52,41,41,46,00,ff,ff,ff,ff,ff,ff,ff,ff,ff,e1,ff" | ./rel/linux-x86_64/analyzer -json -debug -raw -si
			// {"timestamp":"2022-09-10T12:07:03.527Z","prio":6,"src":23,"dst":255,"pgn":129809,"description":"AIS Class B static data (msg 24 Part A)",
			// "fields":{
			// {"Message ID":"Static data report","bytes"="18","bits"="011000"},
			// {"Repeat Indicator":"Initial","bytes"="00","bits"="00"},
			// {"User ID":"244810069","bytes"="55 81 97 0E"},
			// {"Name":"WITTE RAAF","bytes"="57 49 54 54 45 20 52 41 41 46 00 FF FF FF FF FF FF FF FF FF"},
			// {"AIS Transceiver information":"Channel B VDL reception","bytes"="01","bits"="00001"},
			// {{"Sequence ID":null,"bytes"="FF"}}}
			name: "ok, 129809 with MSSI and STRING_FIX fields",
			whenRaw: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    6,
					PGN:         129809,
					Destination: 255,
					Source:      23,
				},
				Data: []uint8{
					0x18, 0x55, 0x81, 0x97, 0x0e, 0x57, 0x49, 0x54, 0x54, 0x45,
					0x20, 0x52, 0x41, 0x41, 0x46, 0x00, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xe1, 0xff,
				},
			},
			expect: nmea.Message{
				Header: nmea.CanBusHeader{
					Priority:    6,
					PGN:         129809,
					Destination: 255,
					Source:      23,
				},
				Fields: nmea.FieldValues{
					{ID: "messageId", Type: "UINT64", Value: uint64(24)},
					{ID: "repeatIndicator", Type: "UINT64", Value: uint64(0)},
					{ID: "userId", Type: "UINT64", Value: uint64(244810069)},            // MMSI => 0x5581970E
					{ID: "name", Type: "STRING", Value: "WITTE RAAF"},                   // STRING_FIX
					{ID: "aisTransceiverInformation", Type: "UINT64", Value: uint64(1)}, // 0xe1, 0xff
				},
			},
		},
		{
			// echo "2022-09-10T12:08:37.151Z,6,126998,35,255,42,02,01,02,01,26,01,41,69,72,6d,61,72,20,31,2d,36,30,33,2d,36,37,33,2d,39,35,37,30,20,77,77,77,2e,61,69,72,6d,61,72,2e,63,6f,6d" | ./rel/linux-x86_64/analyzer -json -debug -raw -si
			//{"timestamp":"2022-09-10T12:08:37.151Z","prio":6,"src":35,"dst":255,"pgn":126998,"description":"Configuration Information",
			//  "fields":{
			//   {"Installation Description #1":null,"bytes"="02 01"},
			//   {"Installation Description #2":null,"bytes"="02 01"},
			//   {"Manufacturer Information":"Airmar 1-603-673-9570 www.airmar.com","bytes"="26 01 41 69 72 6D 61 72 20 31 2D 36 30 33 2D 36 37 33 2D 39 35 37 30 20 77 77 77 2E 61 69 72 6D 61 72 2E 63 6F 6D"}}}
			name:     "ok, 126998 with STRING_LAU fields",
			givenPGN: pgn126998,
			whenRaw: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    6,
					PGN:         126998,
					Destination: 255,
					Source:      35,
				},
				Data: []byte{
					0x02, 0x01, 0x02, 0x01, 0x26, 0x01, 0x41, 0x69, 0x72, 0x6d,
					0x61, 0x72, 0x20, 0x31, 0x2d, 0x36, 0x30, 0x33, 0x2d, 0x36,
					0x37, 0x33, 0x2d, 0x39, 0x35, 0x37, 0x30, 0x20, 0x77, 0x77,
					0x77, 0x2e, 0x61, 0x69, 0x72, 0x6d, 0x61, 0x72, 0x2e, 0x63,
					0x6f, 0x6d,
				},
			},
			expect: nmea.Message{
				Header: nmea.CanBusHeader{
					Priority:    6,
					PGN:         126998,
					Destination: 255,
					Source:      35,
				},
				Fields: []nmea.FieldValue{
					{ID: "installationDescription1", Type: "STRING", Value: ""},
					{ID: "installationDescription2", Type: "STRING", Value: ""},
					{ID: "manufacturerInformation", Type: "STRING", Value: "Airmar 1-603-673-9570 www.airmar.com"}, // STRING_LAU
				},
			},
			expectError: "",
		},
		{
			// echo "2020-08-22T13:52:52.054Z,7,130820,49,255,20,a3,99,0b,80,01,02,00,c6,3e,05,c7,08,41,56,52,4f,54,52,4f,53" | ./rel/linux-x86_64/analyzer -json -debug -raw -si
			//{"timestamp":"2020-08-22T13:52:52.054Z","prio":7,"src":49,"dst":255,"pgn":130820,"description":"Fusion: AM/FM Station",
			// "fields":{
			// {"Manufacturer Code":"Fusion Electronics","bytes"="A3 01","bits"="00110100011"},
			// {{"Industry Code":"Marine Industry","bytes"="80","bits"="100"},
			// {"Message ID":"11","bytes"="0B"},
			// {"A":128,"bytes"="80"},
			// {"AM/FM":"FM","bytes"="01"},
			// {"B":2,"bytes"="02"},
			// {"Frequency":88000000,"bytes"="00 C6 3E 05"},
			// {"C":199,"bytes"="C7"},
			// {"Track":"AVROTROS","bytes"="08 41 56 52 4F 54 52 4F 53"}}}
			name:     "ok, PGN 130820 with STRINGLZ field",
			givenPGN: pgn130820,
			whenRaw: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    7,
					PGN:         130820,
					Destination: 255,
					Source:      49,
				},
				Data: []byte{
					0xa3, 0x99, 0x0b, 0x80, 0x01, 0x02, 0x00, 0xc6, 0x3e, 0x05,
					0xc7, 0x08, 0x41, 0x56, 0x52, 0x4f, 0x54, 0x52, 0x4f, 0x53,
				},
			},
			expect: nmea.Message{
				Header: nmea.CanBusHeader{
					Priority:    7,
					PGN:         130820,
					Destination: 255,
					Source:      49,
				},
				Fields: []nmea.FieldValue{
					{ID: "manufacturerCode", Type: "UINT64", Value: uint64(419)},
					{ID: "industryCode", Type: "UINT64", Value: uint64(4)},
					{ID: "messageId", Type: "UINT64", Value: uint64(11)},
					{ID: "a", Type: "UINT64", Value: uint64(128)},
					{ID: "amFm", Type: "UINT64", Value: uint64(1)},
					{ID: "b", Type: "UINT64", Value: uint64(2)},
					{ID: "frequency", Type: "UINT64", Value: uint64(88000000)},
					{ID: "c", Type: "UINT64", Value: uint64(199)},
					{ID: "track", Type: "STRING", Value: "AVROTROS"},
				},
			},
			expectError: "",
		},
		{
			// echo "2016-04-09T16:41:29.748Z,6,60928,2,255,8,ae,86,22,22,00,82,32,c0" | ./rel/linux-x86_64/analyzer -json -debug -raw -si{"version":"4.6.1","units":"si","showLookupValues":false}
			//{"timestamp":"2016-04-09T16:41:29.748Z","prio":6,"src":2,"dst":255,"pgn":60928,"description":"ISO Address Claim",
			//"fields":{
			//  {"Unique Number":165550,"bytes"="AE 86 02","bits"="010110101011110101110"},
			//  {"Manufacturer Code":"Actisense","bytes"="20 22","bits"="00000010001"},
			//  {"Device Instance Lower":0,"bytes"="00","bits"="000"},
			//  {"Device Instance Upper":0,"bytes"="00","bits"="00000"},
			//  {"Device Function":"PC Gateway","bytes"="82"},
			//  {{"Device Class":"Internetwork device","bytes"="32","bits"="0011001"},
			//  {"System Instance":0,"bytes"="00","bits"="0000"},
			//  {"Industry Group":"Marine","bytes"="40","bits"="100"},{}}
			name:        "ok, PGN 60928 with SPARE and INDIRECT_LOOKUP field",
			givenPGN:    pgn60928,
			givenConfig: DecoderConfig{DecodeSpareFields: true},
			whenRaw: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    6,
					PGN:         60928,
					Destination: 255,
					Source:      2,
				},
				Data: []byte{
					0xae, 0x86, 0x22, 0x22, 0x00, 0x82, 0x32, 0xc0,
				},
			},
			expect: nmea.Message{
				Header: nmea.CanBusHeader{
					Priority:    6,
					PGN:         60928,
					Destination: 255,
					Source:      2,
				},
				Fields: []nmea.FieldValue{
					{ID: "uniqueNumber", Type: "UINT64", Value: uint64(165550)},
					{ID: "manufacturerCode", Type: "UINT64", Value: uint64(273)},
					{ID: "deviceInstanceLower", Type: "UINT64", Value: uint64(0)},
					{ID: "deviceInstanceUpper", Type: "UINT64", Value: uint64(0)},
					{ID: "deviceFunction", Type: "UINT64", Value: uint64(130)},
					{ID: "spare", Type: "BYTES", Value: []byte{0x0}},
					{ID: "deviceClass", Type: "UINT64", Value: uint64(25)},
					{ID: "systemInstance", Type: "UINT64", Value: uint64(0)},
					{ID: "industryGroup", Type: "UINT64", Value: uint64(4)},
				},
			},
			expectError: "",
		},
		{
			// echo "2016-04-09T16:41:18.104Z,6,60928,16,255,8,99,ad,22,22,00,a0,64,c0" | ./rel/linux-x86_64/analyzer -json -debug -raw -si{"version":"4.7.0","units":"si","showLookupValues":false}
			//{"timestamp":"2016-04-09T16:41:18.104Z","prio":6,"src":16,"dst":255,"pgn":60928,"description":"ISO Address Claim",
			//"fields":{
			//  "Unique Number":{"value":175513,"bytes":"99 AD 02","bits":"001101100110010011001"},
			//  "Manufacturer Code":{"value":"Actisense","bytes":"20 22","bits":"00000010001"},
			//  "Device Instance Lower":{"value":0,"bytes":"00","bits":"000"},
			//  "Device Instance Upper":{"value":0,"bytes":"00","bits":"00000"},
			//  "Device Function":{"value":"Engine Gateway","bytes":"A0"},
			//  "Device Class":{"value":"Propulsion","bytes":"64","bits":"0110010"},
			//  "System Instance":{"value":0,"bytes":"00","bits":"0000"},
			//  "Industry Group":{"value":"Marine","bytes":"40","bits":"100"}}}
			name:        "ok, PGN 60928 with INDIRECT_LOOKUP field converted to enum",
			givenPGN:    pgn60928,
			givenConfig: DecoderConfig{DecodeLookupsToEnumType: true},
			whenRaw: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    6,
					PGN:         60928,
					Destination: 255,
					Source:      16,
				},
				Data: []byte{
					0x99, 0xad, 0x22, 0x22, 0x00, 0xa0, 0x64, 0xc0,
				},
			},
			expect: nmea.Message{
				Header: nmea.CanBusHeader{
					Priority:    6,
					PGN:         60928,
					Destination: 255,
					Source:      16,
				},
				Fields: []nmea.FieldValue{
					{ID: "uniqueNumber", Type: "UINT64", Value: uint64(175513)},
					{ID: "manufacturerCode", Type: "UINT64", Value: nmea.EnumValue{Value: 273, Code: "Actisense"}},
					{ID: "deviceInstanceLower", Type: "UINT64", Value: uint64(0)},
					{ID: "deviceInstanceUpper", Type: "UINT64", Value: uint64(0)},
					{ID: "deviceFunction", Type: "UINT64", Value: nmea.EnumValue{Value: 160, Code: "Engine Gateway"}}, // indirect lookup
					{ID: "deviceClass", Type: "UINT64", Value: nmea.EnumValue{Value: 50, Code: "Propulsion"}},
					{ID: "systemInstance", Type: "UINT64", Value: uint64(0)},
					{ID: "industryGroup", Type: "UINT64", Value: nmea.EnumValue{Value: 4, Code: "Marine"}},
				},
			},
			expectError: "",
		},
		{
			// echo "2022-09-23T11:05:05.383Z,2,127489,236,255,26,00,28,00,ff,ff,bb,71,57,03,00,00,e0,b0,05,00,ff,ff,ff,ff,ff,20,00,00,00,7e,ff" | ./rel/linux-x86_64/analyzer -json -debug -raw -si
			// {"timestamp":"2022-09-23T11:05:05.383Z","prio":2,"src":236,"dst":255,"pgn":127489,"description":"Engine Parameters, Dynamic",
			//"fields":{
			//  "Instance":{"value":"Single Engine or Dual Engine Port","bytes":"00"},
			//  "Oil pressure":{"value":4000,"bytes":"28 00"},
			//  "Oil temperature":{"value":null,"bytes":"FF FF"},
			//  "Temperature":{"value":291.15,"bytes":"BB 71"},
			//  "Alternator Potential":{"value":8.55,"bytes":"57 03"},
			//  "Fuel Rate":{"value":0.0,"bytes":"00 00"},
			//  "Total Engine hours":{"value":"103:36:00","bytes":"E0 B0 05 00"},
			//  "Coolant Pressure":{"value":null,"bytes":"FF FF"},
			//  "Fuel Pressure":{"value":null,"bytes":"FF FF"},
			//  "Discrete Status 1":{"value":["Low System Voltage"],"bytes":"20 00"},
			//  "Discrete Status 2":{"value":null,"bytes":"00 00"},
			//  "Engine Load":{"value":null,"bytes":"7E"},
			//  "Engine Torque":{"value":-1,"bytes":"FF"}}}
			name:     "ok, PGN 127489 with BIT_LOOKUP field",
			givenPGN: loadPGN(t, "canboat_pgn_127489.json"),
			whenRaw: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    2,
					PGN:         127489,
					Destination: 255,
					Source:      236,
				},
				Data: []byte{
					0x00, 0x28, 0x00, 0xff, 0xff, 0xbb, 0x71, 0x57, 0x03, 0x00,
					0x00, 0xe0, 0xb0, 0x05, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff,
					0x20, 0x00, 0x00, 0x00, 0x7e, 0xff,
				},
			},
			expect: nmea.Message{
				Header: nmea.CanBusHeader{
					Priority:    2,
					PGN:         127489,
					Destination: 255,
					Source:      236,
				},
				Fields: []nmea.FieldValue{
					{ID: "instance", Type: "UINT64", Value: uint64(0)},
					{ID: "oilPressure", Type: "FLOAT64", Value: float64(4000)},
					{ID: "temperature", Type: "FLOAT64", Value: float64(291.15)},
					{ID: "alternatorPotential", Type: "FLOAT64", Value: float64(8.55)},
					{ID: "fuelRate", Type: "FLOAT64", Value: float64(0)},
					{ID: "totalEngineHours", Type: "DURATION", Value: 103*time.Hour + 36*time.Minute},
					{ID: "discreteStatus1", Type: "UINT64", Value: uint64(32)}, // BITLOOKUP field
					{ID: "discreteStatus2", Type: "UINT64", Value: uint64(0)},  // BITLOOKUP field
					{ID: "engineTorque", Type: "INT64", Value: int64(-1)},
				},
			},
			expectError: "",
		},
		{
			name:        "ok, PGN 127489 with BIT_LOOKUP field and decode lookups",
			givenConfig: DecoderConfig{DecodeLookupsToEnumType: true},
			givenPGN:    loadPGN(t, "canboat_pgn_127489.json"),
			whenRaw: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    2,
					PGN:         127489,
					Destination: 255,
					Source:      236,
				},
				Data: []byte{
					0x00, 0x28, 0x00, 0xff, 0xff, 0xbb, 0x71, 0x57, 0x03, 0x00,
					0x00, 0xe0, 0xb0, 0x05, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff,
					0x20, 0x00, 0x00, 0x00, 0x7e, 0xff,
				},
			},
			expect: nmea.Message{
				Header: nmea.CanBusHeader{
					Priority:    2,
					PGN:         127489,
					Destination: 255,
					Source:      236,
				},
				Fields: []nmea.FieldValue{
					{ID: "instance", Type: "UINT64", Value: nmea.EnumValue{Value: 0x0, Code: "Single Engine or Dual Engine Port"}},
					{ID: "oilPressure", Type: "FLOAT64", Value: float64(4000)},
					{ID: "temperature", Type: "FLOAT64", Value: float64(291.15)},
					{ID: "alternatorPotential", Type: "FLOAT64", Value: float64(8.55)},
					{ID: "fuelRate", Type: "FLOAT64", Value: float64(0)},
					{ID: "totalEngineHours", Type: "DURATION", Value: 103*time.Hour + 36*time.Minute},
					{ID: "discreteStatus1", Type: "UINT64", Value: []nmea.EnumValue{
						{Value: 5, Code: "Low System Voltage"},
					}}, // BITLOOKUP field
					{ID: "discreteStatus2", Type: "UINT64", Value: []nmea.EnumValue{}}, // BITLOOKUP field
					{ID: "engineTorque", Type: "INT64", Value: int64(-1)},
				},
			},
			expectError: "",
		},
		{
			// echo "2022-04-17-04:35:34.254,4,129808,3,255,83,70,70,33,14,00,5f,1e,6e,64,ff,ff,ff,ff,ff,ff,ff,ff,ff,ff,ff,ff,12,01,ff,ff,ff,ff,ff,ff,ff,ff,ff,ff,ff,ff,ff,ff,ff,ff,70,69,80,e7,77,b1,3a,68,c0,a6,c1,04,ff,ff,ff,ff,ff,7f,fd,ff,ff,ff,ff,ff,ff,ff,ff,ff,ff,ff,ff,ff,ff,ff,ff,ff,ff,ff,ff,64,0a,01,30,38" | ./rel/linux-x86_64/analyzer -json -debug -raw -si
			//{"timestamp":"2022-04-17-04:35:34.254","prio":4,"src":3,"dst":255,"pgn":129808,"description":"DSC Distress Call Information",
			//"fields":{
			//  "DSC Format":{"value":"Distress","bytes":"70"},
			//  "DSC Category":{"value":112,"bytes":"70"},
			//  "DSC Message Address":{"value":5120009530,"bytes":"33 14 00 5F 1E"},
			//  "Nature of Distress":{"value":"Man overboard","bytes":"6E"},
			//  "Subsequent Communication Mode or 2nd Telecommand":{"value":"No reason given","bytes":"64"},
			//  "Proposed Rx Frequency/Channel":{"value":null,"bytes":"FF FF FF FF FF FF"},
			//  "Proposed Tx Frequency/Channel":{"value":null,"bytes":"FF FF FF FF FF FF"},
			//  "Telephone Number":{"value":null,"bytes":"12 01 FF FF FF FF FF FF FF FF FF FF FF FF FF FF FF FF"},
			//  "Latitude of Vessel Reported":{"value":-41.1014800,"bytes":"70 69 80 E7"},
			//  "Longitude of Vessel Reported":{"value":174.8676983,"bytes":"77 B1 3A 68"},
			//  "Time of Position":{"value":"02:13:00","bytes":"C0 A6 C1 04"},
			//  "MMSI of Ship In Distress":{"value":,"bytes":"FF FF FF FF FF"},
			//  "DSC EOS Symbol":{"value":127,"bytes":"7F"},
			//  "Expansion Enabled":{"value":"Yes","bytes":"01","bits":"01"},
			//  "Calling Rx Frequency/Channel":{"value":null,"bytes":"FF FF FF FF FF FF"},
			//  "Calling Tx Frequency/Channel":{"value":null,"bytes":"FF FF FF FF FF FF"},
			//  "Time of Receipt":{"value":null,"bytes":"FF FF FF FF"},
			//  "Date of Receipt":{"value":null,"bytes":"FF FF"},
			//  "DSC Equipment Assigned Message ID":{"value":null,"bytes":"FF FF"},
			//  "list":[{"DSC Expansion Field Symbol":{"value":"Enhanced position","bytes":"64"},"DSC Expansion Field Data":{"value":"08","bytes":"0A 01 30 38"}}]}}
			name:     "ok, PGN 129808 with DECIMAL field",
			givenPGN: loadPGN(t, "canboat_pgn_129808.json"),
			whenRaw: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    4,
					PGN:         129808,
					Destination: 255,
					Source:      3,
				},
				Data: []byte{
					0x70, 0x70, 0x33, 0x14, 0x00, 0x5f, 0x1e, 0x6e, 0x64, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0x12, 0x01, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x70,
					0x69, 0x80, 0xe7, 0x77, 0xb1, 0x3a, 0x68, 0xc0, 0xa6, 0xc1,
					0x04, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f, 0xfd, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x64, 0x0a,
					0x01, 0x30, 0x38,
				},
			},
			expect: nmea.Message{
				Header: nmea.CanBusHeader{
					Priority:    4,
					PGN:         129808,
					Destination: 255,
					Source:      3,
				},
				Fields: []nmea.FieldValue{
					{ID: "dscFormat", Type: "UINT64", Value: uint64(112)},
					{ID: "dscCategory", Type: "UINT64", Value: uint64(112)},
					{ID: "dscMessageAddress", Type: "UINT64", Value: uint64(5120009530)},
					{ID: "natureOfDistress", Type: "UINT64", Value: uint64(110)},
					{ID: "subsequentCommunicationModeOr2ndTelecommand", Type: "UINT64", Value: uint64(100)},
					{ID: "telephoneNumber", Type: "STRING", Value: ""},
					{ID: "proposedRxFrequencyChannel", Type: "STRING", Value: ""},
					{ID: "proposedTxFrequencyChannel", Type: "STRING", Value: ""},
					{ID: "latitudeOfVesselReported", Type: "FLOAT64", Value: -41.101479999999995},
					{ID: "longitudeOfVesselReported", Type: "FLOAT64", Value: 174.8676983},
					{ID: "timeOfPosition", Type: "DURATION", Value: 2*time.Hour + 13*time.Minute},
					{ID: "dscEosSymbol", Type: "UINT64", Value: uint64(127)},
					{ID: "expansionEnabled", Type: "UINT64", Value: uint64(1)},
					{ID: "callingRxFrequencyChannel", Type: "STRING", Value: ""},
					{ID: "callingTxFrequencyChannel", Type: "STRING", Value: ""},
					{ID: "dscExpansionFieldSymbol", Type: "UINT64", Value: uint64(100)},
					{ID: "dscExpansionFieldData", Type: "STRING", Value: "08"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pgns := PGNs{
				*pgn127506,
				*pgn127257,
				pgns130845[0],
				pgns130845[1],
				*pgn129809,
			}
			if tc.givenPGN != nil {
				pgns = append(pgns, *tc.givenPGN)
			}
			decoder := NewDecoderWithConfig(CanboatSchema{
				PGNs:          pgns,
				Enums:         enums,
				BitEnums:      bitEnums,
				IndirectEnums: indirectEnums,
			}, tc.givenConfig)

			result, err := decoder.Decode(tc.whenRaw)

			test_test.AssertRawMessage(t, tc.expect, result, 0.00000_00001)
			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
