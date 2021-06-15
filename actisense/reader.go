package actisense

import (
	"context"
	"errors"
	"fmt"
	"github.com/aldas/go-nmea-client"
	"io"
	"syscall"
	"time"
)

/* Actisense message structure is:

	const STX = 0x02  // Start packet
	const ETX = 0x03  // End packet
	const DLE = 0x10  // Start pto encode a STX or ETX send DLE+STX or DLE+ETX
	const ESC = 0x1B  // Escape

   DLE STX <command> <len> [<data> ...]  <checksum> DLE ETX

   <command> is a byte from the list below.
   In <data> any DLE characters are double escaped (DLE DLE).
   <len> encodes the unescaped length.
   <checksum> is such that the sum of all unescaped data bytes plus the command
              byte plus the length adds up to zero, modulo 256.

	const N2K_MSG_RECEIVED = 0x93  // Receive standard N2K message
	const N2K_MSG_SEND     = 0x94  // Send N2K message
	const NGT_MSG_RECEIVED = 0xA0  // Receive NGT specific message
	const NGT_MSG_SEND     = 0xA1  // Send NGT message

*/

const (
	// STX start packet byte for Actisense parsed NMEA2000 packet
	STX = 0x02
	// ETX end packet byte for Actisense parsed NMEA2000 packet
	ETX = 0x03
	// DLE marker byte before start/end packet byte. Is sent before STX or ETX byte is sent (DLE+STX or DLE+ETX)
	DLE = 0x10
)

const (
	// cmdN2KMessageReceived identifies that packet is NMEA200 message
	cmdN2KMessageReceived = 0x93
	// cmdNGTMessageReceived identifies that packet is Actisense NGT specific message
	cmdNGTMessageReceived = 0xA0
)

// NGT1 is implementing Actisense NGT-1 device
type NGT1 struct {
	ignoreNGT1Messages bool
	device             io.ReadWriter

	sleepFunc func(timeout time.Duration)
	timeNow   func() time.Time

	DebugLogRawMessageBytes bool
}

// NewNGT1Device creates new instance of Actisense NGT-1 device
func NewNGT1Device(reader io.ReadWriter) *NGT1 {
	return &NGT1{
		device: reader,
		sleepFunc: func(timeout time.Duration) {
			time.Sleep(timeout)
		},
		timeNow: time.Now,
	}
}

type state uint8

const (
	waitingStartOfMessage state = iota
	readingMessageData
	processingEscapeSequence
)

// ReadRawMessage reads raw USB data and parses it to nmea.RawMessage
func (d *NGT1) ReadRawMessage(ctx context.Context) (nmea.RawMessage, error) {
	message := make([]byte, nmea.FastRawPacketMaxSize) // TODO: how large it should be? https://en.wikipedia.org/wiki/NMEA_2000 see sizes
	messageByteIndex := 0

	buf := make([]byte, 1)
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

		if err != nil {
			return nmea.RawMessage{}, err
		}
		if n == 0 {
			// return???
			continue
		}
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
					fmt.Printf("# DEBUG raw actisense message: %x\n", message[0:messageByteIndex])
				}
				now := d.timeNow()
				switch message[0] {
				case cmdN2KMessageReceived:
					return fromNmea2000Message(message[0:messageByteIndex], now)
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
	// FIXME: should we include PGN list transfer logic?
	if raw[0] == 0x11 {
		return nmea.RawMessage{}, nil
	}
	return fromNmea2000Message(raw, now)
}

func fromNmea2000Message(raw []byte, now time.Time) (nmea.RawMessage, error) {
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
	dataBytes := data[dataPartIndex:b]
	return nmea.RawMessage{
		Priority:    data[0],
		PGN:         pgn,
		Destination: data[4],
		Source:      data[5],
		// NB: actisense ngt-1 seems to have some none-sense unixtimestamps. incrementing value for each message???
		// ala 0x46f1ba15 -> 1190246933 -> 2007-09-20T03:08:53+03:00
		Timestamp: uint32(data[6]) + uint32(data[7])<<8 + uint32(data[8])<<16 + uint32(data[9])<<24,
		Length:    l,
		Data:      dataBytes,
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
// The following startup command reverse engineered from Actisense NMEAreader.
// It instructs the NGT1 to clear its PGN message TX list, thus it starts sending all PGNs.
func (d *NGT1) Initialize() error {
	clearPGNFilter := []byte{
		0xA1, // command
		3,    // length
		0x11, // msg byte 1, meaning ?
		0x02, // msg byte 2, meaning ?
		0x00, // msg byte 3, meaning ?
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
