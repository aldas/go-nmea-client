package actisense

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/aldas/go-nmea-client"
	"io"
	"time"
)

type RawMessageReader interface {
	ReadRawMessage(ctx context.Context) (nmea.RawMessage, error)
	Initialize() error
}

type RawMessageWriter interface {
	Write(nmea.RawMessage) error
}

type RawMessageReaderWriter interface {
	RawMessageReader
	RawMessageWriter
}

// N2kASCIIDevice is implementing Actisense W2K-1 device capable of decoding NMEA 2000 Ascii format including
// fast-packet and multi-packet (ISO TP) messages
//
// Note: is not go-routine safe
type N2kASCIIDevice struct {
	device  io.ReadWriter
	timeNow func() time.Time

	readBuffer []byte
	readIndex  int

	config Config
}

// NewN2kASCIIDevice creates new instance of Actisense W2K-1 device capable of decoding NMEA 2000 Ascii format
func NewN2kASCIIDevice(reader io.ReadWriter, config Config) *N2kASCIIDevice {
	return &N2kASCIIDevice{
		device:     reader,
		timeNow:    time.Now,
		readBuffer: make([]byte, nmea.ISOTPDataMaxSize*2),

		config: config,
	}
}

func (d *N2kASCIIDevice) Close() error {
	if c, ok := d.device.(io.Closer); ok {
		return c.Close()
	}
	return errors.New("device does not implement Closer interface")
}

func (d *N2kASCIIDevice) Write(nmea.RawMessage) error {
	return errors.New("not implemented")
}

func (d *N2kASCIIDevice) Initialize() error {
	return nil
}

func (d *N2kASCIIDevice) ReadRawMessage(ctx context.Context) (nmea.RawMessage, error) {
	// Example: 'A173321.107 23FF7 1F513 012F3070002F30709F  \n'
	buf := make([]byte, nmea.FastRawPacketMaxSize+100)

	for {
		select {
		case <-ctx.Done():
			return nmea.RawMessage{}, ctx.Err()
		default:
		}

		n, err := d.device.Read(buf) // FIXME: read is blocking call. we need to set read timeouts to work with context cancellations

		if err != nil {
			return nmea.RawMessage{}, err
		}
		if n == 0 {
			// return???
			continue
		}

		messageEndIndex := bytes.IndexByte(buf[0:n], '\n')
		if messageEndIndex == -1 { // no end of message seen. add this line to buff and try reading more
			copy(d.readBuffer[d.readIndex:], buf[0:n])
			d.readIndex += n

			continue
		}
		// if end of message is found we copy data that we just read to previously read data to assemble full message
		copy(d.readBuffer[d.readIndex:], buf[0:messageEndIndex]) // note: \n is not included
		d.readIndex += messageEndIndex

		message := d.readBuffer[0:d.readIndex]
		if d.config.DebugLogRawMessageBytes {
			fmt.Printf("# DEBUG Actisense N2K ASCII message: %x\n", message)
		}
		now := d.timeNow()
		rawMessage, skip, err := parseN2KAscii(message, now)

		// reset read buffer to whatever we were able to read past current message end. probably nothing but could be
		// start of next message etc
		copy(d.readBuffer, buf[messageEndIndex+1:n])
		d.readIndex = n - (messageEndIndex + 1)

		if skip {
			continue
		}

		return rawMessage, err
	}
}

func parseN2KAscii(raw []byte, now time.Time) (nmea.RawMessage, bool, error) {
	// Source: Actisense own documentation `NMEA 2000 ASCII Output format.docx`
	//
	// Ahhmmss.ddd <SS><DD><P> <PPPPP> b0b1b2b3b4b5b6b7.....bn<CR><LF>
	// A = message is N2K or J1939 message
	// 173321.107 - time 17:33:21.107
	// <SS> - source address
	// <DD> - destination address
	// <P> - priority
	// <PPPPP> - PGN number
	// b0b1b2b3b4b5b6b7.....bn - data payload in hex. NB: ISO TP payload could be up to 1786 bytes
	//
	// Example: `A173321.107 23FF7 1F513 012F3070002F30709F\n`
	//                      1     2     3                  4

	if raw[0] != 'A' {
		return nmea.RawMessage{}, true, errors.New("N2K Ascii message should start with A")
	}
	if len(raw) < 22 { // shortest message: 1 bytes of data and time is with second precision
		return nmea.RawMessage{}, true, errors.New("N2K Ascii message too short to be valid message")
	}

	timePartEnd := 0
	for i := 1; i < len(raw); i++ {
		b := raw[i]
		if !('0' <= b && b <= '9') && b != '.' {
			break
		}
		timePartEnd = i
	}
	if timePartEnd == 0 {
		return nmea.RawMessage{}, false, errors.New("N2K Ascii message missing time block")
	}

	headerPartStart, headerPartEnd := findNextNonHexBlock(raw, timePartEnd+1)
	if headerPartEnd == -1 {
		return nmea.RawMessage{}, false, errors.New("N2K Ascii message missing source,destination,priority block")
	}

	var source uint8
	if err := decodeHexToInt(raw[headerPartStart:headerPartStart+2], &source, 1); err != nil {
		return nmea.RawMessage{}, false, fmt.Errorf("N2K Ascii message to decode source, err: %v", err)
	}
	var destination uint8
	if err := decodeHexToInt(raw[headerPartStart+2:headerPartStart+4], &destination, 1); err != nil {
		return nmea.RawMessage{}, false, fmt.Errorf("N2K Ascii message to decode destination, err: %v", err)
	}
	priority := raw[headerPartStart+4] - '0'

	pgnPartStart, pgnPartEnd := findNextNonHexBlock(raw, headerPartEnd+1)
	if pgnPartEnd == -1 {
		return nmea.RawMessage{}, false, errors.New("N2K Ascii message missing source,destination,priority block")
	}
	var pgn uint32
	if err := decodeHexToInt(raw[pgnPartStart:pgnPartEnd+1], &pgn, 4); err != nil {
		return nmea.RawMessage{}, false, fmt.Errorf("N2K Ascii message to decode PGN, err: %v", err)
	}

	dataPartStart, dataPartEnd := findNextNonHexBlock(raw, pgnPartEnd+1)
	if dataPartEnd == -1 {
		return nmea.RawMessage{}, false, errors.New("N2K Ascii message missing data block")
	}
	dataDecoded := make([]byte, (dataPartEnd+1-dataPartStart)/2)
	n, err := hex.Decode(dataDecoded, raw[dataPartStart:dataPartEnd+1])
	if err != nil {
		return nmea.RawMessage{}, false, err
	}
	dataDecoded = dataDecoded[0:n]

	return nmea.RawMessage{
		Time: now,
		Header: nmea.CanBusHeader{
			PGN:         pgn,
			Source:      source,
			Destination: destination,
			Priority:    priority,
		},
		Data: dataDecoded,
	}, false, nil
}

func findNextNonHexBlock(raw []byte, fromIndex int) (int, int) {
	startIndex := -1
	endIndex := -1
	for i := fromIndex; i < len(raw); i++ {
		isHex := isHexChar(raw[i])
		if !isHex {
			if endIndex == -1 {
				continue // skip forward until we find hex block start
			}
			break // hex block has ended
		}
		if startIndex == -1 {
			startIndex = i
		}
		endIndex = i
	}
	return startIndex, endIndex
}

func isHexChar(c byte) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}
