package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	"sort"
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
	noAddressMapper := flag.Bool("dam", false, "disable address mapper")
	isFile := flag.Bool("is-file", false, "consider device as ordinary file")
	inputFormat := flag.String("input-format", "ngt", "in which format packet are read (ngt, n2k-bin, n2k-ascii, canboat-raw)")
	deviceAddr := flag.String("device", "/dev/ttyUSB0", "path to Actisense NGT-1 USB device")
	pgnsPath := flag.String("pgns", "", "path to Canboat pgns.json file")
	pgnFilter := flag.String("filter", "", "comma separated list of PGNs to filter")
	csvFieldsRaw := flag.String("csv-fields", "", "list of PGNs and their fields to be written in CSV. `129025:time_ms,latitude,longitude;65280:time_ms,manufacturerCode,industryCode`")
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
	case "ngt", "n2k-bin", "n2k-ascii", "canboat-raw":
	default:
		log.Fatal("unknown input format type given\n")
	}

	var csvFields csvPGNs
	isCSV := false
	if csvFieldsRaw != nil {
		csvFields, err = parseCSVFieldsRaw(*csvFieldsRaw)
		if err != nil {
			log.Fatalf("%v\n", err)
		}
		for _, cf := range csvFields {
			filter = append(filter, cf.PGN)
		}
		if len(csvFields) > 0 {
			isCSV = true
		}
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

	var device nmea.RawMessageReaderWriter
	switch *inputFormat {
	case "canboat-raw":
		device = canboat.NewCanBoatReader(reader)
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

	isAddressMapperEnabled := noAddressMapper == nil || !*noAddressMapper
	var addressMapper *nmea.AddressMapper
	if isAddressMapperEnabled {
		addressMapper = nmea.NewAddressMapper(device)
		fmt.Printf("# Starting address mapper process\n")
		go func(ctx context.Context, am *nmea.AddressMapper) {
			if err := am.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				fmt.Printf("# AddressMapper ended with error: %v\n", err)
			}
		}(ctx, addressMapper)
		if !*isFile {
			go func(ctx context.Context, am *nmea.AddressMapper) {
				// After 1 sec delay send ISO Address claim to all Nodes on bus to learn their NAME values
				select {
				case <-ctx.Done():
					return
				case <-time.After(1 * time.Second):
					fmt.Printf("# Broadcasting ISO Address claim\n")
					am.BroadcastIsoAddressClaimRequest()
				}

			}(ctx, addressMapper)
		}
	}

	if onlyRead != nil && !*onlyRead && !*isFile {
		fmt.Printf("# Starting STDIN process\n")
		go handleSTDIO(device, addressMapper)
	}

	msgCount := uint64(0)
	errorCountDecode := uint64(0)
	errorCountRead := uint64(0)
	nodesBySource := map[uint8]nmea.Node{}
	for {
		rawMessage, err := device.ReadRawMessage(ctx)
		msgCount++
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			errorCountRead++
			if errors.Is(err, context.Canceled) {
				return
			}
			fmt.Printf("# Error ReadRawMessage: %v\n", err)
			if errorCountRead > 20 {
				return
			}
			continue
		}
		errorCountRead = 0

		isNodeChanged := false
		if isAddressMapperEnabled {
			isNodeChanged, err = addressMapper.Process(rawMessage)
			if err != nil {
				fmt.Printf("# Error at addressMapper processing: %v\n", err)
			}
			if isNodeChanged {
				nodesBySource = addressMapper.NodesInUseBySource()
			}
		}

		if filter != nil && !contains(filter, rawMessage.Header.PGN) {
			continue
		}

		var nodeNAME uint64
		if node, ok := nodesBySource[rawMessage.Header.Source]; ok {
			nodeNAME = node.NAME
			if isNodeChanged {
				fmt.Printf("# New or changed Node: %+v\n", node)
			}
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
			fmt.Printf("# unknown PGN: %v NodeNAME: %v (msgCount: %v, errCount: %v)\n", rawMessage.Header.PGN, nodeNAME, msgCount, errorCountDecode)
			fmt.Printf("%s\n", b)
			continue
		}

		pgn.NodeNAME = nodeNAME
		if isCSV {
			if fields, cpgn, ok := csvFields.Match(pgn, rawMessage.Time); ok {
				if err := writeCSV(cpgn, fields); err != nil {
					log.Fatal(err)
				}
			}
		}

		if *noShowPNG {
			continue
		}
		var b []byte
		switch *outputFormat {
		case "json":
			b, err = json.Marshal(pgn)
		case "canboat":
			b, err = canboat.MarshalRawMessage(rawMessage) // FIXME: as raw and not as canboat json
		}
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s\n", b)
	}
	fmt.Printf("# Finishing, number of processed messages: %v, errors: %v\n", msgCount, errorCountDecode)
}

func handleSTDIO(device nmea.RawMessageWriter, addressMapper *nmea.AddressMapper) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "!nodes" {
			nodes := addressMapper.Nodes()
			sort.Sort(nodesBySrc(nodes))

			fmt.Printf("# Known nodes: %v\n", len(nodes))
			for _, n := range nodes {
				fmt.Printf("# node: NAME: %v, source: %v\n", n.NAME, n.Source)
			}
			continue
		} else if strings.HasPrefix(line, "!addr-claim") {
			addressMapper.BroadcastIsoAddressClaimRequest()
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

type nodesBySrc nmea.Nodes

func (v nodesBySrc) Len() int           { return len(v) }
func (v nodesBySrc) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (v nodesBySrc) Less(i, j int) bool { return v[i].Source < v[j].Source }
