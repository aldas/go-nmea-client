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

	// cmdN2KMessageReceived identifies that packet is received/incoming NMEA200 data message.
	cmdN2KMessageReceived = 0x93
	// cmdN2KMessageRequestReceived identifies that packet is sent/outgoing NMEA200 data message.
	cmdN2KMessageSend = 0x94
	// cmdNGTMessageReceived identifies that received packet is (BEMCMD) Actisense NGT specific message
	cmdNGTMessageReceived = 0xA0
	// cmdNGTMessageSend identifies that sent packet is Actisense NGT specific message
	cmdNGTMessageSend = 0xA1

	// CanBoatFakePGNOffset is offset for PGNs that Actisense devices create for their own information. We add it to
	// parsed PGN and after that we can find match from Canboat PGN database with that
	CanBoatFakePGNOffset uint32 = 0x40000
)

// NGT1 is implementing Actisense NGT-1 device
type NGT1 struct {
	ignoreNGT1Messages bool
	device             io.ReadWriter

	sleepFunc func(timeout time.Duration)
	timeNow   func() time.Time
	// receiveDataTimeout is to limit amount of time reads can result no data. to timeout the connection when there is no
	// interaction in bus. This is different from for example serial device readTimeout which limits how much time Read
	// call blocks but we want to Reads block small amount of time to be able to check if context was cancelled during read
	// but at the same time we want to be able to detect when there are no coming from bus for excessive amount of time.
	receiveDataTimeout time.Duration

	DebugLogRawMessageBytes bool
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
}

// NewNGT1Device creates new instance of Actisense NGT-1 device
func NewNGT1Device(reader io.ReadWriter) *NGT1 {
	return NewNGT1DeviceWithConfig(reader, Config{ReceiveDataTimeout: 150 * time.Millisecond})
}

// NewNGT1DeviceWithConfig creates new instance of Actisense NGT-1 device with given config
func NewNGT1DeviceWithConfig(reader io.ReadWriter, config Config) *NGT1 {
	device := &NGT1{
		device: reader,
		sleepFunc: func(timeout time.Duration) {
			time.Sleep(timeout)
		},
		timeNow:            time.Now,
		receiveDataTimeout: 5 * time.Second,
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

// ReadRawMessage reads raw USB data and parses it to nmea.RawMessage. This method block until full RawMessage is read or
// an error occurs (including context related errors).
func (d *NGT1) ReadRawMessage(ctx context.Context) (nmea.RawMessage, error) {
	message := make([]byte, nmea.FastRawPacketMaxSize) // TODO: how large it should be? https://en.wikipedia.org/wiki/NMEA_2000 see sizes
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
				if d.DebugLogRawMessageBytes {
					fmt.Printf("# DEBUG raw actisense binary message: %x\n", message[0:messageByteIndex])
				}
				switch message[0] {
				case cmdN2KMessageReceived, cmdN2KMessageSend:
					return fromActisenseBinaryMessage(message[0:messageByteIndex], now)
				case cmdNGTMessageReceived:
					if !d.ignoreNGT1Messages {
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
		return nmea.RawMessage{}, errors.New("raw message length too short to be valid NGT1 message")
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

func fromActisenseBinaryMessage(raw []byte, now time.Time) (nmea.RawMessage, error) {
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

// Initialize initializes connection to device. Otherwise NGT1 will not send data.
//
// Canboat notes:
// The following startup command reverse engineered from Actisense NMEAreader.
// It instructs the NGT1 to clear its PGN message TX list, thus it starts sending all PGNs.
func (d *NGT1) Initialize() error {
	// Page 14: ACommsCommand_SetOperatingMode
	// https://www.actisense.com/wp-content/uploads/2020/01/ActisenseComms-SDK-User-Manual-Issue-1.07-1.pdf
	clearPGNFilter := []byte{ // `Receive All Transfer` Operating Mode
		cmdNGTMessageSend, // NGT specific message
		3,                 // length
		0x11,              // msg byte 1, meaning `operating mode`
		0x02,              // msg byte 2, meaning 'receive all' (2 bytes)
		0x00,              // msg byte 3
	}
	return d.write(clearPGNFilter)
}

func (d *NGT1) write(data []byte) error {
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
			return errors.New("actisense NGT1 writes failed. retry count reached")
		}
		d.sleepFunc(250 * time.Millisecond)
	}
	return nil
}

func (d *NGT1) Close() error {
	if c, ok := d.device.(io.Closer); ok {
		return c.Close()
	}
	return errors.New("device does not implement Closer interface")
}
