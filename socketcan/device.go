package socketcan

import (
	"context"
	"errors"
	"github.com/aldas/go-nmea-client"
	"time"
)

type DeviceConfig struct {
	// InterfaceName is SocketCAN interface name.
	// Defaults to: can0
	InterfaceName string

	// ReceiveDataTimeout is to limit amount of time reads can result no data. to timeout the connection when there is no
	// interaction in bus. This is different from for example serial device readTimeout which limits how much time Read
	// call blocks but we want to Reads block small amount of time to be able to check if context was cancelled during read
	// but at the same time we want to be able to detect when there are no coming from bus for excessive amount of time.
	// Defaults to: 5 seconds
	ReceiveDataTimeout time.Duration

	// FastPacketAssembler assembles fast-packet PGN frames to complete messages.
	// Optional: if not set, messages are directly created out of frames with no assembly
	FastPacketAssembler nmea.Assembler
}

type Device struct {
	conn    *Connection
	config  DeviceConfig
	timeNow func() time.Time
}

func NewDevice(config DeviceConfig) *Device {
	if config.InterfaceName == "" {
		config.InterfaceName = "can0"
	}
	if config.ReceiveDataTimeout <= 0 {
		config.ReceiveDataTimeout = 5 * time.Second
	}

	return &Device{
		conn:    nil,
		config:  config,
		timeNow: time.Now,
	}
}

func (d *Device) Close() error {
	return d.conn.Close()
}

func (d *Device) Initialize() error {
	conn, err := NewConnection(d.config.InterfaceName)
	if err != nil {
		return err
	}
	d.conn = conn

	return nil
}

func (d *Device) WriteRawMessage(ctx context.Context, msg nmea.RawMessage) error {
	return errors.New("not implemented") // FIXME: implement
}

func (d *Device) ReadRawMessage(ctx context.Context) (nmea.RawMessage, error) {
	msg := nmea.RawMessage{}
	start := d.timeNow()
	for {
		select {
		case <-ctx.Done():
			return nmea.RawMessage{}, ctx.Err()
		default:
		}

		if err := d.conn.SetReadTimeout(50 * time.Millisecond); err != nil { // max 50ms block time for read per iteration
			return nmea.RawMessage{}, err
		}
		frame, err := d.conn.ReadRawFrame()

		now := d.timeNow()
		// on read errors we do not return immediately as for:
		// os.ErrDeadlineExceeded - we set new deadline on next iteration
		// io.EOF - we check if already read + received is enough to form complete message
		if err != nil {
			if errors.Is(err, errReadTimeout) {
				if now.Sub(start) > d.config.ReceiveDataTimeout {
					return nmea.RawMessage{}, err
				}
				continue
			}
			return nmea.RawMessage{}, err
		}

		if d.config.FastPacketAssembler != nil {
			if d.config.FastPacketAssembler.Assemble(frame, &msg) {
				return msg, err
			}
			continue
		}

		return nmea.RawMessage{
			Time:   frame.Time,
			Header: frame.Header,
			Data:   frame.Data[:],
		}, nil
	}
}
