package main

import (
	"context"
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
	deviceAddr := flag.String("device", "/dev/ttyUSB0", "path to Actisense NGT-1 USB device")
	pgnsPath := flag.String("pgns", "", "path to Canboat pgns.json file")
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

	stream, err := serial.OpenPort(&serial.Config{
		Name: *deviceAddr,
		Baud: *baudRate,
		// ReadTimeout is duration that Read call is allowed to block. Device has different timeout for situation when
		// there is no activity on bus. Can not be smaller than 100ms
		ReadTimeout: 100 * time.Millisecond,
		Size:        8,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	config := actisense.Config{
		ReceiveDataTimeout: 5 * time.Second,
	}
	device := actisense.NewNGT1DeviceWithConfig(stream, config)
	device.DebugLogRawMessageBytes = *printRaw // || *onlyRaw // FIXME

	fmt.Printf("# Initializing device: %v\n", *deviceAddr)
	if err := device.Initialize(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("# Starting to read device: %v\n", *deviceAddr)
	time.Sleep(1 * time.Second)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	for {
		rawMessage, err := device.ReadRawMessage(ctx)
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
			fmt.Printf("{time: \"%v\", pgn: %v, src: %v, dst: %v, p: %v data: %#v}\n",
				rawMessage.Time.Format(time.RFC3339Nano),
				rawMessage.Header.PGN,
				rawMessage.Header.Source,
				rawMessage.Header.Destination,
				rawMessage.Header.Priority,
				rawMessage.Data.AsHex(),
			)
			continue
		}

		_, err = decoder.Decode(rawMessage)
		if err != nil {
			fmt.Printf("# uknown PGN: %v: %#v\n", rawMessage.Header.PGN, rawMessage)
			continue
		}

		if *noShowPNG {
			continue
		}
		// TODO print decoded message
	}
}
