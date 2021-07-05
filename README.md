# go-nmea-client
Go client to read NMEA 2000 messages from canbus interfaces or usb devices

> WORK IN PROGRESS
> Only public because including private Go library is too much of a hustle for CI

> Relies heavily on work done by https://github.com/canboat/canboat developers


## TODO

Research/check following libraries:

* CAN libraries
1. https://github.com/algleason/canlib (7) (socketcan) BSD2
2. https://github.com/angelodlfrtr/go-can (4) (serial, socketcan) MIT?  
   uses https://github.com/brutella/can
3. https://github.com/brutella/can (133) (socketcan) MIT
4. https://github.com/go-daq/canbus (16) BSD3  (socketcan)  no deps
5. https://github.com/einride/can-go (socketcan) MIT

NMEA libraries:
1. https://github.com/canboat/canboat (307) Apache2 (nmea2000)
2. https://github.com/adrianmo/go-nmea (147) MIT (nmea0183 only)
3. https://github.com/pilebones/go-nmea (2) GPL3 (nmea0183 only)
4. https://github.com/BertoldVdb/go-ais (18) MIT (nmea0183 only)
