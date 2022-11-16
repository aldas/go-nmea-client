# go-nmea-client
Go client to read NMEA 2000 messages from canbus interfaces or usb devices

> WORK IN PROGRESS
> Only public because including private Go library is too much of a hustle for CI

> Relies heavily on work done by https://github.com/canboat/canboat developers (basically is port of CanBoat)


## TODO

* Canboat Field type `VARIABLE` printing. This field has value as integer that references other PGN field by their order.
* Assembling of ISO TP and Fastpacket frames into complete message
* Better test coverage


Research/check following libraries:

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
make actisense
```

Create Actisense reader that can be run on MIPS architecture (Teltonika RUT955 router ,CPU: Atheros Wasp, MIPS 74Kc, 550 MHz)
```bash
GOOS=linux GOARCH=mips GOMIPS=softfloat go build -ldflags="-s -w" -o actisense-reader-mips cmd/actisense/main.go
```

Help about arguments:
```bash
./actisense-reader -help
```

Example usage:
```bash 
./actisense-reader -pgns=canboat/testdata/canboat.json \
   -device="actisense/testdata/actisense_n2kascii_20221028_10s.txt" \
   -is-file=true \
   -output-format=json \
   -input-format=n2k-ascii
```
This is instructs reader to treat device `actisense/testdata/actisense_n2kascii_20221028_10s.txt` as an ordinary file instead
of serial device. All input read from device is decoded as `Actisense N2K` binary protocol (Actisense [W2K-1](https://actisense.com/products/w2k-1-nmea-2000-wifi-gateway/) device can output this)
and print output in JSON format.


