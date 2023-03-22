package addressmapper

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/aldas/go-nmea-client"
	"sync"
	"time"
)

const addressMapperWriteChannelSize = 20

//func (p PGN) asBytes() []byte {
//	return []byte{
//		uint8(p & 0xff),
//		uint8((p >> 8) & 0xff),
//		uint8((p >> 16) & 0xff),
//	}
//}

type Node struct {
	Source uint8

	NAME      uint64
	Name      NodeName
	ValidName bool

	ProductInfo      ProductInfo
	ValidProductInfo bool

	ConfigurationInfo      ConfigurationInfo
	ValidConfigurationInfo bool
}

type Nodes []Node

type ProductInfo struct { // 22 bytes
	NMEA2000Version uint16 // (16 bits)
	ProductCode     uint16 // (16 bits)

	ModelID             string // (32 bytes)
	SoftwareVersionCode string // (32 bytes)
	ModelVersion        string // (32 bytes)
	ModelSerialCode     string // (32 bytes)

	CertificationLevel uint8 // (8 bits)
	LoadEquivalency    uint8 // (8 bits)
}

func PGN126996ToProductInfo(raw nmea.RawMessage) (ProductInfo, error) {
	if raw.Header.PGN != 126996 {
		return ProductInfo{}, errors.New("product info can only be created from rawMessage with PGN 126996")
	}
	b := raw.Data
	if len(b) != 134 {
		return ProductInfo{}, errors.New("rawMessage has invalid length to be product info")
	}

	NMEA2000Version, err := b.DecodeVariableUint(0, 16)
	if err != nil && err != nmea.ErrValueNoData {
		return ProductInfo{}, fmt.Errorf("failed to extract NMEA200 version for product info, err: %w", err)
	}
	productCode, err := b.DecodeVariableUint(16, 16)
	if err != nil && err != nmea.ErrValueNoData {
		return ProductInfo{}, fmt.Errorf("failed to extract product code for product info, err: %w", err)
	}

	modelID, err := b.DecodeStringFix(32, 256)
	if err != nil {
		return ProductInfo{}, fmt.Errorf("failed to extract model id for product info, err: %w", err)
	}
	softwareVersionCode, err := b.DecodeStringFix(32+256, 256)
	if err != nil {
		return ProductInfo{}, fmt.Errorf("failed to extract software version code for product info, err: %w", err)
	}
	modelVersion, err := b.DecodeStringFix(544, 256)
	if err != nil {
		return ProductInfo{}, fmt.Errorf("failed to extract model version for product info, err: %w", err)
	}
	modelSerialCode, err := b.DecodeStringFix(800, 256)
	if err != nil {
		return ProductInfo{}, fmt.Errorf("failed to extract model serial code for product info, err: %w", err)
	}

	return ProductInfo{
		NMEA2000Version: uint16(NMEA2000Version),
		ProductCode:     uint16(productCode),

		ModelID:             modelID,
		SoftwareVersionCode: softwareVersionCode,
		ModelVersion:        modelVersion,
		ModelSerialCode:     modelSerialCode,

		CertificationLevel: b[132],
		LoadEquivalency:    b[133],
	}, nil
}

// NodeName holds information about node/device to identify it in the NMEA bus. Is acquired by requesting PGN 60928 (ISO Address Claim) from device.
// Related info about SAE1939 Addresses https://embeddedflakes.com/network-management-in-sae-j1939/
type NodeName struct { // PGN 60928 is actually 64 bits
	UniqueNumber        uint32 // ISO Identity Number (21 bits)
	Manufacturer        uint16 // Device Manufacturer (11 bits)
	DeviceInstanceLower uint8  // J1939 ECU Instance (3 bits)
	DeviceInstanceUpper uint8  // J1939 Function Instance (5 bits)
	DeviceFunction      uint8  // (8 bits)
	// reserved (1 bit)
	DeviceClass    uint8 // (7 bits)
	SystemInstance uint8 // ISO Device Class Instance (4 bits)
	IndustryGroup  uint8 // (3 bits)

	// Quote from https://embeddedflakes.com/network-management-in-sae-j1939/:
	// "This 1 bit field indicate whether the CA is arbitrary field capable or not. It is used to resolve address claim
	//  conflict. If this bit is set to 1, this CA will resolve the address conflict with the one whose NAME have higher
	//  priority (lower numeric value) by selecting address from range 128 to 247."
	ArbitraryAddressCapable uint8 // arbitrary address capable (1 bit)
}

func (n NodeName) Bytes() []byte {
	return []byte{
		uint8(n.UniqueNumber >> 16 & 0xff),                                // 0
		uint8(n.UniqueNumber >> 8 & 0xff),                                 // 1
		uint8(n.UniqueNumber&0b11111) | uint8(n.Manufacturer>>8&0b111)<<3, // 2
		uint8(n.Manufacturer >> 3 & 0xff),                                 // 3
		n.DeviceInstanceLower&0b111 | n.DeviceInstanceUpper&0b11111<<3,    // 4
		n.DeviceFunction,   // 5
		n.DeviceClass << 1, // 6
		n.SystemInstance&0b1111 | (n.IndustryGroup&0b111)<<4 | n.ArbitraryAddressCapable<<7, // 7
	}
}

func (n NodeName) Uint64() uint64 {
	return binary.BigEndian.Uint64(n.Bytes())
}

func PGN60928ToNodeName(raw nmea.RawMessage) (NodeName, error) {
	if raw.Header.PGN != uint32(nmea.PGNISOAddressClaim) {
		return NodeName{}, errors.New("device name can only be created from rawMessage with PGN 60928")
	}
	b := raw.Data
	if len(b) != 8 {
		return NodeName{}, errors.New("rawMessage has invalid length to be ISO Address claim")
	}
	uqNumber := uint32(b[2]&0b11111) | uint32(b[1])<<8 | uint32(b[0])<<16
	manufacturer := uint16(b[3])<<3 | uint16(b[2]>>5)
	return NodeName{
		UniqueNumber:            uqNumber,            // 21 bits
		Manufacturer:            manufacturer,        // 11 bits
		DeviceInstanceLower:     b[4] & 0b111,        // 3 bits
		DeviceInstanceUpper:     b[4] >> 3,           // 5 bits
		DeviceFunction:          b[5],                // 8 bits
		DeviceClass:             b[6] >> 1,           // 7 bits + 1 reserved bit
		SystemInstance:          b[7] & 0b1111,       // 4 bits
		IndustryGroup:           (b[7] >> 4) & 0b111, // 3 bits
		ArbitraryAddressCapable: b[7] >> 7,           // 1 bit
	}, nil
}

type ConfigurationInfo struct {
	InstallationDesc1 string
	InstallationDesc2 string
	ManufacturerInfo  string
}

type AddressMapper struct {
	mutex sync.Mutex

	// when new device is detected we use this channel to send additional PGNs to query information about that node
	// ask for device name, product info, configuration info, pgn list
	requestsChan    chan nmea.RawMessage
	toggleWriteChan chan bool

	writeEnabled bool
	isRunning    bool

	nmeaDevice nmea.RawMessageWriter

	knownNodes   map[uint64]*Node
	address2node [255]*busSlot

	now func() time.Time
}

func NewAddressMapper(nmeaDevice nmea.RawMessageWriter) *AddressMapper {
	return &AddressMapper{
		mutex: sync.Mutex{},
		now:   time.Now,

		toggleWriteChan: make(chan bool),
		requestsChan:    make(chan nmea.RawMessage, addressMapperWriteChannelSize), // TODO: messages could come in bursts. 100+ in very short window for broadcasts (dst=255)
		nmeaDevice:      nmeaDevice,

		knownNodes:   make(map[uint64]*Node),
		address2node: [255]*busSlot{},
	}
}

func (m *AddressMapper) ToggleWrite() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.writeEnabled = !m.writeEnabled
	if m.isRunning {
		m.toggleWriteChan <- m.writeEnabled
	}
}

// Run starts AddressMapper process and block until context is cancelled or error occurs
func (m *AddressMapper) Run(ctx context.Context) error {
	buffer := newQueue[nmea.RawMessage](50)
	writeTimer := time.NewTicker(10 * time.Millisecond)

	m.mutex.Lock()
	if m.isRunning {
		return errors.New("address mapper process in already running")
	}
	m.isRunning = true
	defer func() {
		m.mutex.Lock()
		m.isRunning = false
		m.mutex.Unlock()
	}()
	enabled := m.writeEnabled
	m.mutex.Unlock()

	if !enabled {
		writeTimer.Stop()
	}
	for {
		select {
		case writeEnabled := <-m.toggleWriteChan:
			enabled = writeEnabled
			if enabled {
				writeTimer.Reset(10 * time.Millisecond)
			} else {
				writeTimer.Stop()
			}

		case msg, ok := <-m.requestsChan:
			if !ok {
				return errors.New("address mapper request channel is closed unexpectedly")
			}
			if enabled {
				buffer.Enqueue(msg)
			}

		case <-writeTimer.C:
			msg, ok := buffer.Dequeue()
			if !ok {
				continue
			}
			if err := m.nmeaDevice.WriteRawMessage(ctx, msg); err != nil {
				fmt.Printf("# address mapper writer (PGN: %v), err: %v\n", msg.Header.PGN, err)
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

type busSlot struct {
	node    *Node
	claimed time.Time

	productInfoRequested time.Time
	configInfoRequested  time.Time
	pgnListRequested     time.Time

	lastPacket time.Time
}

func (m *AddressMapper) BroadcastIsoAddressClaimRequest() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.requestsChan <- createISORequest(nmea.PGNISOAddressClaim, nmea.AddressGlobal)
}

func (m *AddressMapper) Process(raw nmea.RawMessage) (bool, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	source := raw.Header.Source
	var slot *busSlot
	if source >= nmea.AddressNull { // addresses 254 and 255 have special meaning and does not represent actual address for node
		slot = new(busSlot)
	} else {
		slot = m.address2node[source]
		if slot == nil {
			slot = new(busSlot)
			m.address2node[source] = slot
		}
		slot.lastPacket = raw.Time
	}

	isBusNodeChanged := false
	switch nmea.PGN(raw.Header.PGN) {
	case nmea.PGNISOAddressClaim:
		isChanged, err := m.processISOAddressClaim(slot, raw)
		if err != nil {
			return false, err
		}
		isBusNodeChanged = isChanged
	case nmea.PGNProductInfo:
		if err := m.processProductInfo(slot, raw); err != nil {
			return false, err
		}
	case nmea.PGNConfigurationInformation:
		if err := m.processConfigurationInfo(slot, raw); err != nil {
			return false, err
		}
	case nmea.PGNPGNList:
		if err := m.processPGNList(slot, raw); err != nil {
			return false, err
		}
	}
	return isBusNodeChanged, nil
}

func (m *AddressMapper) processISOAddressClaim(slot *busSlot, raw nmea.RawMessage) (bool, error) {
	name, err := PGN60928ToNodeName(raw)
	if err != nil {
		return false, err
	}
	source := raw.Header.Source
	NAME := binary.LittleEndian.Uint64(raw.Data)

	currentNode, ok := m.knownNodes[NAME]
	if !ok { // is new unseen device so create it
		currentNode = &Node{
			Source:    source,
			NAME:      NAME,
			Name:      name,
			ValidName: true,
		}
		m.knownNodes[NAME] = currentNode
	}

	isBusNodeChanged := false
	if slot.node == nil {
		// a) in this case we probably started to listen already powered-up and claimed N2k network. assume
		//    that this name is actually (settled by claim process) owner of this address
		currentNode.Source = source
		slot.node = currentNode
		slot.claimed = m.now()
		isBusNodeChanged = true
	} else if slot.node.ValidName && currentNode.NAME < slot.node.NAME {
		slot.node.Source = nmea.AddressNull // unassign source from old node

		// b) by J1939 address claim logic this node now claims existing slot as its name is lower
		currentNode.Source = source
		slot.node = currentNode
		slot.claimed = m.now()
		isBusNodeChanged = true
	}

	// if we already have not requested, then request product info for that device
	if m.writeEnabled && slot.productInfoRequested.IsZero() {
		slot.productInfoRequested = m.now()
		m.requestsChan <- createISORequest(nmea.PGNProductInfo, source)
	}
	return isBusNodeChanged, nil
}

func (m *AddressMapper) processProductInfo(slot *busSlot, raw nmea.RawMessage) error {
	if slot.node == nil || slot.node.ValidName {
		return nil
	}

	info, err := PGN126996ToProductInfo(raw)
	if err != nil {
		return err
	}
	slot.node.ProductInfo = info
	slot.node.ValidProductInfo = true

	// if we already have not requested, then request configuration info for that node
	if m.writeEnabled && slot.configInfoRequested.IsZero() {
		slot.configInfoRequested = m.now()
		m.requestsChan <- createISORequest(nmea.PGNConfigurationInformation, raw.Header.Source)
	}
	return nil
}

func (m *AddressMapper) processConfigurationInfo(slot *busSlot, raw nmea.RawMessage) error {
	if slot.node == nil || slot.node.ValidName {
		return nil
	}

	ci, err := PGN126998ToConfigurationInfo(raw)
	if err != nil {
		return err
	}
	slot.node.ConfigurationInfo = ci
	slot.node.ValidConfigurationInfo = true

	// if we already have not requested, then request PGN list for that node
	if m.writeEnabled && slot.pgnListRequested.IsZero() {
		slot.pgnListRequested = m.now()
		m.requestsChan <- createISORequest(nmea.PGNPGNList, raw.Header.Source)
	}
	return nil
}

func PGN126998ToConfigurationInfo(raw nmea.RawMessage) (ConfigurationInfo, error) {
	if raw.Header.PGN != uint32(nmea.PGNConfigurationInformation) {
		return ConfigurationInfo{}, errors.New("configuration info can only be created from rawMessage with PGN 126998")
	}
	instDesc1, offset, err := raw.Data.DecodeStringLAU(0)
	if err != nil {
		return ConfigurationInfo{}, fmt.Errorf("failed to decode configuration info installation description 1, err: %w", err)
	}
	instDesc2, offset, err := raw.Data.DecodeStringLAU(offset)
	if err != nil {
		return ConfigurationInfo{}, fmt.Errorf("failed to decode configuration info installation description 2, err: %w", err)
	}
	manufInfo, _, err := raw.Data.DecodeStringLAU(offset)
	if err != nil {
		return ConfigurationInfo{}, fmt.Errorf("failed to decode configuration info manufacturer info, err: %w", err)
	}
	return ConfigurationInfo{
		InstallationDesc1: instDesc1,
		InstallationDesc2: instDesc2,
		ManufacturerInfo:  manufInfo,
	}, nil
}

func (m *AddressMapper) processPGNList(slot *busSlot, raw nmea.RawMessage) error {
	if slot.node == nil || slot.node.ValidName {
		return nil
	}

	return nil
}

// Nodes returns all known (current and previous) nodes from NMEA bus
func (m *AddressMapper) Nodes() Nodes {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	result := make(Nodes, 0, len(m.knownNodes))
	for _, n := range m.knownNodes {
		result = append(result, *n)
	}
	return result
}

// NodesInUseBySource returns list of Nodes that are currently in use (assigned valid source address).
func (m *AddressMapper) NodesInUseBySource() map[uint8]Node {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	result := make(map[uint8]Node)
	for _, n := range m.knownNodes {
		node := *n
		if node.Source >= nmea.AddressNull && !node.ValidName {
			continue
		}
		result[node.Source] = node
	}
	return result
}

func createISORequest(forPGN nmea.PGN, destination uint8) nmea.RawMessage {
	return nmea.RawMessage{
		Header: nmea.CanBusHeader{
			PGN:      uint32(nmea.PGNISORequest),
			Priority: 6,
			// https://copperhilltech.com/blog/sae-j1939-address-claim-procedure-sae-j193981-network-management/
			// "A node, that has not yet claimed an address, must use the NULL address (254) as the source address
			//  when sending a Request for Address Claimed message."
			// So we use 254 as source, until the day this library decides to start claiming its own address
			Source:      nmea.AddressNull,
			Destination: destination,
		},
		Data: []byte{ // order as little endian
			uint8(forPGN & 0xff),
			uint8((forPGN >> 8) & 0xff),
			uint8((forPGN >> 16) & 0xff),
		},
	}
}

type queue[T any] struct {
	items  []T
	length int
}

func newQueue[T any](length int) *queue[T] {
	return &queue[T]{
		items:  make([]T, 0, length),
		length: length,
	}
}

func (q *queue[T]) Enqueue(item T) bool {
	if len(q.items) == q.length {
		return false
	}
	q.items = append(q.items, item)
	return true
}

func (q *queue[T]) Dequeue() (T, bool) {
	var empty T
	if len(q.items) == 0 {
		return empty, false
	}
	value := q.items[0]

	q.items[0] = empty

	q.items = q.items[1:]
	return value, true
}
