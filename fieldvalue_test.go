package nmea

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestRawData_DecodeVariableUint(t *testing.T) {
	var testCases = []struct {
		name          string
		given         []byte
		whenBitOffset uint16
		whenBitLength uint16
		expect        uint64
		expectError   string
	}{
		{
			name:          "decode unsigned 16bit value",
			given:         []byte{0xFF, 0x01, 0x00, 0xFF},
			whenBitOffset: 8,
			whenBitLength: 16,
			expect:        1,
		},
		{
			name:          "decode unsigned 3bit value",
			given:         []byte{0xFF, 0b1001_1111, 0xFF, 0xFF},
			whenBitOffset: 12,
			whenBitLength: 3,
			expect:        1,
		},
		{
			name:          "decode unsigned 16bit value",
			given:         []byte{0xFF, 0x44, 0x00, 0xFF},
			whenBitOffset: 8,
			whenBitLength: 16,
			expect:        68,
		},
		{
			name:          "decode unsigned 16bit value starting at middle of byte including next byte",
			given:         []byte{0xFF, 0b0001_1111, 0b1111_0000, 0xFF},
			whenBitOffset: 12,
			whenBitLength: 8,
			expect:        1,
		},
		{
			name:          "decode unsigned 16bit value starting at middle of byte to end of same byte",
			given:         []byte{0xFF, 0b0001_1111, 0xFF, 0xFF},
			whenBitOffset: 12,
			whenBitLength: 4,
			expect:        1,
		},
		{
			name:          "decode unsigned 16bit value as no data 0xFFFF",
			given:         []byte{0xFF, 0xFF, 0xFF, 0xFF},
			whenBitOffset: 8,
			whenBitLength: 16,
			expect:        0,
			expectError:   ErrValueNoData.Error(),
		},
		{
			name:          "decode unsigned 16bit value as out of range 0xFFFE",
			given:         []byte{0xFF, 0xFE, 0xFF, 0xFF},
			whenBitOffset: 8,
			whenBitLength: 16,
			expect:        0,
			expectError:   ErrValueOutOfRange.Error(),
		},
		{
			name:          "decode unsigned 16bit value as reserved 0xFFFD",
			given:         []byte{0xFF, 0xFD, 0xFF, 0xFF},
			whenBitOffset: 8,
			whenBitLength: 16,
			expect:        0,
			expectError:   ErrValueReserved.Error(),
		},
		{
			name:          "decode unsigned 16bit value from last bytes",
			given:         []byte{0xFF, 0xff, 0x01, 0x00},
			whenBitOffset: 16,
			whenBitLength: 16,
			expect:        1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rd := RawData(tc.given)
			result, err := rd.DecodeVariableUint(tc.whenBitOffset, tc.whenBitLength)

			assert.Equal(t, tc.expect, result)
			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRawData_DecodeVariableInt(t *testing.T) {
	var testCases = []struct {
		name          string
		given         []byte
		whenBitOffset uint16
		whenBitLength uint16
		expect        int64
		expectError   string
	}{
		{
			name:          "decode signed 16bit value",
			given:         []byte{0xFF, 0x01, 0x00, 0xFF},
			whenBitOffset: 8,
			whenBitLength: 16,
			expect:        1,
		},
		{
			name:          "decode signed 16bit value",
			given:         []byte{0xFF, 0x77, 0xfc, 0xFF},
			whenBitOffset: 8,
			whenBitLength: 16,
			expect:        -905,
		},
		{
			name:          "decode signed 16bit value starting at middle of byte including next byte",
			given:         []byte{0xFF, 0b0001_1111, 0b1111_0000, 0xFF},
			whenBitOffset: 12,
			whenBitLength: 8,
			expect:        1, // 0x0000_0001
		},
		{
			name:          "decode signed 16bit value starting at middle of byte to end of same byte",
			given:         []byte{0xFF, 0b0001_1111, 0xFF, 0xFF},
			whenBitOffset: 12,
			whenBitLength: 4,
			expect:        1,
		},
		{
			name:          "decode signed 16bit value as no data 0x7FFF",
			given:         []byte{0xFF, 0xFF, 0x7F, 0xFF},
			whenBitOffset: 8,
			whenBitLength: 16,
			expect:        0,
			expectError:   ErrValueNoData.Error(),
		},
		{
			name:          "decode signed 16bit value as out of range 0x7FFE",
			given:         []byte{0xFF, 0xFE, 0x7F, 0xFF},
			whenBitOffset: 8,
			whenBitLength: 16,
			expect:        0,
			expectError:   ErrValueOutOfRange.Error(),
		},
		{
			name:          "decode signed 16bit value as reserved 0x7FFD",
			given:         []byte{0xFF, 0xFD, 0x7F, 0xFF},
			whenBitOffset: 8,
			whenBitLength: 16,
			expect:        0,
			expectError:   ErrValueReserved.Error(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rd := RawData(tc.given)
			result, err := rd.DecodeVariableInt(tc.whenBitOffset, tc.whenBitLength)

			assert.Equal(t, tc.expect, result)
			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRawData_DecodeBytes(t *testing.T) {
	var testCases = []struct {
		name                 string
		given                []byte
		whenBitOffset        uint16
		whenBitLength        uint16
		whenIsVariableLength bool
		expect               []byte
		expectReadBits       uint16
		expectError          string
	}{
		{
			name:           "decode 7bits",
			given:          []byte{0xFF},
			whenBitOffset:  0,
			whenBitLength:  7,
			expect:         []byte{0x7F},
			expectReadBits: 7,
		},
		{
			name:           "decode 8bits",
			given:          []byte{0x12},
			whenBitOffset:  0,
			whenBitLength:  8,
			expect:         []byte{0x12},
			expectReadBits: 8,
		},
		{
			name:           "decode 9bits",
			given:          []byte{0xFF, 0xFF},
			whenBitOffset:  0,
			whenBitLength:  9,
			expect:         []byte{0xFF, 0x01},
			expectReadBits: 9,
		},
		{
			name:           "decode 8bits",
			given:          []byte{0xFF, 0x12, 0xFF, 0xFF},
			whenBitOffset:  8,
			whenBitLength:  8,
			expect:         []byte{0x12},
			expectReadBits: 8,
		},
		{
			name:           "decode 3bits, starts and ends at the same byte",
			given:          []byte{0xFF, 0b1001_1111, 0xFF, 0xFF},
			whenBitOffset:  12,
			whenBitLength:  3,
			expect:         []byte{0b001},
			expectReadBits: 3,
		},
		{
			name:           "decode 16bits",
			given:          []byte{0xFF, 0x21, 0x43, 0xFF},
			whenBitOffset:  8,
			whenBitLength:  16,
			expect:         []byte{0x21, 0x43},
			expectReadBits: 16,
		},
		{
			name:           "decode 8bits starting at middle of byte including next byte",
			given:          []byte{0xFF, 0b0001_1111, 0b1111_0000, 0xFF},
			whenBitOffset:  12,
			whenBitLength:  8,
			expect:         []byte{0b0000_0001},
			expectReadBits: 8,
		},
		{
			name:           "decode 16bits starting at middle of byte including next byte",
			given:          []byte{0xFF, 0x1F, 0x32, 0xF4, 0xFF}, // 0001_1111 0011_0010 1111_0100 = 1F32F4
			whenBitOffset:  12,
			whenBitLength:  16,
			expect:         []byte{0x21, 0x43}, // 0010_0001 0100_0011
			expectReadBits: 16,
		},
		{
			name:           "decode 16bits ends at middle of last byte",
			given:          []byte{0xFF, 0x21, 0xF3, 0xFF, 0xFF},
			whenBitOffset:  8,
			whenBitLength:  12,
			expect:         []byte{0x21, 0x03}, // 0010_0001 0000_0011
			expectReadBits: 12,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rd := RawData(tc.given)
			result, bits, err := rd.DecodeBytes(tc.whenBitOffset, tc.whenBitLength, tc.whenIsVariableLength)

			assert.Equal(t, tc.expect, result)
			assert.Equal(t, tc.expectReadBits, bits)
			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRawData_DecodeTime(t *testing.T) {
	var testCases = []struct {
		name           string
		given          []byte
		whenBitOffset  uint16
		whenBitLength  uint16
		whenResolution float64
		expect         time.Duration
		expectError    string
	}{
		{
			name:           "decode time with resolution = 60",
			given:          []byte{0xcd, 0x01, 0x00, 0x64, 0xff, 0x2e, 0x2b, 0xa9, 0x00},
			whenBitOffset:  40,
			whenBitLength:  16,
			whenResolution: 60,
			expect:         184*time.Hour + 14*time.Minute, // 184:14:00 // 2E2B = 11819
		},
		{
			name:           "decode time with resolution = 0.001",
			given:          []byte{0x61, 0xea, 0x24, 0xff, 0xff, 0xff, 0xff, 0xff},
			whenBitOffset:  0,
			whenBitLength:  16,
			whenResolution: 0.001,
			expect:         1*time.Minute + 1*time.Millisecond, // 00:01:00.001 // 61EA = 60001
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rd := RawData(tc.given)
			result, err := rd.DecodeTime(tc.whenBitOffset, tc.whenBitLength, tc.whenResolution)

			assert.Equal(t, tc.expect, result)
			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRawData_DecodeStringFix(t *testing.T) {
	var testCases = []struct {
		name          string
		given         []byte
		whenBitOffset uint16
		whenBitLength uint16
		expect        string
		expectError   string
	}{
		{
			name: "decode at middle of data",
			given: []byte{
				0x18, 0x55, 0x81, 0x97, 0x0e, 0x57, 0x49, 0x54, 0x54, 0x45,
				0x20, 0x52, 0x41, 0x41, 0x46, 0x00, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xe1, 0xff,
			},
			whenBitOffset: 40,
			whenBitLength: 160,
			expect:        "WITTE RAAF",
		},
		{
			name: "decode at start of data",
			given: []byte{
				0x57, 0x49, 0x54, 0x54, 0x45, 0x20, 0x52, 0x41, 0x41, 0x46,
				0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0xe1, 0xff,
			},
			whenBitOffset: 0,
			whenBitLength: 160,
			expect:        "WITTE RAAF",
		},
		{
			name: "ok, all bytes",
			given: []byte{
				0x57, 0x49, 0x54, 0x54, 0x45,
			},
			whenBitOffset: 0,
			whenBitLength: 40,
			expect:        "WITTE",
		},
		{
			name: "ok, first 0x0 means end of string",
			given: []byte{
				0x57, 0x49, 0x0, 0x54, 0x45,
			},
			whenBitOffset: 0,
			whenBitLength: 40,
			expect:        "WI",
		},
		{
			name: "ok, empty string",
			given: []byte{
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0xe1, 0xff,
			},
			whenBitOffset: 0,
			whenBitLength: 32,
			expect:        "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rd := RawData(tc.given)
			result, err := rd.DecodeStringFix(tc.whenBitOffset, tc.whenBitLength)

			assert.Equal(t, tc.expect, result)
			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRawData_DecodeStringLAU(t *testing.T) {
	var testCases = []struct {
		name           string
		given          []byte
		whenBitOffset  uint16
		expect         string
		expectReadBits uint16
		expectError    string
	}{
		{
			name: "decode encoding=1 utf8/ascii",
			given: []byte{
				0x26, 0x01, 0x41, 0x69, 0x72, 0x6d, 0x61, 0x72, 0x20, 0x31, // 10
				0x2d, 0x36, 0x30, 0x33, 0x2d, 0x36, 0x37, 0x33, 0x2d, 0x39, // 20
				0x35, 0x37, 0x30, 0x20, 0x77, 0x77, 0x77, 0x2e, 0x61, 0x69, // 30
				0x72, 0x6d, 0x61, 0x72, 0x2e, 0x63, 0x6f, 0x6d, // 38
			},
			whenBitOffset:  0,
			expect:         "Airmar 1-603-673-9570 www.airmar.com",
			expectReadBits: 304,
		},
		{
			name: "decode encoding=0 utf16",
			given: []byte{
				0x4C, 0x00, 0x52, 0x00, 0x6F, 0x00, 0x73, 0x00, 0x65, 0x00, // 10
				0x20, 0x00, 0x50, 0x00, 0x6F, 0x00, 0x69, 0x00, 0x6E, 0x00, // 20
				0x74, 0x00, 0x20, 0x00, 0x72, 0x00, 0x6F, 0x00, 0x73, 0x00, // 30
				0x65, 0x00, 0x70, 0x00, 0x6F, 0x00, 0x69, 0x00, 0x6E, 0x00, // 40
				0x74, 0x00, 0x2E, 0x00, 0x63, 0x00, 0x6F, 0x00, 0x6D, 0x00, // 50
				0x20, 0x00, 0x34, 0x00, 0x32, 0x00, 0x35, 0x00, 0x2D, 0x00, // 60
				0x36, 0x00, 0x30, 0x00, 0x35, 0x00, 0x2D, 0x00, 0x30, 0x00, // 70
				0x39, 0x00, 0x38, 0x00, 0x35, 0x00, // 76
			},
			whenBitOffset:  0,
			expect:         "Rose Point rosepoint.com 425-605-0985",
			expectReadBits: 608,
		},
		{
			name:           "ok, empty string",
			given:          []byte{0x02, 0x01},
			whenBitOffset:  0,
			expect:         "",
			expectReadBits: 16,
		},
		{
			name:           "nok, invalid encoding",
			given:          []byte{0x03, 0x02, 0x52},
			whenBitOffset:  0,
			expect:         "",
			expectError:    "invalid string lau encoding",
			expectReadBits: 0,
		},
		{
			name:           "nok, invalid size",
			given:          []byte{0x01, 0x02, 0x52},
			whenBitOffset:  0,
			expect:         "",
			expectError:    "string lau has invalid size below 2",
			expectReadBits: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rd := RawData(tc.given)
			result, readBits, err := rd.DecodeStringLAU(tc.whenBitOffset)

			assert.Equal(t, tc.expectReadBits, readBits)
			assert.Equal(t, tc.expect, result)
			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRawData_DecodeStringLZ(t *testing.T) {
	var testCases = []struct {
		name           string
		given          []byte
		whenBitOffset  uint16
		whenBitLength  uint16
		expect         string
		expectReadBits uint16
		expectError    string
	}{
		{
			name: "ok, decode",
			given: []byte{ // this is actual `PGN 130820 - Fusion: AM/FM Station` packet
				0xa3, 0x99, 0x0b, 0x80, 0x01, 0x02, 0x00, 0xc6, 0x3e, 0x05, // 10
				0xc7, 0x08, 0x41, 0x56, 0x52, 0x4f, 0x54, 0x52, 0x4f, 0x53, // 20
			},
			whenBitOffset:  88,
			whenBitLength:  80,
			expect:         "AVROTROS",
			expectReadBits: 64, // 8 * 8
		},
		{
			name: "ok, decode string ended by 0x0",
			given: []byte{
				0xFF, 0x05, 0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x0, 0x0, 0x0, // 10
				0x0, 0x0, 0x24, // 13
			},
			whenBitOffset:  8,
			whenBitLength:  80,
			expect:         "Hello",
			expectReadBits: 40, // 5 * 8
		},
		{
			name:           "ok, empty string terminated by 0x0",
			given:          []byte{0x02, 0x00, 0x0},
			whenBitOffset:  0,
			expect:         "",
			expectReadBits: 0,
		},
		{
			name:           "ok, empty string (null size)",
			given:          []byte{0x00},
			whenBitOffset:  0,
			expect:         "",
			expectReadBits: 8, // 8
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rd := RawData(tc.given)
			result, readBits, err := rd.DecodeStringLZ(tc.whenBitOffset, tc.whenBitLength)

			assert.Equal(t, tc.expectReadBits, readBits)
			assert.Equal(t, tc.expect, result)
			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRawData_DecodeDate(t *testing.T) {
	var testCases = []struct {
		name          string
		given         []byte
		whenBitOffset uint16
		whenBitLength uint16
		expect        time.Time
		expectError   string
	}{
		{
			name: "ok, 2022-10-31 is 19296 days since epoch",
			given: []byte{
				0xff, 0x60, 0x4B, 0xff, // 19296 -> 0x4B60
			},
			whenBitOffset: 8,
			whenBitLength: 16,
			expect:        time.Unix(1667174400, 0).UTC(), // 2022-10-31 UTC
		},
		{
			name: "ok, decode 0 day exactly epoch",
			given: []byte{
				0xff, 0x0, 0x0, 0xff,
			},
			whenBitOffset: 8,
			whenBitLength: 16,
			expect:        epoch,
		},
		{
			name:          "nok, length shorter",
			given:         []byte{0xff, 0x0, 0x0, 0xff},
			whenBitOffset: 8,
			whenBitLength: 15,
			expect:        time.Time{},
			expectError:   "can only decode date with 16 bits",
		},
		{
			name:          "nok, length longer",
			given:         []byte{0xff, 0x0, 0x0, 0xff},
			whenBitOffset: 8,
			whenBitLength: 17,
			expect:        time.Time{},
			expectError:   "can only decode date with 16 bits",
		},
		{
			name:          "nok, no data",
			given:         []byte{0xff, 0xFF, 0xFF, 0xff},
			whenBitOffset: 8,
			whenBitLength: 16,
			expect:        time.Time{},
			expectError:   "field value has no data",
		},
		{
			name:          "nok, out of range",
			given:         []byte{0xff, 0xFE, 0xFF, 0xff},
			whenBitOffset: 8,
			whenBitLength: 16,
			expect:        time.Time{},
			expectError:   "field value out of range",
		},
		{
			name:          "nok, reserved",
			given:         []byte{0xff, 0xFD, 0xFF, 0xff},
			whenBitOffset: 8,
			whenBitLength: 16,
			expect:        time.Time{},
			expectError:   "field value is reserved",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rd := RawData(tc.given)
			result, err := rd.DecodeDate(tc.whenBitOffset, tc.whenBitLength)

			assert.Equal(t, tc.expect, result)
			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRawData_DecodeDecimal(t *testing.T) {
	var testCases = []struct {
		name          string
		given         []byte
		whenBitOffset uint16
		whenBitLength uint16
		expect        uint64
		expectError   string
	}{
		{
			name: "ok, decode",
			given: []byte{
				0xff, 0x33, 0x14, 0x00, 0x5F, 0x1E, 0xff,
			},
			whenBitOffset: 8,
			whenBitLength: 40,
			expect:        uint64(51_20_00_95_30), // 0x33 ,0x14 ,0x00 ,0x5F ,0x1E
		},
		{
			name: "ok, decode nmea example",
			given: []byte{
				0x0C, 0x22, 0x38, 0x4E, 0x5A,
			},
			whenBitOffset: 0,
			whenBitLength: 40,
			expect:        uint64(12_34_56_78_90), // 0x0C, 0x22, 0x38, 0x4E, 0x5A -> 12 34 56 78 90
		},
		{
			name: "nok, no data available",
			given: []byte{
				0xFF, 0xFF,
			},
			whenBitOffset: 0,
			whenBitLength: 16,
			expect:        uint64(0),
			expectError:   ErrValueNoData.Error(),
		},
		{
			name: "nok, byte has too large value to have only 2 digits",
			given: []byte{
				0x0C, 0x64,
			},
			whenBitOffset: 0,
			whenBitLength: 16,
			expect:        uint64(0), // 0x0C, 0x64 -> 12 100
			expectError:   "decimal contains byte with value larger than 2 digits",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rd := RawData(tc.given)
			result, err := rd.DecodeDecimal(tc.whenBitOffset, tc.whenBitLength)

			assert.Equal(t, tc.expect, result)
			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRawData_DecodeFloat(t *testing.T) {
	var testCases = []struct {
		name          string
		given         []byte
		whenBitOffset uint16
		whenBitLength uint16
		expect        float64
		expectError   string
	}{
		{
			name:          "ok, decode",
			given:         []byte{0xcd, 0xcc, 0xec, 0x3f},
			whenBitOffset: 0,
			whenBitLength: 32,
			expect:        1.85,
		},
		{
			name:          "nok, incorrect bit length",
			given:         []byte{0xcd, 0xcc, 0xec, 0x3f},
			whenBitOffset: 0,
			whenBitLength: 31,
			expect:        float64(0),
			expectError:   "can only decode float with 32 bits",
		},
		{
			name:          "nok, no data available",
			given:         []byte{0xFF, 0xFF, 0xFF, 0xFF},
			whenBitOffset: 0,
			whenBitLength: 32,
			expect:        float64(0),
			expectError:   ErrValueNoData.Error(),
		},
		{
			name:          "nok, no data available",
			given:         []byte{0xFE, 0xFF, 0xFF, 0xFF},
			whenBitOffset: 0,
			whenBitLength: 32,
			expect:        float64(0),
			expectError:   ErrValueOutOfRange.Error(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rd := RawData(tc.given)
			result, err := rd.DecodeFloat(tc.whenBitOffset, tc.whenBitLength)

			assert.InDelta(t, tc.expect, result, 0.000001)
			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
