package nmea

import (
	test_test "github.com/aldas/go-nmea-client/test"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestPGN126996ToProductInfo(t *testing.T) {
	var testCases = []struct {
		name        string
		given       RawMessage
		expect      ProductInfo
		expectError string
	}{
		{
			name: "ok, all fields set",
			given: RawMessage{
				Time: time.Time{},
				Header: CanBusHeader{
					PGN:         126996,
					Priority:    6,
					Source:      51,
					Destination: 255,
				},
				Data: []byte{
					0x34, 0x08, 0x15, 0x0b, 0x41, 0x50, 0x37, 0x30, 0x20, 0x4d, // 10
					0x6b, 0x32, 0x20, 0x41, 0x75, 0x74, 0x6f, 0x70, 0x69, 0x6c, // 20
					0x6f, 0x74, 0x20, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, // 30
					0x6c, 0x65, 0x72, 0x20, 0x20, 0x20, 0x30, 0x31, 0x30, 0x30, // 40
					0x30, 0x5f, 0x45, 0x20, 0x32, 0x2e, 0x30, 0x2e, 0x30, 0x2e, // 50
					0x36, 0x34, 0x2e, 0x34, 0x2e, 0x33, 0x34, 0x20, 0x20, 0x20, // 60
					0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, // 70
					0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, // 80
					0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, // 90
					0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, // 100
					0x31, 0x32, 0x38, 0x37, 0x38, 0x37, 0x30, 0x39, 0x33, 0x20, // 110
					0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, // 120
					0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, // 130
					0x20, 0x20, 0x02, 0x01, // 134
				},
			},
			expect: ProductInfo{
				NMEA2000Version:     2100,
				ProductCode:         2837,
				ModelID:             "AP70 Mk2 Autopilot Controller   ",
				SoftwareVersionCode: "01000_E 2.0.0.64.4.34           ",
				ModelVersion:        "                                ",
				ModelSerialCode:     "128787093                       ",
				CertificationLevel:  0x2,
				LoadEquivalency:     0x1,
			},
		},
		{
			name: "ok, some fields set",
			given: RawMessage{
				Time: time.Time{},
				Header: CanBusHeader{
					PGN:         126996,
					Priority:    6,
					Source:      47,
					Destination: 255,
				},
				Data: []byte{
					0xff, 0xff, 0xff, 0xff, 0x51, 0x53, 0x38, 0x30, 0x20, 0x20, // 10
					0x20, 0x20, 0x5f, 0x50, 0x69, 0x6c, 0x6f, 0x74, 0x20, 0x63, // 20
					0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x6c, 0x65, 0x72, 0x20, // 30
					0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x31, 0x31, 0x30, 0x30, // 40
					0x20, 0x20, 0x20, 0x20, 0x31, 0x33, 0x30, 0x32, 0x30, 0x30, // 50
					0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, // 60
					0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, // 70
					0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, // 80
					0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, // 90
					0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, // 100
					0x30, 0x30, 0x37, 0x37, 0x38, 0x37, 0x23, 0x20, 0x20, 0x20, // 110
					0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, // 120
					0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, // 130
					0x20, 0x20, 0x01, 0x03, //  134
				},
			},
			expect: ProductInfo{
				NMEA2000Version:     0,
				ProductCode:         0,
				ModelID:             "QS80    _Pilot controller       ", // NB: Canboat trims tailing spaces
				SoftwareVersionCode: "1100    130200                  ",
				ModelVersion:        "                                ",
				ModelSerialCode:     "007787#                         ",
				CertificationLevel:  1,
				LoadEquivalency:     3,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := PGN126996ToProductInfo(tc.given)
			assert.Equal(t, tc.expect, result)
			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPGN60928ToDeviceName(t *testing.T) {
	now := test_test.UTCTime(1665488842) // Tue Oct 11 2022 11:47:22 GMT+0000

	var testCases = []struct {
		name        string
		given       RawMessage
		expect      NodeName
		expectError string
	}{
		{
			name: "ok",
			given: RawMessage{
				Time: now,
				Header: CanBusHeader{
					PGN:         60928,
					Priority:    6,
					Source:      23,
					Destination: 255,
				},
				Data: []byte{0x1e, 0x7d, 0x3e, 0xe8, 0x00, 0x87, 0x32, 0xc0},
			},
			expect: NodeName{
				UniqueNumber:            1998110, // 0x1E7D1E
				Manufacturer:            1857,    // Simrad (0x741)
				DeviceInstanceLower:     0,
				DeviceInstanceUpper:     0,
				DeviceFunction:          135, // NMEA 0183 Gateway
				DeviceClass:             25,  // Internetwork device
				SystemInstance:          0,
				IndustryGroup:           4, // Marine
				ArbitraryAddressCapable: 1,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := PGN60928ToNodeName(tc.given)
			assert.Equal(t, tc.expect, result)
			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNodeName_Uint64(t *testing.T) {
	var testCases = []struct {
		name   string
		given  NodeName
		expect uint64
	}{
		{
			name: "ok",
			given: NodeName{
				UniqueNumber:            1998110, // 0x1E7D1E
				Manufacturer:            1857,    // Simrad (0x741)
				DeviceInstanceLower:     0,
				DeviceInstanceUpper:     0,
				DeviceFunction:          135, // NMEA 0183 Gateway
				DeviceClass:             25,  // Internetwork device
				SystemInstance:          0,
				IndustryGroup:           4, // Marine
				ArbitraryAddressCapable: 1,
			},
			expect: 0x1e7d3ee8008732c0,
			// 0001 1110 // 0-7   // 0x1e
			// 0111 1101 // 8-15  // 0x7d
			// 0011 1110 // 16-23 // 0x3e
			// 1110 1000 // 24-33 // 0xe8
			// 0000 0000 // 32-39 // 0x00
			// 1000 0111 // 40-47 // 0x87
			// 0011 0010 // 48-53 // 0x32
			// 1100 0000 // 54-63 // 0xc0
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.given.Uint64()
			assert.Equal(t, tc.expect, result)
		})
	}
}

func TestCreateISORequest(t *testing.T) {
	var testCases = []struct {
		name            string
		whenPGN         PGN
		whenDestination uint8
		expect          RawMessage
	}{
		{
			name:            "ok, ISO address claim broadcast",
			whenPGN:         PGNISOAddressClaim,
			whenDestination: addressGlobal,
			expect: RawMessage{
				Header: CanBusHeader{
					PGN:         59904,
					Priority:    6,
					Source:      addressNull,
					Destination: addressGlobal,
				},
				Data: []byte{0x0, 0xEE, 0x0},
			},
		},
		{
			name:            "ok, ISO address claim addressed",
			whenPGN:         PGNISOAddressClaim,
			whenDestination: 32,
			expect: RawMessage{
				Header: CanBusHeader{
					PGN:         uint32(PGNISORequest),
					Priority:    6,
					Source:      addressNull,
					Destination: 32,
				},
				Data: []byte{0x0, 0xEE, 0x0},
			},
		},
		{
			name:            "ok, product info to addressed",
			whenPGN:         PGNProductInfo,
			whenDestination: 32,
			expect: RawMessage{
				Header: CanBusHeader{
					PGN:         uint32(PGNISORequest),
					Priority:    6,
					Source:      addressNull,
					Destination: 32,
				},
				Data: []byte{0x14, 0xf0, 0x1},
			},
		},
		{
			name:            "ok, configuration info to broadcast",
			whenPGN:         PGNConfigurationInformation,
			whenDestination: addressGlobal,
			expect: RawMessage{
				Header: CanBusHeader{
					PGN:         uint32(PGNISORequest),
					Priority:    6,
					Source:      addressNull,
					Destination: addressGlobal,
				},
				Data: []byte{0x16, 0xf0, 0x1},
			},
		},
		{
			name:            "ok, PGN list addressed",
			whenPGN:         PGNPGNList,
			whenDestination: 45,
			expect: RawMessage{
				Header: CanBusHeader{
					PGN:         uint32(PGNISORequest),
					Priority:    6,
					Source:      addressNull,
					Destination: 45,
				},
				Data: []byte{0x0, 0xEE, 0x1},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := createISORequest(tc.whenPGN, tc.whenDestination)
			assert.Equal(t, tc.expect, result)
		})
	}
}

func TestQueue(t *testing.T) {
	q := newQueue[int](5)

	q.Enqueue(1)

	item, ok := q.Dequeue()
	assert.True(t, ok)
	assert.Equal(t, 1, item)

	q.Enqueue(2)
	q.Enqueue(3)

	item, ok = q.Dequeue()
	assert.True(t, ok)
	assert.Equal(t, 2, item)

	item, ok = q.Dequeue()
	assert.True(t, ok)
	assert.Equal(t, 3, item)

}
