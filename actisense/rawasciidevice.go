package actisense

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/aldas/go-nmea-client"
	"io"
	"math"
	"time"
)

const rawASCIIDelimiter = ' '

// RawASCIIDevice is implementing Actisense W2K-1 device capable of decoding RAW Ascii format
type RawASCIIDevice struct {
	device  io.ReadWriter
	timeNow func() time.Time

	readBuffer []byte
	readIndex  int

	DebugLogRawFrameBytes bool
}

// NewRawASCIIDevice creates new instance of Actisense W2K-1 device capable of decoding RAW Ascii format. RAW ASCII
// format is ordinary Canbus frame with 8 bytes of data so fast-packet and multi-packet (ISO TP) assembly must be done
// separately.
func NewRawASCIIDevice(reader io.ReadWriter) *RawASCIIDevice {
	return &RawASCIIDevice{
		device:     reader,
		timeNow:    time.Now,
		readBuffer: make([]byte, 100),
	}
}

func (d *RawASCIIDevice) Close() error {
	if c, ok := d.device.(io.Closer); ok {
		return c.Close()
	}
	return errors.New("device does not implement Closer interface")
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
		// if end of line is found we copy data that we just read to previously read data to assemble full line
		copy(d.readBuffer[d.readIndex:], buf[0:endIndex]) // note: \n is not included
		d.readIndex += endIndex

		frame := d.readBuffer[0:d.readIndex]
		if d.DebugLogRawFrameBytes {
			fmt.Printf("# DEBUG Actisense RAW ASCII frame: %x\n", frame)
		}
		now := d.timeNow()
		rawFrame, skip, err := parseRawASCII(frame, now)

		// reset read buffer to whatever we were able to read past current frame end. probably nothing but could be
		// start of next frame etc
		copy(d.readBuffer, buf[endIndex+1:n])
		d.readIndex = n - (endIndex + 1)

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
