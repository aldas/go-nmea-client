package socketcan

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/aldas/go-nmea-client"
	"golang.org/x/sys/unix"
	"net"
	"syscall"
	"time"
)

const (
	canRaw = 1

	// canIDMask is bitmask to get 0-28bits belonging to CAN ID from socketCAN struct
	canIDMask = uint32(0b111) << 29
	// canIDERRFlag is bit 29 in CAN ID and means ERR error message flag (0 = data frame, 1 = error message)
	canIDERRFlag = uint32(1 << 29)
	// canIDRTRFlag is bit 30 in CAN ID and means RTR remote transmission request (1 = rtr frame)
	canIDRTRFlag = uint32(1 << 30)
	// canIDEFFFlag is bit 31 in CAN ID and means EFF extended frame format / IDE identifier extension flag (0 = standard 11 bit, 1 = extended 29 bit)
	canIDEFFFlag = uint32(1 << 31)
)

type Connection struct {
	socketFD int
	timeNow  func() time.Time
}

func NewConnection(ifName string) (*Connection, error) {
	ifi, err := net.InterfaceByName(ifName)
	if err != nil {
		return nil, fmt.Errorf("bad ifName: %w", err)
	}

	fd, err := unix.Socket(unix.AF_CAN, unix.SOCK_RAW, canRaw)
	if err != nil {
		return nil, fmt.Errorf("could not create CAN socket: %w", err)
	}

	addr := &unix.SockaddrCAN{Ifindex: ifi.Index}
	if err = unix.Bind(fd, addr); err != nil {
		return nil, fmt.Errorf("could not bind CAN socket: %w", err)
	}

	return &Connection{
		socketFD: fd,
		timeNow:  time.Now,
	}, nil
}

func isContinuableSocketErr(err error) bool {
	// EWOULDBLOCK - If you set a timeout on the socket with SO_RCVTIMEO or SO_SNDTIMEO - in this case, a receive or
	// send will return with EWOULDBLOCK if the timeout elapses while no input data becomes available or the output
	// buffer remains full

	// EINTR - If a signal occurs during a blocking operation, then the operation will either (a) return partial
	// completion, or (b) return failure, do nothing, and set errno to EINTR.

	return err == syscall.EWOULDBLOCK || err == syscall.EINTR
}

var errReadTimeout = errors.New("read timeout")
var errWriteTimeout = errors.New("write timeout")

func (i Connection) SetReadTimeout(timeout time.Duration) error {
	return i.setSocketTimeout(unix.SO_RCVTIMEO, timeout)
}

func (i Connection) SetSendTimeout(timeout time.Duration) error {
	return i.setSocketTimeout(unix.SO_SNDTIMEO, timeout)
}

func (i Connection) setSocketTimeout(opt int, timeout time.Duration) error {
	tv := unix.NsecToTimeval(timeout.Nanoseconds())
	err := unix.SetsockoptTimeval(i.socketFD, unix.SOL_SOCKET, opt, &tv)
	return err
}

func (i Connection) Close() error {
	return unix.Close(i.socketFD)
}

func (i Connection) SendFrame(raw nmea.RawFrame) error {
	// Can frame structure: https://github.com/linux-can/can-utils/blob/affdc1b79973c7497bb8607603c24734e11a91aa/include/linux/can.h#L107
	canFrame := make([]byte, 16)

	// bits 0-28 is CAN ID
	// bit 29 is ERR error message flag (0 = data frame, 1 = error message)
	// bit 30 is RTR remote transmission request (1 = rtr frame)
	// bit 31 is EFF extended frame format / IDE identifier extension flag (0 = standard 11 bit, 1 = extended 29 bit)
	canID := raw.Header.Uint32() | canIDEFFFlag         // canID + EFF flag
	binary.LittleEndian.PutUint32(canFrame[0:4], canID) // FIXME: for big-endian arch (mips64, ppc64) we should use big-endian

	// bits 32-40 data length
	canFrame[4] = raw.Length
	copy(canFrame[8:], raw.Data[:raw.Length])

	_, err := unix.Write(i.socketFD, canFrame)
	if isContinuableSocketErr(err) {
		return errWriteTimeout
	}
	return err
}

func (i Connection) ReadRawFrame() (nmea.RawFrame, error) {
	canFrame := make([]byte, 16)
	_, err := unix.Read(i.socketFD, canFrame)
	if err != nil {
		if isContinuableSocketErr(err) {
			return nmea.RawFrame{}, errReadTimeout
		}
		return nmea.RawFrame{}, err
	}
	canID := binary.LittleEndian.Uint32(canFrame[0:4])
	if canID&canIDRTRFlag != 0 {
		return nmea.RawFrame{}, errors.New("read CAN remote transmission request frame")
	} else if canID&canIDERRFlag != 0 {
		return nmea.RawFrame{}, errors.New("read CAN error message frame")
	}

	f := nmea.RawFrame{
		Time:   i.timeNow(),
		Header: nmea.ParseCANID(canID ^ canIDMask),
		Length: canFrame[4],
	}
	copy(f.Data[:], canFrame[8:8+f.Length])

	return f, nil
}
