package actisense

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"github.com/aldas/go-nmea-client"
	test_test "github.com/aldas/go-nmea-client/test"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
	"time"
)

func TestParseRawMessages(t *testing.T) {
	now := time.Unix(1623928400, 0)
	var testCases = []struct {
		name        string
		when        string
		expect      nmea.RawMessage
		expectError string
	}{
		{
			name: "ok, 129025, position rapid update",
			when: "93130201f801ff7faf3a0a0908e715b322c318590dca",
			expect: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    0x2,     // 2
					PGN:         0x1f801, // 129025
					Destination: 0xff,    // 255
					Source:      0x7f,    // 127
				},
				//Timestamp:   0x90a3aaf, // 151665327
				Data: []uint8{0xe7, 0x15, 0xb3, 0x22, 0xc3, 0x18, 0x59, 0xd},
			},
		},
		{
			name: "ok, 127250, vessel heading",
			when: "93130212f101ff80af3a0a090800fde3ff7f3005fd41",
			expect: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    0x2,     // 2
					PGN:         0x1f112, // 127250
					Destination: 0xff,    // 255
					Source:      0x80,    // 128
				},
				//Timestamp:   0x90a3aaf,                                             // 151665327
				Data: []uint8{0x0, 0xfd, 0xe3, 0xff, 0x7f, 0x30, 0x5, 0xfd}, // 00 fd e3 ff 7f 30 05 fd
			},
		},
		{
			name: "ok, 129029, GNSS Position Data",
			when: "93360305f801ff7f083d0a092b004949d8343e0f00463eb928411408a064944bd69a1b03f0d8ffffffffffff12fc003c005a00ac08000000fd",
			expect: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    0x3,     // 3
					PGN:         0x1f805, // 129029
					Destination: 0xff,    // 255
					Source:      0x7f,    // 127
				},
				//Timestamp:   0x90a3d08, // 151665928
				Data: []uint8{
					0x0, 0x49, 0x49, 0xd8, 0x34, 0x3e, 0xf, 0x0, 0x46, 0x3e,
					0xb9, 0x28, 0x41, 0x14, 0x8, 0xa0, 0x64, 0x94, 0x4b, 0xd6,
					0x9a, 0x1b, 0x3, 0xf0, 0xd8, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0x12, 0xfc, 0x0, 0x3c, 0x0, 0x5a, 0x0, 0xac, 0x8,
					0x0, 0x0, 0x0,
				},
			},
		},
		{
			// canboatjs equivalent:
			// {"prio":2,"pgn":129026,"dst":255,"src":127,"timestamp":"2021-05-26T07:35:59.958Z",
			// "input":["2021-05-26T07:35:59.958Z,2,129026,127,255,8,00,fc,ff,ff,00,00,ff,ff"],
			// "fields":{"SID":0,"COG Reference":"True","SOG":0},"description":"COG & SOG, Rapid Update"}
			name: "ok, 129026, COG & SOG, Rapid Update",
			when: "93130202f801ff7f15baf1460800fcffff0000ffffd9",
			expect: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    2,
					PGN:         129026,
					Destination: 255,
					Source:      127,
				},
				//Timestamp:   0x46f1ba15, // 2007-09-20T03:08:53+03:00
				Data: []uint8{0x0, 0xfc, 0xff, 0xff, 0x0, 0x0, 0xff, 0xff},
			},
		},
		{
			// canboatjs equivalent:
			// {"prio":2,"pgn":129025,"dst":255,"src":127,"timestamp":"2021-05-26T07:35:59.962Z",
			// "input":["2021-05-26T07:35:59.962Z,2,129025,127,255,8,1e,17,b3,22,49,19,59,0d"],
			// "fields":{"Latitude":58.2162206,"Longitude":22.3942985},"description":"Position, Rapid Update"}
			name: "ok, 129025, Position, Rapid Update",
			when: "93130201f801ff7f15baf146081e17b3224919590d00",
			expect: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    2,
					PGN:         129025,
					Destination: 255,
					Source:      127,
				},
				//Timestamp:   0x46f1ba15, // 2007-09-20T03:08:53+03:00
				Data: []uint8{0x1e, 0x17, 0xb3, 0x22, 0x49, 0x19, 0x59, 0xd},
			},
		},
		{
			// canboatjs equivalent:
			// {"prio":2,"pgn":127250,"dst":255,"src":128,"timestamp":"2021-05-26T07:35:59.963Z",
			// "input":["2021-05-26T07:35:59.963Z,2,127250,128,255,8,00,bd,ee,ff,7f,31,05,fd"],
			// "fields":{"SID":0,"Heading":6.1117,"Variation":0.1329,"Reference":"Magnetic"},"description":"Vessel Heading"}
			name: "ok, 127250, Vessel Heading",
			when: "93130212f101ff8016baf1460800bdeeff7f3105fd6a",
			expect: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    2,
					PGN:         127250,
					Destination: 255,
					Source:      128,
				},
				//Timestamp:   0x46f1ba16,
				Data: []uint8{0x0, 0xbd, 0xee, 0xff, 0x7f, 0x31, 0x5, 0xfd},
			},
		},
		{
			// canboatjs equivalent:
			// {"prio":3,"pgn":127251,"dst":255,"src":128,"timestamp":"2021-05-26T07:35:59.964Z",
			// "input":["2021-05-26T07:35:59.964Z,3,127251,128,255,8,00,f2,e6,1d,00,00,ff,ff"],
			// "fields":{"SID":0,"Rate":0.0612395625},"description":"Rate of Turn"}
			name: "ok, 127251, Rate of Turn",
			when: "93130313f101ff8017baf1460800f2e61d0000ffffd0",
			expect: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    3,
					PGN:         127251,
					Destination: 255,
					Source:      128,
				},
				//Timestamp:   0x46f1ba17,
				Data: []uint8{0x0, 0xf2, 0xe6, 0x1d, 0x0, 0x0, 0xff, 0xff},
			},
		},
		{
			// canboatjs equivalent:
			// {"prio":3,"pgn":129029,"dst":255,"src":127,"timestamp":"2021-05-26T07:36:00.454Z",
			// "input":["2021-05-26T07:36:00.454Z,3,129029,127,255,43,00,55,49,b8,d9,4e,10,80,32,06,4a,71,41,14,08,00,9a,dd,56,f5,9a,1b,03,50,15,17,01,00,00,00,00,12,fc,00,0e,01,9a,01,ac,08,00,00,00"],
			// "fields":{"SID":0,"Date":"2021.05.26","Time":"07:36:00.30000","Latitude":58.21622066666666,"Longitude":22.394298499999998,"Altitude":18.29,"GNSS type":"GPS+GLONASS","Method":"GNSS fix","Integrity":"No integrity checking","Number of SVs":0,"HDOP":2.7,"PDOP":4.1,"Geoidal Separation":22.2,"Reference Stations":0,"list":[]},"description":"GNSS Position Data"}
			name: "ok, 129029, GNSS Position Data",
			when: "93360305f801ff7f0cbcf1462b005549b8d94e108032064a71411408009add56f59a1b03501517010000000012fc000e019a01ac08000000ce",
			expect: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    3,
					PGN:         129029,
					Destination: 255,
					Source:      127,
				},
				//Timestamp:   0x46f1bc0c,
				Data: []uint8{
					0x0, 0x55, 0x49, 0xb8, 0xd9, 0x4e, 0x10, 0x80, 0x32, 0x6,
					0x4a, 0x71, 0x41, 0x14, 0x8, 0x0, 0x9a, 0xdd, 0x56, 0xf5,
					0x9a, 0x1b, 0x3, 0x50, 0x15, 0x17, 0x1, 0x0, 0x0, 0x0, 0x0,
					0x12, 0xfc, 0x0, 0xe, 0x1, 0x9a, 0x1, 0xac, 0x8, 0x0, 0x0,
					0x0,
				},
			},
		},
		{
			// canboatjs equivalent:
			// {"prio":6,"pgn":129540,"dst":255,"src":127,"timestamp":"2021-05-26T07:36:00.494Z",
			//"input":["2021-05-26T07:36:00.494Z,6,129540,127,255,135,00,ff,0b,02,96,1a,72,50,1c,0c,00,00,00,00,f2,03,d1,06,ae,00,48,0d,00,00,00,00,f2,06,e8,19,c4,31,74,0e,00,00,00,00,f2,0c,39,37,5b,4c,fc,08,00,00,00,00,f2,13,f4,0c,73,23,d8,0e,00,00,00,00,f2,56,d1,06,0b,11,6c,07,00,00,00,00,f2,1d,39,0a,f1,93,6c,07,00,00,00,00,f0,20,c5,13,1f,ba,14,05,00,00,00,00,f0,46,f4,0c,58,f1,6c,07,00,00,00,00,f0,4d,8b,18,cf,51,dc,05,00,00,00,00,f0,57,00,00,00,00,f0,0a,00,00,00,00,f0"],
			//"fields":{"SID":0,"Sats in View":11,"list":[{"PRN":2,"Elevation":0.6806,"Azimuth":2.0594,"SNR":31,"Range residuals":0,"Status":"Used"},{"PRN":3,"Elevation":0.1745,"Azimuth":0.0174,"SNR":34,"Range residuals":0,"Status":"Used"},{"PRN":6,"Elevation":0.6632,"Azimuth":1.274,"SNR":37,"Range residuals":0,"Status":"Used"},{"PRN":12,"Elevation":1.4137,"Azimuth":1.9547,"SNR":23,"Range residuals":0,"Status":"Used"},{"PRN":19,"Elevation":0.3316,"Azimuth":0.9075,"SNR":38,"Range residuals":0,"Status":"Used"},{"PRN":86,"Elevation":0.1745,"Azimuth":0.4363,"SNR":19,"Range residuals":0,"Status":"Used"},{"PRN":29,"Elevation":0.2617,"Azimuth":3.7873,"SNR":19,"Range residuals":0,"Status":"Not tracked"},{"PRN":32,"Elevation":0.5061,"Azimuth":4.7647,"SNR":13,"Range residuals":0,"Status":"Not tracked"},{"PRN":70,"Elevation":0.3316,"Azimuth":6.1784,"SNR":19,"Range residuals":0,"Status":"Not tracked"},{"PRN":77,"Elevation":0.6283,"Azimuth":2.0943,"SNR":15,"Range residuals":0,"Status":"Not tracked"},{"PRN":87,"Elevation":0,"Azimuth":0,"SNR":28,"Range residuals":0,"Status":"Not tracked"}]},"description":"GNSS Sats in View"}
			name: "ok, 129540, GNSS Sats in View",
			when: "93920604fa01ff7f10bcf1468700ff0b02961a72501c0c00000000f203d106ae00480d00000000f206e819c431740e00000000f20c39375b4cfc0800000000f213f40c7323d80e00000000f256d1060b116c0700000000f21d390af1936c0700000000f020c5131fba140500000000f046f40c58f16c0700000000f04d8b18cf51dc0500000000f05700000000f00a00000000f07a",
			expect: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    6,
					PGN:         129540,
					Destination: 255,
					Source:      127,
				},
				//Timestamp:   0x46f1bc10,
				Data: []uint8{
					0x0, 0xff, 0xb, 0x2, 0x96, 0x1a, 0x72, 0x50, 0x1c, 0xc,
					0x0, 0x0, 0x0, 0x0, 0xf2, 0x3, 0xd1, 0x6, 0xae, 0x0, 0x48,
					0xd, 0x0, 0x0, 0x0, 0x0, 0xf2, 0x6, 0xe8, 0x19, 0xc4, 0x31,
					0x74, 0xe, 0x0, 0x0, 0x0, 0x0, 0xf2, 0xc, 0x39, 0x37, 0x5b,
					0x4c, 0xfc, 0x8, 0x0, 0x0, 0x0, 0x0, 0xf2, 0x13, 0xf4, 0xc,
					0x73, 0x23, 0xd8, 0xe, 0x0, 0x0, 0x0, 0x0, 0xf2, 0x56, 0xd1,
					0x6, 0xb, 0x11, 0x6c, 0x7, 0x0, 0x0, 0x0, 0x0, 0xf2, 0x1d,
					0x39, 0xa, 0xf1, 0x93, 0x6c, 0x7, 0x0, 0x0, 0x0, 0x0, 0xf0,
					0x20, 0xc5, 0x13, 0x1f, 0xba, 0x14, 0x5, 0x0, 0x0, 0x0, 0x0,
					0xf0, 0x46, 0xf4, 0xc, 0x58, 0xf1, 0x6c, 0x7, 0x0, 0x0, 0x0,
					0x0, 0xf0, 0x4d, 0x8b, 0x18, 0xcf, 0x51, 0xdc, 0x5, 0x0, 0x0,
					0x0, 0x0, 0xf0, 0x57, 0x0, 0x0, 0x0, 0x0, 0xf0, 0xa, 0x0, 0x0,
					0x0, 0x0, 0xf0,
				},
			},
		},
		{
			// canboatjs equivalent:
			// {"prio":3,"pgn":126992,"dst":255,"src":127,"timestamp":"2021-05-26T07:36:00.495Z",
			// "input":["2021-05-26T07:36:00.495Z,3,126992,127,255,8,00,f0,55,49,b8,d9,4e,10"],
			// "fields":{"SID":0,"Source":"GPS","Date":"2021.05.26","Time":"07:36:00.30000"},"description":"System Time"}
			name: "ok, 126992, System Time",
			when: "93130310f001ff7f1bbcf1460800f05549b8d94e1045",
			expect: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    3,
					PGN:         126992,
					Destination: 255,
					Source:      127,
				},
				//Timestamp:   0x46f1bc1b,
				Data: []uint8{0x0, 0xf0, 0x55, 0x49, 0xb8, 0xd9, 0x4e, 0x10},
			},
		},
		{
			// canboatjs equivalent:
			// {"prio":6,"pgn":129539,"dst":255,"src":127,"timestamp":"2021-05-26T07:36:00.496Z",
			// "input":["2021-05-26T07:36:00.496Z,6,129539,127,255,8,00,d3,0e,01,36,01,ff,7f"],
			// "fields":{"SID":0,"Desired Mode":"Auto","Actual Mode":"3D","HDOP":2.7,"VDOP":3.1},"description":"GNSS DOPs"}
			name: "ok, 129539, GNSS DOPs",
			when: "93130603fa01ff7f1cbcf1460800d30e013601ff7f2a",
			expect: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    6,
					PGN:         129539,
					Destination: 255,
					Source:      127,
				},
				//Timestamp:   0x46f1bc1c,
				Data: []uint8{0x0, 0xd3, 0xe, 0x1, 0x36, 0x1, 0xff, 0x7f},
			},
		},
		{
			// canboatjs equivalent:
			// {"prio":7,"pgn":127258,"dst":255,"src":127,"timestamp":"2021-05-26T07:36:00.496Z",
			// "input":["2021-05-26T07:36:00.496Z,7,127258,127,255,8,00,f6,ff,ff,31,05,ff,ff"],
			// "fields":{"SID":0,"Source":"WMM 2010","Variation":0.1329},"description":"Magnetic Variation"}
			name: "ok, 127258, Magnetic Variation",
			when: "9313071af101ff7f1dbcf1460800f6ffff3105ffff89",
			expect: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    7,
					PGN:         127258,
					Destination: 255,
					Source:      127,
				},
				//Timestamp:   0x46f1bc1d,
				Data: []uint8{0x0, 0xf6, 0xff, 0xff, 0x31, 0x5, 0xff, 0xff},
			},
		},
		{
			// canboatjs equivalent:
			// {"prio":3,"pgn":127257,"dst":255,"src":128,"timestamp":"2021-05-26T07:36:00.497Z",
			// "input":["2021-05-26T07:36:00.497Z,3,127257,128,255,8,00,ff,7f,77,fc,ec,f9,ff"],
			// "fields":{"SID":0,"Pitch":-0.0905,"Roll":-0.1556},"description":"Attitude"}
			name: "ok, 127257, Attitude",
			when: "93130319f101ff801dbcf1460800ff7f77fcecf9ffe0",
			expect: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    3,
					PGN:         127257,
					Destination: 255,
					Source:      128,
				},
				//Timestamp:   0x46f1bc1d,
				Data: []uint8{0x0, 0xff, 0x7f, 0x77, 0xfc, 0xec, 0xf9, 0xff},
			},
		},
		{
			name: "ok, 130827, Lowrance: unknown",
			when: "9310070bff01ff08af172e00053f9f0200006b",
			expect: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    0x7,
					PGN:         130827, // 0x1ff0b
					Destination: 0xff,
					Source:      0x8,
				},
				//Timestamp:   3020719, // 0x2e17af,
				Data: []uint8{0x3f, 0x9f, 0x2, 0x0, 0x0},
			},
		},
		{
			name: "ok, 126208",
			when: "93110300ed01080353a07200060200ef01010002",
			expect: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    0x3,
					PGN:         126208, // 0x1ed00
					Destination: 0x8,
					Source:      0x3,
				},
				//Timestamp:   0x72a053, // 7512147
				Data: []uint8{0x2, 0x0, 0xef, 0x1, 0x1, 0x0},
			},
		},
		{
			name:        "nok, actual length 8!=10",
			when:        "9313020df101ff0c1f23d30908ff0700ff7f0000ffffa6",
			expect:      nmea.RawMessage{},
			expectError: "data length byte value is different from actual length, 8!=10",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := hex.DecodeString(tc.when)
			assert.NoError(t, err)

			result, err := fromActisenseNGTBinaryMessage(raw, now)

			assert.Equal(t, tc.expect, result)
			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFromActisenseN2KBinaryMessage(t *testing.T) {
	now := time.Unix(1623928400, 0)
	var testCases = []struct {
		name        string
		when        string
		expect      nmea.RawMessage
		expectError string
	}{
		{
			name: "ok, parse PGN 130845",
			when: "d0ec00ff0b1dff1de118" +
				"e419003f9f1212ff1515" +
				"074816de819411ffffff" +
				"7f0110470fcb38100eff" +
				"ffff7f011ac10fc822a0" +
				"0fffffff7f011dbe0669" +
				"f33c0fffffff7f010b7f" +
				"1a12c75c12ffffff7f01" +
				"047a25a9395c12ffffff" +
				"7f0114820ff0ce740eff" +
				"ffff7f01066a1ca6a6c0" +
				"12ffffff7f01094338c7" +
				"955014ffffff7f01cf12" +
				"0ac5213c0fffffff7f01" +
				"58f908029d3c0fffffff" +
				"7f01487d13db403011ff" +
				"ffff7f01497107b80b10" +
				"0effffff7f01418036c4" +
				"23c012ffffff7f0142c8" +
				"17e3c39411ffffff7f01" +
				"515618b9c0c012ffffff" +
				"7f014aa61da824cc10ff" +
				"ffff7f014b1a1b4e5b5c" +
				"12ffffff7f01c3",
			expect: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					PGN:         130845,
					Source:      11,
					Destination: 255,
					Priority:    7,
				},
				Data: []byte{
					0x3f,
					0x9f, 0x12, 0x12, 0xff, 0x15, 0x15, 0x7, 0x48, 0x16, 0xde,
					0x81, 0x94, 0x11, 0xff, 0xff, 0xff, 0x7f, 0x1, 0x10, 0x47,
					0xf, 0xcb, 0x38, 0x10, 0xe, 0xff, 0xff, 0xff, 0x7f, 0x1,
					0x1a, 0xc1, 0xf, 0xc8, 0x22, 0xa0, 0xf, 0xff, 0xff, 0xff,
					0x7f, 0x1, 0x1d, 0xbe, 0x6, 0x69, 0xf3, 0x3c, 0xf, 0xff,
					0xff, 0xff, 0x7f, 0x1, 0xb, 0x7f, 0x1a, 0x12, 0xc7, 0x5c,
					0x12, 0xff, 0xff, 0xff, 0x7f, 0x1, 0x4, 0x7a, 0x25, 0xa9,
					0x39, 0x5c, 0x12, 0xff, 0xff, 0xff, 0x7f, 0x1, 0x14, 0x82,
					0xf, 0xf0, 0xce, 0x74, 0xe, 0xff, 0xff, 0xff, 0x7f, 0x1,
					0x6, 0x6a, 0x1c, 0xa6, 0xa6, 0xc0, 0x12, 0xff, 0xff, 0xff,
					0x7f, 0x1, 0x9, 0x43, 0x38, 0xc7, 0x95, 0x50, 0x14, 0xff,
					0xff, 0xff, 0x7f, 0x1, 0xcf, 0x12, 0xa, 0xc5, 0x21, 0x3c,
					0xf, 0xff, 0xff, 0xff, 0x7f, 0x1, 0x58, 0xf9, 0x8, 0x2,
					0x9d, 0x3c, 0xf, 0xff, 0xff, 0xff, 0x7f, 0x1, 0x48, 0x7d,
					0x13, 0xdb, 0x40, 0x30, 0x11, 0xff, 0xff, 0xff, 0x7f, 0x1,
					0x49, 0x71, 0x7, 0xb8, 0xb, 0x10, 0xe, 0xff, 0xff, 0xff,
					0x7f, 0x1, 0x41, 0x80, 0x36, 0xc4, 0x23, 0xc0, 0x12, 0xff,
					0xff, 0xff, 0x7f, 0x1, 0x42, 0xc8, 0x17, 0xe3, 0xc3, 0x94,
					0x11, 0xff, 0xff, 0xff, 0x7f, 0x1, 0x51, 0x56, 0x18, 0xb9,
					0xc0, 0xc0, 0x12, 0xff, 0xff, 0xff, 0x7f, 0x1, 0x4a, 0xa6,
					0x1d, 0xa8, 0x24, 0xcc, 0x10, 0xff, 0xff, 0xff, 0x7f, 0x1,
					0x4b, 0x1a, 0x1b, 0x4e, 0x5b, 0x5c, 0x12, 0xff, 0xff, 0xff,
					0x7f, 0x1, 0xc3,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := hex.DecodeString(tc.when)
			assert.NoError(t, err)

			result, err := fromActisenseN2KBinaryMessage(raw, now)

			assert.Equal(t, tc.expect, result)
			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNGT1Device_Read(t *testing.T) {
	exampleData := test_test.LoadBytes(t, "actisense-serial-ng1-cat-usb-2021-05-14-1005.bin")
	r := bytes.NewReader(exampleData)
	wr := bufio.NewReadWriter(bufio.NewReader(r), nil)

	device := NewBinaryDevice(wr)
	i := 0
	for {
		packet, err := device.ReadRawMessage(context.Background())
		if err == io.EOF {
			break
		}
		i++

		assert.NoError(t, err)
		assert.Equal(t, packet, packet)
		if err != nil || i > 20 {
			break
		}
	}
}

func TestFromRawActisenseMessage(t *testing.T) {
	now := time.Unix(1623928400, 0)
	var testCases = []struct {
		name        string
		when        string
		expect      nmea.RawMessage
		expectError string
	}{
		{
			name: "ok, ISORequest broadcast, address claim",
			when: "95093eb7feffea1800ee0080",
			expect: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    0x6,
					PGN:         uint32(nmea.PGNISORequest),
					Destination: nmea.AddressGlobal,
					Source:      nmea.AddressNull,
				},
				Data: []uint8{0x0, 0xee, 0x0},
			},
		},
		{
			name: "ok, 130310",
			when: "950ea57f1606fd1501c170ffffffffffde",
			expect: nmea.RawMessage{
				Time: now,
				Header: nmea.CanBusHeader{
					Priority:    0x5,
					PGN:         130310,
					Destination: nmea.AddressGlobal,
					Source:      22,
				},
				Data: []uint8{0x1, 0xc1, 0x70, 0xff, 0xff, 0xff, 0xff, 0xff},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := hex.DecodeString(tc.when)
			assert.NoError(t, err)

			result, err := fromRawActisenseMessage(raw, now)

			assert.Equal(t, tc.expect, result)
			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
