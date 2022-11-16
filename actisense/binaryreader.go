package actisense

import (
	"context"
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

	// cmdN2KMessageReceived identifies that packet is received/incoming NMEA200 data message as N2K binary format.
	cmdN2KMessageReceived = 0xD0

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
	// receiveDataTimeout is to limit amount of time reads can result no data. to timeout the connection when there is no
	// interaction in bus. This is different from for example serial device readTimeout which limits how much time Read
	// call blocks but we want to Reads block small amount of time to be able to check if context was cancelled during read
	// but at the same time we want to be able to detect when there are no coming from bus for excessive amount of time.
	receiveDataTimeout time.Duration

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

	// DebugLogRawMessageBytes instructs device to log all received raw messages
	DebugLogRawMessageBytes bool
	// OutputActisenseMessages instructs device to output Actisense own messages
	OutputActisenseMessages bool
}

// NewBinaryDevice creates new instance of Actisense device using binary formats (NGT1 and N2K binary)
func NewBinaryDevice(reader io.ReadWriter) *BinaryFormatDevice {
	return NewBinaryDeviceWithConfig(reader, Config{ReceiveDataTimeout: 150 * time.Millisecond})
}

// NewBinaryDeviceWithConfig creates new instance of Actisense device using binary formats (NGT1 and N2K binary) with given config
func NewBinaryDeviceWithConfig(reader io.ReadWriter, config Config) *BinaryFormatDevice {
	device := &BinaryFormatDevice{
		device: reader,
		sleepFunc: func(timeout time.Duration) {
			time.Sleep(timeout)
		},
		timeNow:            time.Now,
		receiveDataTimeout: 5 * time.Second,
		config:             config,
	}
	if config.ReceiveDataTimeout > 0 {
		device.receiveDataTimeout = config.ReceiveDataTimeout
	}

	return device
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
			if errors.Is(err, io.EOF) && now.Sub(lastReadWithDataTime) > d.receiveDataTimeout {
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
				if d.config.DebugLogRawMessageBytes {
					fmt.Printf("# DEBUG raw actisense binary message: %x\n", message[0:messageByteIndex])
				}
				switch message[0] {
				case cmdNGTMessageReceived, cmdNGTMessageSend:
					return fromActisenseNGTBinaryMessage(message[0:messageByteIndex], now)
				case cmdN2KMessageReceived:
					return fromActisenseN2KBinaryMessage(message[0:messageByteIndex], now)
				case cmdDeviceMessageReceived:
					if d.config.OutputActisenseMessages {
						return fromNGTMessage(message[0:messageByteIndex], now)
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
	length := len(raw) - 2 // 2 bytes for: command(@0) + len(@1)
	data := raw[2:]

	const dataPartIndex = int(11)
	l := data[10]
	if length < 11 || length < dataPartIndex+int(l) {
		return nmea.RawMessage{}, errors.New("raw message length too short to be valid NMEA message")
	}

	if err := crcCheck(raw); err != nil {
		return nmea.RawMessage{}, err
	}

	pgn := uint32(data[1]) + uint32(data[2])<<8 + uint32(data[3])<<16
	b := dataPartIndex + int(l)
	dataBytes := make([]byte, l)
	copy(dataBytes, data[dataPartIndex:b])

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
func (d *BinaryFormatDevice) Initialize() error {
	// Page 14: ACommsCommand_SetOperatingMode
	// https://www.actisense.com/wp-content/uploads/2020/01/ActisenseComms-SDK-User-Manual-Issue-1.07-1.pdf
	clearPGNFilter := []byte{ // `Receive All Transfer` Operating Mode
		cmdDeviceMessageSend, // NGT specific message
		3,                    // length
		0x11,                 // msg byte 1, meaning `operating mode`
		0x02,                 // msg byte 2, meaning 'receive all' (2 bytes)
		0x00,                 // msg byte 3
	}
	return d.write(clearPGNFilter)
}

func (d *BinaryFormatDevice) write(data []byte) error {
	packet := append([]byte{DLE, STX}, data...)
	crcByte := 0 - crc(data)
	packet = append(packet, []byte{crcByte, DLE, ETX}...)

	toWrite := len(packet)
	totalWritten := 0
	retryCount := 0
	maxRetry := 5
	for {
		n, err := d.device.Write(packet)
		if err != nil {
			if !errors.Is(err, syscall.EAGAIN) {
				return fmt.Errorf("actisense initialization write failure: %w", err)
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
