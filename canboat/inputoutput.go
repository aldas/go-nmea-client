package canboat

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/aldas/go-nmea-client"
	"strconv"
	"strings"
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

func UnmarshalString(raw string) (nmea.RawMessage, error) {
	// 2021-07-29T10:18:31.758Z,6,126208,36,0,7,02,82,ff,00,10,02,00
	// 2023-02-07T11:55:11.002803898+02:00,2,127245,13,255,8,ff,07,ff,7f,00,00,ff,ff
	// time                               ,prio,pgn,src,dst,len,data...
	parts := strings.Split(raw, ",")
	if len(parts) < 7 {
		return nmea.RawMessage{}, errors.New("canboat input has fewer components than expected")
	}
	dLen, err := strconv.ParseUint(parts[5], 10, 16)
	if err != nil {
		return nmea.RawMessage{}, fmt.Errorf("canboat input invalid data length, err: %w", err)
	}
	if len(parts)-6 != int(dLen) {
		return nmea.RawMessage{}, errors.New("canboat input data length does not match bytes count")
	}

	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nmea.RawMessage{}, fmt.Errorf("canboat input invalid time format, err: %w", err)
	}
	prio, err := strconv.ParseUint(parts[1], 10, 8)
	if err != nil {
		return nmea.RawMessage{}, fmt.Errorf("canboat input invalid priority, err: %w", err)
	}
	pgn, err := strconv.ParseUint(parts[2], 10, 32)
	if err != nil {
		return nmea.RawMessage{}, fmt.Errorf("canboat input invalid PGN, err: %w", err)
	}
	source, err := strconv.ParseUint(parts[3], 10, 8)
	if err != nil {
		return nmea.RawMessage{}, fmt.Errorf("canboat input invalid source, err: %w", err)
	}
	destination, err := strconv.ParseUint(parts[4], 10, 8)
	if err != nil {
		return nmea.RawMessage{}, fmt.Errorf("canboat input invalid destination, err: %w", err)
	}

	data, err := hex.DecodeString(strings.Join(parts[6:], ""))
	if err != nil {
		return nmea.RawMessage{}, fmt.Errorf("canboat input failure to convert hex into bytes, err: %w", err)
	}

	return nmea.RawMessage{
		Time: t.UTC(),
		Header: nmea.CanBusHeader{
			PGN:         uint32(pgn),
			Priority:    uint8(prio),
			Source:      uint8(source),
			Destination: uint8(destination),
		},
		Data: data,
	}, nil
}
