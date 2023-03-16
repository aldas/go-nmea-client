package nmea

import (
	"context"
)

type RawMessageReader interface {
	ReadRawMessage(ctx context.Context) (msg RawMessage, err error)
	Initialize() error
	Close() error
}

type RawMessageWriter interface {
	WriteRawMessage(ctx context.Context, msg RawMessage) error
	Close() error
}

type RawMessageReaderWriter interface {
	RawMessageReader
	RawMessageWriter
}
