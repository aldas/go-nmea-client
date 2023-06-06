package actisense

import (
	"context"
	"errors"
	"github.com/aldas/go-nmea-client"
	"io"
	"os"
	"time"
)

// EBL log file format used by Actisense W2K-1. Probably called "CAN-Raw (BST-95) message format"
// NGT1 ebl files are probably in different format.
//
// Example data frame from one EBL file:
// 1b 01 07 95 0e 28 9a 00 01 f8 09 3d 0d b3 22 48 32 59 0d 1b 0a
//
// 1b 01 <-- start of data frame (ESC+SOH)
//
//	07 95 <-- "95" is maybe row type. Actisense EBL Reader v2.027 says "now has added support for the new CAN-Raw (BST-95) message format that is used for all data logging on Actisense W2K-1"
//	     0e <-- lengths 14 bytes till end
//	       28 9a <-- timestamp 39464 (hex 9A28) (little endian)
//	            00 01 f8 09  <--- 0x09f80100 = src:0, dst:255, pgn:129025 (1f801), prio:2 (little endian)
//	                       3d 0d b3 22 48 32 59 0d <-- CAN payload (N2K endian rules), lat(32bit) 22b30d3d = 582159677, lon(32bit) 0d593248 = 223949384
//	                                               1b 0a <-- end of data frame (ESC+LF)
const (
	// SOH is start of data frame byte for Actisense BST-95 (EBL file created by Actisense W2K-1 device)
	SOH = 0x01
	// NL is end of data frame byte
	NL = 0x0A
	// ESC is marker byte before start/end data frame byte. Is sent before SOH or NL byte is sent (ESC+SOH or ESC+NL). Is escaped by sending double ESC+ESC characters.
	ESC = 0x1b
)

// EBLFormatDevice is implementing Actisense EBL file format
type EBLFormatDevice struct {
	device io.ReadWriter

	sleepFunc func(timeout time.Duration)
	timeNow   func() time.Time

	config Config
}

// NewEBLFormatDevice creates new instance of Actisense device using binary formats (NGT1 and N2K binary)
func NewEBLFormatDevice(reader io.ReadWriter) *EBLFormatDevice {
	return NewEBLFormatDeviceWithConfig(reader, Config{ReceiveDataTimeout: 150 * time.Millisecond})
}

// NewEBLFormatDeviceWithConfig creates new instance of Actisense device using binary formats (NGT1 and N2K binary) with given config
func NewEBLFormatDeviceWithConfig(reader io.ReadWriter, config Config) *EBLFormatDevice {
	if config.ReceiveDataTimeout > 0 {
		config.ReceiveDataTimeout = 5 * time.Second
	}
	return &EBLFormatDevice{
		device:    reader,
		sleepFunc: time.Sleep,
		timeNow:   time.Now,
		config:    config,
	}
}

// ReadRawMessage reads raw data and parses it to nmea.RawMessage. This method block until full RawMessage is read or
// an error occurs (including context related errors).
func (d *EBLFormatDevice) ReadRawMessage(ctx context.Context) (nmea.RawMessage, error) {
	// Actisense N2K binary message can be up to ISOTP size 1785
	message := make([]byte, nmea.ISOTPDataMaxSize)
	messageByteIndex := 0

	buf := make([]byte, 1)
	lastReadWithDataTime := d.timeNow()
	var previousByteWasEscape bool
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
		previousByteWasEscape = currentByte == ESC
		currentByte = buf[0]

		switch state {
		case waitingStartOfMessage: // start of message is (ESC + SOH)
			if previousByteWasEscape && currentByte == SOH {
				state = readingMessageData
			}
		case readingMessageData:
			if currentByte == ESC {
				state = processingEscapeSequence
				break
			}
			message[messageByteIndex] = currentByte
			messageByteIndex++
		case processingEscapeSequence:
			if currentByte == ESC { // any ESC characters are double escaped (ESC ESC)
				state = readingMessageData
				message[messageByteIndex] = currentByte
				messageByteIndex++
				break
			}
			if currentByte == NL { // end of message sequence (ESC + NL)
				if messageByteIndex-2 <= 2 {
					return nmea.RawMessage{}, errors.New("message too short to be BST95 format")
				}
				msg := message[0:messageByteIndex]
				if d.config.DebugLogRawMessageBytes && d.config.LogFunc != nil {
					d.config.LogFunc("# DEBUG read raw actisense ELB message: %x\n", msg)
				}
				if msg[0] == 0x7 && msg[1] == cmdRAWActisenseMessageReceived { // 0x07+0x95 seems to identify BST-95 message
					return fromActisenseBST95Message(msg[2:], now)
				}
				if d.config.LogFunc != nil {
					d.config.LogFunc("# ERROR unknown message type read: %x\n", msg)
				}
			}
			// when unknown ESC + ??? sequence - discard this current message and wait for next start sequence
			state = waitingStartOfMessage
			messageByteIndex = 0
		}
	}

}

func fromActisenseBST95Message(raw []byte, now time.Time) (nmea.RawMessage, error) {
	const startOfData = 7 // length(1) + timestamp(2) + canid(4) = 7
	if len(raw) < 8 {     // startOfData + min length of data (1)
		return nmea.RawMessage{}, errors.New("raw message actual length too short to be valid BST-95 message")
	}
	if int(raw[0]) != len(raw)-1 {
		return nmea.RawMessage{}, errors.New("raw message length field does not match actual length")
	}

	canID := uint32(raw[3]) + uint32(raw[4])<<8 + uint32(raw[5])<<16 + uint32(raw[6])<<24

	dataBytes := make([]byte, len(raw)-startOfData)
	copy(dataBytes, raw[startOfData:])

	return nmea.RawMessage{
		Time:   now,
		Header: nmea.ParseCANID(canID),
		// W2K-1 seems to use some kind of (offset) counter for timestamp. Probably some other message type in beginning
		// of the EBL file has "start" time for that file to which this timestamp offset should be added to.
		//Timestamp: uint16(raw[1]) + uint16(raw[2]) << 8,
		Data: dataBytes,
	}, nil
}

// Initialize initializes connection to device. Otherwise BinaryFormatDevice will not send data.
func (d *EBLFormatDevice) Initialize() error {
	return nil
}

func (d *EBLFormatDevice) WriteRawMessage(ctx context.Context, msg nmea.RawMessage) error {
	return nil
}

func (d *EBLFormatDevice) Close() error {
	if c, ok := d.device.(io.Closer); ok {
		return c.Close()
	}
	return errors.New("device does not implement Closer interface")
}
