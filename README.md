# go-nmea-client

> WORK IN PROGRESS
> Only public because including private Go library is too much of a hustle for CI

Go library to read NMEA 2000 messages from SocketCAN interfaces or usb devices (Actisense NGT1/W2K-1 etc).

In addition, this repository contains command line application [actisense-reader](./actisense/main.go) to provide
following features:

* Can read different input formats:
  * SocketCAN format (TODO)
  * CanBoat raw format
  * Actisense format:
      * NGT1 Binary,
      * N2K Ascii,
      * N2K Binary,
      * Raw ASCII
* Can output read raw frames/messages as:
    * JSON,
    * HEX,
    * BASE64,
    * CanBoat format
* Can decode CAN messages to fields with CanBoat PGN database
* Can output decoded messages fields as: 
  * JSON (stdout)
  * CSV file (each PGN has own csv file). columns (fields) can be customized
* Can send STDIN input to CAN interface/device
* Can do basic NMEA2000 bus NODE mapping (which devices/nodes exist in bus)
    * Can list known nodes (send `!nodes` as input)
    * Can request nodes NAMES from STDIN (send `!addr-claim` as input)

## Disclaimer

This repository exists only because of [CanBoat](https://github.com/canboat/canboat) authors. They have done a lot of
work to acquire knowledge of NMEA2000 protocol and made it free.

## NMEA2000 reader

Compile NMEA2000 reader for different achitectures/platforms (AMD64,ARM32v6,ARM32v7,ARM64,MIPS32 (softfloat)).

```bash
make n2kreader-all
```

Run reader suitable for Raspberry Pi Zero with Canboat PGN database (canboat.json). Only decode PGNs 126996,126998 and output decoded
messages as JSON.

```bash
./n2kreader-reader-arm32v6 -pgns ./canboat.json -filter 126996,126998 -output-format json
```

* You can write data to NMEA bus by sending text to STDIN. Example `6,59904,0,255,3,14,f0,01` + `\n` sends PGN 59904 from src 0 to dst 255 requesting PGN 126996 (0x01, 0xf0, 0x14)
* `!nodes` - lists all knowns node NAME and their associated Source values
* `!addr-claim` - sends broadcast request for ISO Address Claim 

## Example

```go
func main() {
	f, err := os.Open("canboat.json")
	schema := canboat.CanboatSchema{}
	if err := json.NewDecoder(f).Decode(&schema); err != nil {
		log.Fatal(err)
	}
	decoder := canboat.NewDecoder(schema)

	// reader, err = os.OpenFile("/path/to/some/logged_traffic.bin", os.O_RDONLY, 0)
	reader, err := serial.OpenPort(&serial.Config{
		Name: "/dev/ttyUSB0",
		Baud: 115200,
		// ReadTimeout is duration that Read call is allowed to block. Device has different timeout for situation when
		// there is no activity on bus. Can not be smaller than 100ms
		ReadTimeout: 5 * time.Millisecond,
		Size:        8,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	config := actisense.Config{
		ReceiveDataTimeout:      5 * time.Second,
		DebugLogRawMessageBytes: false,
	}
	// device = actisense.NewN2kASCIIDevice(reader, config) // W2K-1 has support for Actisense N2K Ascii format
	device := actisense.NewBinaryDeviceWithConfig(reader, config)
	if err := device.Initialize(); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	for {
		rawMessage, err := device.ReadRawMessage(ctx)
		if err != nil {
			if err == io.EOF || err == context.Canceled {
				return
			}
			log.Fatal(err)
		}
		b, _ := json.Marshal(rawMessage)
		fmt.Printf("#Raw %s\n", b)

		pgn, err := decoder.Decode(rawMessage)
		if err != nil {
			fmt.Println(err)
			continue
		}

		b, _ = json.Marshal(pgn)
		fmt.Printf("%s\n", b)
	}
}
```

## Useful commands

### Actisense reader utility

Build command line utility for your current arch

```bash 
make build-reader
```

Create Actisense reader that can be run on MIPS architecture (Teltonika RUT955 router ,CPU: Atheros Wasp, MIPS 74Kc, 550
MHz)

```bash
GOOS=linux GOARCH=mips GOMIPS=softfloat go build -ldflags="-s -w" -o n2k-reader-mips cmd/n2kreader/main.go
```

Help about arguments:

```bash
./n2k-reader -help
```

Example usage:

```bash 
./n2k-reader -pgns=canboat/testdata/canboat.json \
   -device="actisense/testdata/actisense_n2kascii_20221028_10s.txt" \
   -is-file=true \
   -output-format=json \
   -input-format=n2k-ascii
```

This is instructs reader to treat device `actisense/testdata/actisense_n2kascii_20221028_10s.txt` as an ordinary file
instead
of serial device. All input read from device is decoded as `Actisense N2K` binary protocol (
Actisense [W2K-1](https://actisense.com/products/w2k-1-nmea-2000-wifi-gateway/) device can output this)
and print output in JSON format.

# Research/check following libraries:

* CAN libraries

1. https://github.com/algleason/canlib (7) (socketcan) BSD2
2. https://github.com/angelodlfrtr/go-can (4) (serial, socketcan) MIT?  
   uses https://github.com/brutella/can
3. https://github.com/brutella/can (133) (socketcan) MIT
4. https://github.com/go-daq/canbus (16) BSD3  (socketcan)  no deps
5. https://github.com/einride/can-go (socketcan) MIT

* NMEA libraries

1. https://github.com/canboat/canboat (307) Apache2 (nmea2000) lang: C
2. https://github.com/timmathews/argo GPL3 (nmea2000)  lang: GO
3. https://github.com/adrianmo/go-nmea (147) MIT (nmea0183 only)
4. https://github.com/pilebones/go-nmea (2) GPL3 (nmea0183 only)
5. https://github.com/BertoldVdb/go-ais (18) MIT (nmea0183 only)

Useful links:

1. https://gist.github.com/jackm/f33d6e3a023bfcc680ec3bfa7076e696