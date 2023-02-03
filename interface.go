package nmea

import (
	"context"
)

type RawMessageReader interface {
	ReadRawMessage(ctx context.Context) (RawMessage, error)
	Initialize() error
}

type RawMessageWriter interface {
	Write(RawMessage) error
}

type RawMessageReaderWriter interface {
	RawMessageReader
	RawMessageWriter
}
