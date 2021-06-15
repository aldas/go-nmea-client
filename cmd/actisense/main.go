package main

import (
	"context"
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
	"syscall"
	"time"
)

func main() {
	printRaw := flag.Bool("raw", false, "prints raw message")
	onlyRaw := flag.Bool("raw-only", false, "prints only raw message (does not parse pgn)")
	noShowPNG := flag.Bool("np", false, "do not print parsed PNGs")
	deviceAddr := flag.String("device", "/dev/ttyUSB0", "path to Actisense NGT-1 USB device")
	pgnsPath := flag.String("pgns", "", "path to Canboat pgns.json file")
	baudRate := flag.Int("baud", 115200, "device baud rate.")
	flag.Parse()

	if deviceAddr == nil || *deviceAddr == "" {
		log.Fatal("# missing device path\n")
	}
	if pgnsPath == nil || *pgnsPath == "" {
		log.Fatal("# missing pgns.json path\n")
	}

	schema, err := canboat.LoadCANBoatSchema(os.DirFS("."), *pgnsPath)
	if err != nil {
		log.Fatal(err)
	}

	stream, err := serial.OpenPort(&serial.Config{
		Name:        *deviceAddr,
		Baud:        *baudRate,
		ReadTimeout: 1 * time.Second,
		Size:        8,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	fmt.Printf("# Parsed %v known PGN definitions\n", len(schema.PGNs))

	device := actisense.NewNGT1Device(stream)
	device.DebugLogRawMessageBytes = *printRaw || *onlyRaw

	fmt.Printf("# Initializing device: %v\n", *deviceAddr)
	if err := device.Initialize(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("# Starting to read device: %v\n", *deviceAddr)
	time.Sleep(1 * time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var gracefulStop = make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)
	go func() {
		sig := <-gracefulStop
		fmt.Printf("# exiting. caught signal: %+v\n", sig)
		cancel()
	}()

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
		pgn, ok := schema.PGNs.FindByPGN(rawMessage.PGN)
		if !ok {
			fmt.Printf("# uknown PGN: %v: %#v\n", rawMessage.PGN, rawMessage)
			continue
		}
		if !pgn.Complete {
			fmt.Printf("# incomplete PGN: %v: %#v\n", rawMessage.PGN, rawMessage)
			continue
		}
		if *onlyRaw {
			continue
		}
		result, err := parsePGN(pgn, rawMessage)
		if err != nil {
			fmt.Printf("# error parsing parse PGN: %v: %#v, err: %v\n", rawMessage.PGN, rawMessage, err)
			continue
		}

		if *noShowPNG {
			continue
		}
		jsonBytes, _ := json.Marshal(result)
		fmt.Printf("%v\n", string(jsonBytes))
	}
}

func parsePGN(pgnConf canboat.PGN, raw nmea.RawMessage) (pgn nmea.CustomPGN, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("PGN parse paniced: %v", r)
		}
	}()
	return nmea.ParsePGN(pgnConf, raw)
}
