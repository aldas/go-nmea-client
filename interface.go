package nmea

import (
	"context"
)

type RawMessageReader interface {
	ReadRawMessage(ctx context.Context) (RawMessage, error)
	Initialize() error
	Close() error
}

type RawMessageWriter interface {
	Write(RawMessage) error
	Close() error
}

type RawMessageReaderWriter interface {
	RawMessageReader
	RawMessageWriter
}
