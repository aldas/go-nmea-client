package canboat

import (
	"bytes"
	"fmt"
	"github.com/aldas/go-nmea-client"
	"strconv"
	"time"
)

func MarshalRawMessage(v nmea.RawMessage) ([]byte, error) {
	buf := new(bytes.Buffer)
	buf.WriteString(v.Time.Format(time.RFC3339Nano))
	buf.WriteByte(',')
	buf.WriteString(strconv.Itoa(int(v.Header.Priority)))
	buf.WriteByte(',')
	buf.WriteString(strconv.Itoa(int(v.Header.PGN)))
	buf.WriteByte(',')
	buf.WriteString(strconv.Itoa(int(v.Header.Source)))
	buf.WriteByte(',')
	buf.WriteString(strconv.Itoa(int(v.Header.Destination)))
	buf.WriteByte(',')
	buf.WriteString(strconv.Itoa(len(v.Data)))
	for _, b := range v.Data {
		if _, err := fmt.Fprintf(buf, ",%02x", b); err != nil {
			return nil, fmt.Errorf("MarshalRawMessage failure, err: %w", err)
		}
	}
	return buf.Bytes(), nil
}
