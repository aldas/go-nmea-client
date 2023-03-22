package actisense

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/aldas/go-nmea-client"
	"github.com/aldas/go-nmea-client/internal/utils"
	"io"
	"math"
	"strconv"
	"strings"
	"time"
)

const rawASCIIDelimiter = ' '

// RawASCIIDevice is implementing Actisense W2K-1 device capable of decoding RAW Ascii format
type RawASCIIDevice struct {
	device  io.ReadWriter
	timeNow func() time.Time

	readBuffer []byte
	readIndex  int

	config Config
}

// NewRawASCIIDevice creates new instance of Actisense W2K-1 device capable of decoding RAW Ascii format. RAW ASCII
// format is ordinary Canbus frame with 8 bytes of data so fast-packet and multi-packet (ISO TP) assembly must be done
// separately.
func NewRawASCIIDevice(reader io.ReadWriter, config Config) *RawASCIIDevice {
	return &RawASCIIDevice{
		device:     reader,
		timeNow:    time.Now,
		readBuffer: make([]byte, 100),
		config:     config,
	}
}

func (d *RawASCIIDevice) Close() error {
	if c, ok := d.device.(io.Closer); ok {
		return c.Close()
	}
	return errors.New("device does not implement Closer interface")
}

func (d *RawASCIIDevice) Initialize() error {
	return nil // no-op
}

const hextable = "0123456789ABCDEF"

func toRawASCIIBytes(frame nmea.RawFrame) []byte {
	canID := frame.Header.Uint32()
	f := []byte{
		// example: `00:00:00.000 S 1F223355 01 02 03 04 05 06 07 08\n`
		0x30, 0x30, 0x3a, 0x30, 0x30, 0x3a, 0x30, 0x30, 0x2e, 0x30, 0x30, 0x30, 0x20, 0x53, 0x20, // `00:00:00.000 S ` (0-14)
		0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, // canID part `1F223355` (15-22)
		0x20, 0x0, 0x0, 0x20, 0x0, 0x0, 0x20, 0x0, 0x0, // ` 01 02 03` (23-31)
		0x20, 0x0, 0x0, 0x20, 0x0, 0x0, 0x20, 0x0, 0x0, // ` 04 05 06` (32-40)
		0x20, 0x0, 0x0, 0x20, 0x0, 0x0, 0x0d, 0x0a, // ` 07 08\r\n` (41-48)
	}
	hexCanID := strings.ToUpper(strconv.FormatUint(uint64(canID), 16))
	canIDStart := 23 - len(hexCanID)
	for i, s := range hexCanID {
		f[canIDStart+i] = byte(s)
	}

	idx := uint8(24)
	for i := uint8(0); i < frame.Length; i++ {
		v := frame.Data[i]
		f[idx] = hextable[v>>4]
		f[idx+1] = hextable[v&0x0f]
		idx += 3 // additional byte is for space (0x20)
	}
	if frame.Length < 8 {
		// `\r\n` at the end
		f[idx-1] = 0x0d
		f[idx] = 0x0a
	}
	return f[0 : idx+1]
}

func (d *RawASCIIDevice) WriteRawFrame(ctx context.Context, frame nmea.RawFrame) error {
	rawB := toRawASCIIBytes(frame)
	if d.config.DebugLogRawMessageBytes {
		fmt.Printf("# DEBUG Writing Actisense N2K RAW ASCII bytes: `%v`\n", utils.FormatSpaces(rawB))
	}
	_, err := d.device.Write(rawB)
	return err
}

func (d *RawASCIIDevice) WriteRawMessage(ctx context.Context, msg nmea.RawMessage) error {
	dLen := uint8(len(msg.Data))
	if len(msg.Data) > 8 {
		panic("message longer than 8 bytes")
	}
	frame := nmea.RawFrame{
		Time:   msg.Time,
		Header: msg.Header,
		Length: dLen,
		Data:   [8]byte{},
	}
	copy(frame.Data[0:], msg.Data)
	return d.WriteRawFrame(ctx, frame)
}

func (d *RawASCIIDevice) assembleRawMessage(ctx context.Context) (nmea.RawMessage, error) {
	msg := nmea.RawMessage{}
	for {
		frame, err := d.ReadRawFrame(ctx)
		if err != nil {
			return nmea.RawMessage{}, err
		}
		if d.config.FastPacketAssembler.Assemble(frame, &msg) {
			return msg, nil
		}
	}
}

func (d *RawASCIIDevice) ReadRawMessage(ctx context.Context) (nmea.RawMessage, error) {
	if d.config.FastPacketAssembler != nil {
		return d.assembleRawMessage(ctx)
	}

	frame, err := d.ReadRawFrame(ctx)
	if err != nil {
		return nmea.RawMessage{}, err
	}
	return nmea.RawMessage{
		Time:   frame.Time,
		Header: frame.Header,
		Data:   frame.Data[:],
	}, err
}

func (d *RawASCIIDevice) ReadRawFrame(ctx context.Context) (nmea.RawFrame, error) {
	// Example: '00:34:02.718 R 15FD0800 FF 00 01 CA 6F FF FF FF\n'
	buf := make([]byte, 50)

	for {
		select {
		case <-ctx.Done():
			return nmea.RawFrame{}, ctx.Err()
		default:
		}

		n, err := d.device.Read(buf) // FIXME: read is blocking call. we need to set read timeouts to work with context cancellations

		if err != nil {
			return nmea.RawFrame{}, err
		}
		if n == 0 {
			// return???
			continue
		}

		endIndex := bytes.IndexByte(buf[0:n], '\n')
		if endIndex == -1 { // no end of line seen. add this line to buff and try reading more
			copy(d.readBuffer[d.readIndex:], buf[0:n])
			d.readIndex += n

			continue
		}
		endIndex++ // note: include \n
		// if end of line is found we copy data that we just read to previously read data to assemble full line
		copy(d.readBuffer[d.readIndex:], buf[0:endIndex])
		d.readIndex += endIndex

		frame := d.readBuffer[0:d.readIndex]
		if d.config.DebugLogRawMessageBytes {
			fmt.Printf("# DEBUG Read Actisense RAW ASCII frame: %v\n", utils.FormatSpaces(frame))
		}
		now := d.timeNow()
		rawFrame, skip, err := parseRawASCII(frame, now)

		// reset read buffer to whatever we were able to read past current frame end. probably nothing but could be
		// start of next frame etc
		copy(d.readBuffer, buf[endIndex:n])
		d.readIndex = n - endIndex

		if skip {
			continue
		}

		return rawFrame, err
	}
}

func parseRawASCII(raw []byte, now time.Time) (nmea.RawFrame, bool, error) {
	// Example: '00:34:02.718 R 15FD0800 FF 00 01 CA 6F FF FF FF\n'
	//                       1 2        3  4  5  6  7  8  9  0
	// I do not have documentation for RAW ASCII format so compared to N2K ASCII format we do this in more naive way
	// We will find 2 and 3rd spaces so we can check for "R" meaning frame is received and parse CANID to PGN etc
	// and then decode hex to bytes everything after CanID block
	spacesSeen := 0
	spaceIndex := 0
	previousSpaceIndex := 0
	for i, b := range raw {
		if b != rawASCIIDelimiter {
			continue
		}
		previousSpaceIndex = spaceIndex
		spaceIndex = i
		spacesSeen++
		if spacesSeen == 3 {
			break
		}
	}
	if spacesSeen != 3 { // skippable - this is probably some garbage from the wire, or we started reading frame not from the beginning
		return nmea.RawFrame{}, true, errors.New("failed to find correct space index in raw ascii frame")
	}
	if raw[previousSpaceIndex-1] != 'R' { // skippable - this is not received frame
		return nmea.RawFrame{}, true, errors.New("raw ascii frame does not seem to be received frame")
	}

	var CanID uint32
	if err := decodeHexToInt(raw[previousSpaceIndex+1:spaceIndex], &CanID, 4); err != nil {
		return nmea.RawFrame{}, false, err
	}
	canHeader := nmea.ParseCANID(CanID)

	hexBytes := make([]byte, 16)
	dstIndex := 0
	for i := spaceIndex; i < len(raw); i++ {
		b := raw[i]
		if b == rawASCIIDelimiter {
			continue
		}
		if b == '\r' || b == '\n' {
			break
		}
		hexBytes[dstIndex] = b
		dstIndex++
	}
	dataDecoded := make([]byte, dstIndex)
	n, err := hex.Decode(dataDecoded, hexBytes)
	if err != nil {
		return nmea.RawFrame{}, false, err
	}
	data := [8]byte{}
	copy(data[:], dataDecoded[:n])

	return nmea.RawFrame{
		Time:   now,
		Header: canHeader,
		Length: uint8(n),
		Data:   data,
	}, false, nil
}

func decodeHexToInt(raw []byte, target interface{}, dstLength int) error {
	dst := make([]byte, dstLength)

	diffInBytes := dstLength - int(math.Ceil(float64(len(raw))/2))
	if diffInBytes != 0 {
		tmp := make([]byte, dstLength*2)
		start := (dstLength * 2) - len(raw)
		for i := 0; i < start; i++ {
			tmp[i] = '0'
		}
		copy(tmp[start:], raw)
		raw = tmp
	}

	_, err := hex.Decode(dst, raw)
	if err != nil {
		return err
	}

	buffer := bytes.NewReader(dst)
	err = binary.Read(buffer, binary.BigEndian, target)
	return err
}
