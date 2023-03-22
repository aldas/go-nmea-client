package actisense

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/aldas/go-nmea-client"
	"io"
	"os"
	"syscall"
	"time"
)

const (
	// STX start packet byte for Actisense parsed NMEA2000 packet
	STX = 0x02
	// ETX end packet byte for Actisense parsed NMEA2000 packet
	ETX = 0x03
	// DLE marker byte before start/end packet byte. Is sent before STX or ETX byte is sent (DLE+STX or DLE+ETX)
	DLE = 0x10

	// cmdNGTMessageReceived identifies that packet is received/incoming NMEA200 data message as NGT binary format.
	cmdNGTMessageReceived = 0x93
	// cmdN2KMessageRequestReceived identifies that packet is sent/outgoing NMEA200 data message as NGT binary format.
	cmdNGTMessageSend = 0x94

	// cmdRAWActisenseMessageReceived identifies that packet is received/incoming NMEA200 data message as RAW Actisense format.
	cmdRAWActisenseMessageReceived = 0x95
	// cmdRAWActisenseMessageSend identifies that packet is sent/outgoing NMEA200 data message as RAW Actisense format.
	cmdRAWActisenseMessageSend = 0x96

	cmdN2KMessageReceived = 0xD0
	// cmdN2KMessageReceived identifies that packet is sent/outgoing NMEA200 data message as N2K binary format.
	cmdN2KMessageSend = 0xD1

	// cmdDeviceMessageReceived identifies that received packet is (BEMCMD) Actisense NGT specific message
	cmdDeviceMessageReceived = 0xA0
	// cmdDeviceMessageSend identifies that sent packet is Actisense NGT specific message
	cmdDeviceMessageSend = 0xA1

	// CanBoatFakePGNOffset is offset for PGNs that Actisense devices create for their own information. We add it to
	// parsed PGN and after that we can find match from Canboat PGN database with that
	CanBoatFakePGNOffset uint32 = 0x40000
)

// BinaryFormatDevice is implementing Actisense device using binary formats (NGT1 and N2K binary)
type BinaryFormatDevice struct {
	device io.ReadWriter

	sleepFunc func(timeout time.Duration)
	timeNow   func() time.Time

	config Config
}

// Config is configuration for Actisense NGT-1 device
type Config struct {
	// ReceiveDataTimeout is maximum duration reads from device can produce no data until we error out (idle).
	//
	// It is to limit amount of time reads can result no data. to timeout the connection when there is no
	// interaction in bus. This is different from for example serial device readTimeout which limits how much time Read
	// call blocks. We want to `Read` calls block small amount of time to be able to check if context was cancelled
	// during read but at the same time we want to be able to detect when there are no coming from bus for excessive
	// amount of time.
	ReceiveDataTimeout time.Duration

	// DebugLogRawMessageBytes instructs device to log all sent/received raw messages
	DebugLogRawMessageBytes bool
	// OutputActisenseMessages instructs device to output Actisense own messages
	OutputActisenseMessages bool

	// IsN2KWriter instructs device to write/send messages to NMEA200 bus as N2K binary format (used by Actisense W2K-1)
	IsN2KWriter bool

	// FastPacketAssembler assembles fast-packet PGN frames to complete messages.
	// Optional: if set is used by devices/format that do not do packet assembly inside hardware (i.e. W2K-1 Raw ASCII format)
	FastPacketAssembler nmea.Assembler
}

// NewBinaryDevice creates new instance of Actisense device using binary formats (NGT1 and N2K binary)
func NewBinaryDevice(reader io.ReadWriter) *BinaryFormatDevice {
	return NewBinaryDeviceWithConfig(reader, Config{ReceiveDataTimeout: 150 * time.Millisecond})
}

// NewBinaryDeviceWithConfig creates new instance of Actisense device using binary formats (NGT1 and N2K binary) with given config
func NewBinaryDeviceWithConfig(reader io.ReadWriter, config Config) *BinaryFormatDevice {
	if config.ReceiveDataTimeout > 0 {
		config.ReceiveDataTimeout = 5 * time.Second
	}
	return &BinaryFormatDevice{
		device:    reader,
		sleepFunc: time.Sleep,
		timeNow:   time.Now,
		config:    config,
	}
}

type state uint8

const (
	waitingStartOfMessage state = iota
	readingMessageData
	processingEscapeSequence
)

// ReadRawMessage reads raw data and parses it to nmea.RawMessage. This method block until full RawMessage is read or
// an error occurs (including context related errors).
func (d *BinaryFormatDevice) ReadRawMessage(ctx context.Context) (nmea.RawMessage, error) {
	// Actisense N2K binary message can be up to ISOTP size 1785
	message := make([]byte, nmea.ISOTPDataMaxSize)
	messageByteIndex := 0

	buf := make([]byte, 1)
	lastReadWithDataTime := d.timeNow()
	var previousByte byte
	var currentByte byte

	state := waitingStartOfMessage
	for {
		select {
		case <-ctx.Done():
			return nmea.RawMessage{}, ctx.Err()
		default:
		}

		n, err := d.device.Read(buf)
		// on read errors we do not return immediately as for:
		// os.ErrDeadlineExceeded - we set new deadline on next iteration
		// io.EOF - we check if already read + received is enough to form complete message
		if err != nil && !(errors.Is(err, os.ErrDeadlineExceeded) || errors.Is(err, io.EOF)) {
			return nmea.RawMessage{}, err
		}

		now := d.timeNow()
		if n == 0 {
			if errors.Is(err, io.EOF) && now.Sub(lastReadWithDataTime) > d.config.ReceiveDataTimeout {
				return nmea.RawMessage{}, err
			}
			continue
		}
		lastReadWithDataTime = now
		previousByte = currentByte
		currentByte = buf[0]

		switch state {
		case waitingStartOfMessage:
			if previousByte == DLE && currentByte == STX {
				state = readingMessageData
			}
		case readingMessageData:
			if currentByte == DLE {
				state = processingEscapeSequence
				break
			}
			message[messageByteIndex] = currentByte
			messageByteIndex++
		case processingEscapeSequence:
			if currentByte == DLE { // any DLE characters are double escaped (DLE DLE)
				state = readingMessageData
				message[messageByteIndex] = currentByte
				messageByteIndex++
				break
			}
			if currentByte == ETX { // end of message sequence
				msg := message[0:messageByteIndex]
				if d.config.DebugLogRawMessageBytes {
					fmt.Printf("# DEBUG read raw actisense binary message: %x\n", msg)
				}
				switch message[0] {
				case cmdNGTMessageReceived, cmdNGTMessageSend:
					return fromActisenseNGTBinaryMessage(msg, now)
				case cmdN2KMessageReceived, cmdN2KMessageSend:
					return fromActisenseN2KBinaryMessage(msg, now)
				case cmdRAWActisenseMessageReceived, cmdRAWActisenseMessageSend:
					return fromRawActisenseMessage(msg, now)
				case cmdDeviceMessageReceived:
					if d.config.OutputActisenseMessages {
						return fromNGTMessage(msg, now)
					}
				}
			}
			// when ignoreNGT1Messages or unknown DLE + ??? sequence - discard this current message and wait for next start sequence
			state = waitingStartOfMessage
			messageByteIndex = 0
		}
	}

}

func fromNGTMessage(raw []byte, now time.Time) (nmea.RawMessage, error) {
	// first 2 bytes for raw are command(@0) + len(@1)
	if len(raw) < (12 + 2) {
		return nmea.RawMessage{}, errors.New("raw message length too short to be valid BinaryFormatDevice message")
	}
	payloadLen := int(raw[1])
	if len(raw)-2 > payloadLen {
		payloadLen = int(raw[1])
		//return nmea.RawMessage{}, errors.New("raw message payload length does not match payload length")
	}
	dataBytes := make([]byte, payloadLen)
	copy(dataBytes, raw[2:payloadLen])

	return nmea.RawMessage{
		Time: now,
		Header: nmea.CanBusHeader{
			PGN:         CanBoatFakePGNOffset + uint32(dataBytes[0]),
			Source:      0,
			Destination: 0,
			Priority:    0,
		},
		Data: dataBytes,
	}, nil
}

func fromActisenseNGTBinaryMessage(raw []byte, now time.Time) (nmea.RawMessage, error) {
	length := len(raw) - 2 // 2 bytes for: command(raw[0]) + len(raw[1])
	data := raw[2:]
	if length < 11 {
		return nmea.RawMessage{}, errors.New("raw message length too short to be valid NMEA message")
	}

	const dataPartIndex = int(11)
	l := data[10]
	endIndex := dataPartIndex + int(l)
	if length != endIndex+1 {
		return nmea.RawMessage{}, fmt.Errorf("data length byte value is different from actual length, %v!=%v", l, length-dataPartIndex)
	}

	if err := crcCheck(raw); err != nil {
		return nmea.RawMessage{}, err
	}

	pgn := uint32(data[1]) + uint32(data[2])<<8 + uint32(data[3])<<16
	dataBytes := make([]byte, l)
	copy(dataBytes, data[dataPartIndex:endIndex])

	return nmea.RawMessage{
		Time: now,
		Header: nmea.CanBusHeader{
			PGN:         pgn,
			Source:      data[5],
			Destination: data[4],
			Priority:    data[0],
		},
		// NB: actisense ngt-1 seems to have some incrementing value for each message
		// ala 0x46f1ba15 -> 1190246933 -> 2007-09-20T03:08:53+03:00
		//Timestamp: uint32(data[6]) + uint32(data[7])<<8 + uint32(data[8])<<16 + uint32(data[9])<<24,
		Data: dataBytes,
	}, nil
}

func fromActisenseN2KBinaryMessage(raw []byte, now time.Time) (nmea.RawMessage, error) {
	// first 3 bytes are: 1 byte for message type, 2 bytes for rest of message length
	length := uint32(raw[1]) + uint32(raw[2])<<8
	if int(length)+1 != len(raw) {
		return nmea.RawMessage{}, errors.New("raw message length do not match actual data length")
	}

	dst := raw[3] // destination
	src := raw[4] // source

	dprp := raw[7]          // data page (1bit) + reserved (1bit) + priority bits (3bits)
	prio := (dprp >> 2) & 7 // priority bits are 3,4,5th bit
	rAndDP := dprp & 3      // data page + reserved is first 2 bits

	pduFormat := raw[6] // PF (PDU Format)
	pgn := uint32(rAndDP)<<16 + uint32(pduFormat)<<8
	if pduFormat >= 240 { // message is broadcast, PS contains group extension
		pgn += uint32(raw[5]) // +PS (PDU Specific)
	}
	//control := raw[8] // `PGN control ID bits and 3-bit Fast-Packet sequence ID` I do not know where this is useful.

	const dataPartIndex = int(13)
	dataBytes := make([]byte, len(raw)-dataPartIndex)
	copy(dataBytes, raw[dataPartIndex:])

	return nmea.RawMessage{
		Time: now,
		Header: nmea.CanBusHeader{
			PGN:         pgn,
			Source:      src,
			Destination: dst,
			Priority:    prio,
		},
		// NB: actisense n2k has (four bytes) for timestamp in milliseconds
		//Timestamp: uint32(raw[9]) + uint32(raw[10])<<8 + uint32(raw[11])<<16 + uint32(raw[12])<<24,
		Data: dataBytes,
	}, nil
}

// Example Send: `cansend can0 18EAFFFE#00EE00`
// Output from W2K RAW Actisense server: `95093eb7feffea1800ee0080`
//
// Message format:
// byte 0: command identifier
// byte 1: length of time counter + canid + data
// byte 2,3: time/counter
// byte 4,5,6,7: CanID (little endian)
// byte 8 ... (N-1): data
// byte N (last): CRC
func fromRawActisenseMessage(raw []byte, now time.Time) (nmea.RawMessage, error) {
	if len(raw) < 8 {
		return nmea.RawMessage{}, errors.New("raw actisense message length too short to be valid")
	}

	dLen := int(raw[1])
	if dLen+3 != len(raw) {
		return nmea.RawMessage{}, fmt.Errorf("data length byte value is different from actual length, %v!=%v", dLen, len(raw)-3)
	}

	if err := crcCheck(raw); err != nil {
		return nmea.RawMessage{}, err
	}

	CanID := nmea.ParseCANID(binary.LittleEndian.Uint32(raw[4:8]))
	dataBytes := make([]byte, dLen-6)
	copy(dataBytes, raw[8:len(raw)-1])

	return nmea.RawMessage{
		Time: now,
		Header: nmea.CanBusHeader{
			PGN:         CanID.PGN,
			Source:      CanID.Source,
			Destination: CanID.Destination,
			Priority:    CanID.Priority,
		},
		// NB: RAW actisense seems to have some incrementing value (2 bytes) for each message
		Data: dataBytes,
	}, nil
}

// crcCheck calculates and checks message checksum.
func crcCheck(data []byte) error {
	if crc(data) != 0 {
		return errors.New("raw message has invalid crc")
	}
	return nil
}

// crc calculates message checksum. CRC is such that the sum of all unescaped data bytes plus the command byte
// plus the length adds up to zero, modulo 256.
func crc(data []byte) uint8 {
	crc := uint16(0)
	for _, d := range data {
		dd := uint16(d)
		if crc+dd > 255 {
			crc = dd - (256 - crc)
			continue
		}
		crc = crc + dd
	}
	return uint8(crc)
}

// Initialize initializes connection to device. Otherwise BinaryFormatDevice will not send data.
//
// Canboat notes:
// The following startup command reverse engineered from Actisense NMEAreader.
// It instructs the BinaryFormatDevice to clear its PGN message TX list, thus it starts sending all PGNs.
//
// Actisense own documentation:
// Page 14: ACommsCommand_SetOperatingMode
// https://www.actisense.com/wp-content/uploads/2020/01/ActisenseComms-SDK-User-Manual-Issue-1.07-1.pdf
func (d *BinaryFormatDevice) Initialize() error {
	clearPGNFilter := []byte{ // `Receive All Transfer` Operating Mode
		cmdDeviceMessageSend, // Op code (NGT specific message)
		3,                    // length
		0x11,                 // msg byte 1, command `operating mode`
		0x02,                 // msg byte 2, argument 'receive all' (2 bytes)
		0x00,                 // msg byte 3
	}
	return d.writeBstMessage(clearPGNFilter)
}

func (d *BinaryFormatDevice) WriteRawMessage(ctx context.Context, msg nmea.RawMessage) error {
	if d.config.DebugLogRawMessageBytes {
		fmt.Printf("# DEBUG sending raw message: %+v\n", msg)
	}

	header := msg.Header

	dataLen := len(msg.Data)
	buf := make([]byte, dataLen+2+6)

	buf[0] = cmdNGTMessageSend // NGT1 device, NGT binary format
	if d.config.IsN2KWriter {
		buf[0] = cmdN2KMessageSend // W2K1 device, N2K Binary format
	}
	buf[1] = byte(dataLen + 6) // length

	buf[2] = header.Priority        // 1
	buf[3] = byte(header.PGN)       // 2
	buf[4] = byte(header.PGN >> 8)  // 3
	buf[5] = byte(header.PGN >> 16) // 4
	buf[6] = header.Destination     // 5
	buf[7] = byte(dataLen)          // 6
	copy(buf[8:], msg.Data)

	return d.writeBstMessage(buf)
}

func (d *BinaryFormatDevice) writeBstMessage(data []byte) error {
	packet := make([]byte, 0, len(data)+4+3) // 4 for prefix/suffix bytes and 3 for possible DLEs that need escaping
	packet = append(packet, DLE, STX)
	for _, b := range data {
		if b == DLE { // need to be escaped DLE => DLE, DLE
			packet = append(packet, DLE)
		}
		packet = append(packet, b)
	}
	crcByte := 0 - crc(data)
	packet = append(packet, crcByte, DLE, ETX)

	toWrite := len(packet)
	totalWritten := 0
	retryCount := 0
	maxRetry := 5

	if d.config.DebugLogRawMessageBytes {
		fmt.Printf("# DEBUG sent raw actisense binary message: %x\n", packet)
	}
	for {
		n, err := d.device.Write(packet)
		if err != nil {
			if !errors.Is(err, syscall.EAGAIN) {
				return fmt.Errorf("actisense write failure: %w", err)
			}
			retryCount++
		}
		totalWritten += n

		if totalWritten >= toWrite {
			break
		}
		if retryCount > maxRetry {
			return errors.New("actisense BinaryFormatDevice writes failed. retry count reached")
		}
		d.sleepFunc(250 * time.Millisecond)
	}
	return nil
}

func (d *BinaryFormatDevice) Close() error {
	if c, ok := d.device.(io.Closer); ok {
		return c.Close()
	}
	return errors.New("device does not implement Closer interface")
}
