package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/aldas/go-nmea-client"
	"github.com/aldas/go-nmea-client/actisense"
	"github.com/aldas/go-nmea-client/canboat"
	"github.com/tarm/serial"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	printRaw := flag.Bool("raw", false, "prints raw message")
	onlyRead := flag.Bool("read-only", false, "only reads device/file and does not write into it")
	onlyRaw := flag.Bool("raw-only", false, "prints only raw message (does not parse to pgn)")
	noShowPNG := flag.Bool("np", false, "do not print parsed PNGs")
	isFile := flag.Bool("is-file", false, "consider device as ordinary file")
	inputFormat := flag.String("input-format", "ngt", "in which format packet are read (ngt, n2k-bin, n2k-ascii)")
	deviceAddr := flag.String("device", "/dev/ttyUSB0", "path to Actisense NGT-1 USB device")
	pgnsPath := flag.String("pgns", "", "path to Canboat pgns.json file")
	pgnFilter := flag.String("filter", "", "comma separated list of PGNs to filter")
	outputFormat := flag.String("output-format", "json", "in which format raw and decoded packet should be printed out (json, canboat, hex, base64)")
	baudRate := flag.Int("baud", 115200, "device baud rate.")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if deviceAddr == nil || *deviceAddr == "" {
		log.Fatal("# missing device path\n")
	}

	var decoder *canboat.Decoder
	if !*onlyRaw {
		if pgnsPath == nil || *pgnsPath == "" {
			log.Fatal("# missing pgns.json path\n")
		}

		schema, err := canboat.LoadCANBoatSchema(os.DirFS("."), *pgnsPath)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("# Parsed %v known PGN definitions\n", len(schema.PGNs))

		decoder = canboat.NewDecoder(schema)
	}

	var err error
	var filter []uint32
	if pgnFilter != nil && *pgnFilter != "" {
		filter, err = string2intSlice(*pgnFilter)
		if err != nil {
			log.Fatalf("invalid pgn filter given, %v\n", err)
		}
		fmt.Printf("# Using PGN filter: %v\n", filter)
	}

	switch *inputFormat {
	case "ngt", "n2k-bin", "n2k-ascii":
	default:
		log.Fatal("unknown input format type given\n")
	}

	switch *outputFormat {
	case "json", "canboat", "hex", "base64":
	default:
		log.Fatal("unknown output format type given\n")
	}

	var reader io.ReadWriteCloser
	if *isFile {
		reader, err = os.OpenFile(*deviceAddr, os.O_RDONLY, 0)
	} else {
		reader, err = serial.OpenPort(&serial.Config{
			Name: *deviceAddr,
			Baud: *baudRate,
			// ReadTimeout is duration that Read call is allowed to block. Device has different timeout for situation when
			// there is no activity on bus. Can not be smaller than 100ms
			ReadTimeout: 100 * time.Millisecond,
			Size:        8,
		})
	}
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	config := actisense.Config{
		ReceiveDataTimeout:      5 * time.Second,
		DebugLogRawMessageBytes: *printRaw, // || *onlyRaw // FIXME
	}
	if *isFile {
		config.ReceiveDataTimeout = 100 * time.Millisecond
	}

	var device actisense.RawMessageReaderWriter
	switch *inputFormat {
	case "ngt", "n2k-bin":
		device = actisense.NewBinaryDeviceWithConfig(reader, config)
	case "n2k-ascii":
		device = actisense.NewN2kASCIIDevice(reader, config)
	}

	if !*isFile {
		fmt.Printf("# Initializing device: %v\n", *deviceAddr)
		if err := device.Initialize(); err != nil {
			log.Fatal(err)
		}
	}
	fmt.Printf("# Starting to read device: %v\n", *deviceAddr)
	time.Sleep(1 * time.Second)

	if onlyRead != nil && !*onlyRead {
		go scanLines(device)
	}

	msgCount := uint64(0)
	errorCountDecode := uint64(0)
	errorCountRead := uint64(0)
	for {
		rawMessage, err := device.ReadRawMessage(ctx)
		msgCount++
		if err == io.EOF {
			break
		}
		if err != nil {
			errorCountRead++
			if err == context.Canceled {
				return
			}
			fmt.Printf("# Error ReadRawMessage: %v\n", err)
			if errorCountRead > 20 {
				return
			}
			continue
		}
		errorCountRead = 0

		if filter != nil && !contains(filter, rawMessage.Header.PGN) {
			continue
		}

		if *onlyRaw {
			var b []byte
			switch *outputFormat {
			case "json":
				b, _ = json.Marshal(rawMessage)
			case "canboat":
				b, _ = canboat.MarshalRawMessage(rawMessage)
			case "hex":
				b = []byte(hex.EncodeToString(nmea.MarshalRawMessage(rawMessage)))
			case "base64":
				b = []byte(base64.StdEncoding.EncodeToString(nmea.MarshalRawMessage(rawMessage)))
			}
			fmt.Printf("%s\n", b)
			continue
		}

		pgn, err := decoder.Decode(rawMessage)
		if err != nil {
			errorCountDecode++
			var b []byte
			switch *outputFormat {
			case "json":
				b, _ = json.Marshal(rawMessage)
			case "canboat":
				b, _ = canboat.MarshalRawMessage(rawMessage)
			}
			fmt.Printf("# unknown PGN: %v (msgCount: %v, errCount: %v)\n", rawMessage.Header.PGN, msgCount, errorCountDecode)
			fmt.Printf("%s\n", b)
			continue
		}

		if *noShowPNG {
			continue
		}

		var b []byte
		switch *outputFormat {
		case "json":
			b, err = json.Marshal(pgn)
		case "canboat":
			b, _ = canboat.MarshalRawMessage(rawMessage) // FIXME: as raw and not as canboat json
		}
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s\n", b)
	}
	fmt.Printf("# Finishing, number of processed messages: %v, errors: %v\n", msgCount, errorCountDecode)
}

func scanLines(device actisense.RawMessageWriter) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		msg, err := parseLine(line)
		if err != nil {
			fmt.Printf("%v", err)
			continue
		}

		if err = device.Write(msg); err != nil {
			fmt.Printf("# Error at writing: %v", err)
		}
	}
}

func parseLine(line string) (nmea.RawMessage, error) {
	// Canboat format is
	// prio, pgn, src, dst, len, data...
	// 6,59904,0,128,3,16,f0,01
	parts := strings.Split(line, ",")
	if len(parts) < 6 {
		return nmea.RawMessage{}, fmt.Errorf("# Error invalid input format")
	}
	msg := nmea.RawMessage{}
	n, err := parseUint8(parts[0], 0, 7, "priority")
	if err != nil {
		return nmea.RawMessage{}, fmt.Errorf("# Error parsing priority, err: %v", err)
	}
	msg.Header.Priority = n

	pgn, err := strconv.Atoi(parts[1])
	if err != nil {
		return nmea.RawMessage{}, fmt.Errorf("# Error parsing PGN, err: %v", err)
	}
	msg.Header.PGN = uint32(pgn)

	n, err = parseUint8(parts[2], 0, 256, "src")
	if err != nil {
		return nmea.RawMessage{}, fmt.Errorf("# Error parsing src, err: %v", err)
	}
	msg.Header.Source = n

	n, err = parseUint8(parts[3], 0, 256, "dst")
	if err != nil {
		return nmea.RawMessage{}, fmt.Errorf("# Error parsing dst, err: %v", err)
	}
	msg.Header.Destination = n

	dataLen, err := strconv.Atoi(parts[4])
	if err != nil {
		return nmea.RawMessage{}, fmt.Errorf("# Error parsing data length, err: %v", err)
	}
	data, err := hex.DecodeString(strings.Join(parts[5:], ""))
	if err != nil {
		return nmea.RawMessage{}, fmt.Errorf("# Error decoding hex data, err: %v", err)
	}
	msg.Data = data[0:dataLen]

	return msg, nil
}

func parseUint8(raw string, min int, max int, name string) (uint8, error) {
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("# Error failed to parse %v, err: %w", name, err)
	}
	if n < min || n > max {
		return 0, fmt.Errorf("# Error invalid %v", name)
	}
	return uint8(n), nil
}

func string2intSlice(s string) ([]uint32, error) {
	result := make([]uint32, 0, 10)
	for _, p := range strings.Split(s, ",") {
		pgn, err := strconv.Atoi(p)
		if err != nil {
			return nil, err
		}
		result = append(result, uint32(pgn))
	}
	return result, nil
}

func contains[T comparable](elems []T, v T) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}
