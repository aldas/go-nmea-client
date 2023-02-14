package canboat

import (
	"bufio"
	"context"
	"github.com/aldas/go-nmea-client"
	"io"
	"strings"
)

type Device struct {
	reader  io.Reader
	scanner *bufio.Scanner
}

func NewCanBoatReader(reader io.Reader) *Device {
	return &Device{
		reader:  reader,
		scanner: bufio.NewScanner(reader),
	}
}

func (d *Device) Initialize() error {
	return nil // do nothing
}

func (d *Device) ReadRawMessage(ctx context.Context) (nmea.RawMessage, error) {
	for d.scanner.Scan() {
		line := strings.TrimSpace(d.scanner.Text())
		if line == "" || line[0] == '#' {
			continue
		}
		return UnmarshalString(line)
	}
	if err := d.scanner.Err(); err != nil {
		return nmea.RawMessage{}, err
	}
	return nmea.RawMessage{}, io.EOF
}

func (d *Device) Write(nmea.RawMessage) error {
	return nil // do nothing
}

func (d *Device) Close() error {
	closer, ok := d.reader.(io.Closer)
	if ok {
		return closer.Close()
	}
	return nil
}
