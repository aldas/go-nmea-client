package socketcan

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

// sudo ip link set can0 down && sudo /sbin/ip link set can0 up type can bitrate 250000

func xTestName(t *testing.T) {
	con, err := NewConnection("can0")
	if err != nil {
		assert.NoError(t, err)
		return
	}
	defer con.Close()

	f, err := con.ReadRawFrame()
	if err != nil {
		assert.NoError(t, err)
		return
	}
	fmt.Printf("frame: %+v\n", f)
}

func xTestName2(t *testing.T) {
	dev := NewDevice(DeviceConfig{InterfaceName: "can0"})

	if err := dev.Initialize(); err != nil {
		assert.NoError(t, err)
		return
	}
	defer dev.Close()

	for i := 0; i < 100; i++ {
		f, err := dev.ReadRawMessage(context.Background())
		if err != nil {
			assert.NoError(t, err)
			return
		}
		fmt.Printf("frame: %+v\n", f)
	}
}
