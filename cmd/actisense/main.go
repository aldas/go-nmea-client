package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/aldas/go-nmea-client/actisense"
	"github.com/aldas/go-nmea-client/canboat"
	"github.com/tarm/serial"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	printRaw := flag.Bool("raw", false, "prints raw message")
	onlyRaw := flag.Bool("raw-only", false, "prints only raw message (does not parse to pgn)")
	noShowPNG := flag.Bool("np", false, "do not print parsed PNGs")
	isFile := flag.Bool("is-file", false, "consider device as ordinary file")
	inputFormat := flag.String("input-format", "ngt", "in which format packet are read (ngt,n2k-ascii)")
	deviceAddr := flag.String("device", "/dev/ttyUSB0", "path to Actisense NGT-1 USB device")
	pgnsPath := flag.String("pgns", "", "path to Canboat pgns.json file")
	outputFormat := flag.String("output-format", "json", "in which format raw and decoded packet should be printed out (json, canboat)")
	baudRate := flag.Int("baud", 115200, "device baud rate.")
	flag.Parse()

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

	switch *inputFormat {
	case "ngt", "n2k-ascii":
	default:
		log.Fatal("unknown input format type given\n")
	}

	switch *outputFormat {
	case "json", "canboat":
	default:
		log.Fatal("unknown output format type given\n")
	}

	var reader io.ReadWriteCloser
	var err error
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
	var device actisense.RawMessageReader
	switch *inputFormat {
	case "ngt":
		device = actisense.NewNGT1DeviceWithConfig(reader, config)
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

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	msgCount := uint64(0)
	errorCount := uint64(0)
	for {
		rawMessage, err := device.ReadRawMessage(ctx)
		msgCount++
		if err == io.EOF {
			break
		}
		if err != nil {
			if err == context.Canceled {
				return
			}
			log.Fatal(err)
		}
		if *onlyRaw {
			var b []byte
			switch *outputFormat {
			case "json":
				b, _ = json.Marshal(rawMessage)
			case "canboat":
				b, _ = canboat.MarshalRawMessage(rawMessage)
			}
			fmt.Printf("%s\n", b)
			continue
		}

		pgn, err := decoder.Decode(rawMessage)
		if err != nil {
			errorCount++
			var b []byte
			switch *outputFormat {
			case "json":
				b, _ = json.Marshal(rawMessage)
			case "canboat":
				b, _ = canboat.MarshalRawMessage(rawMessage)
			}
			fmt.Printf("# unknown PGN: %v (msgCount: %v, errCount: %v)\n", rawMessage.Header.PGN, msgCount, errorCount)
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
	fmt.Printf("# Finishing, number of processed messages: %v, errors: %v\n", msgCount, errorCount)
}
