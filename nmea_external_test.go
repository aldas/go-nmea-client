package nmea_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aldas/go-nmea-client"
	"github.com/aldas/go-nmea-client/actisense"
	"github.com/aldas/go-nmea-client/canboat"
	test_test "github.com/aldas/go-nmea-client/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExternalNGT1Device_Read(t *testing.T) {
	examplePGNs := test_test.LoadBytes(t, "../canboat/testdata/pgns.json")
	schema := canboat.CanboatSchema{}

	err := json.Unmarshal(examplePGNs, &schema)
	assert.NoError(t, err)

	//for _, pgn := range schema.PGNs {
	//	for _, f := range pgn.Fields {
	//
	//		switch f.Type {
	//		case nmea.FieldTypeUnknownReal,
	//			nmea.FieldTypeInteger,
	//			nmea.FieldTypeEnumValue,
	//			nmea.FieldTypeBitValues,
	//			nmea.FieldTypeManufacturerCode:
	//			continue
	//		case nmea.FieldTypeBinaryData:
	//			fmt.Printf("%v;%v;%v;%v;%v;%v;\n", pgn.PGN, f.ID, "FieldTypeBinaryData", f.BitLength, f.Match, f.Units)
	//		default:
	//			fmt.Printf("%v;%v;%v;%v;%v;%v;\n", pgn.PGN, f.ID, f.Type, f.BitLength, f.Match, f.Units)
	//		}
	//	}
	//}

	//pgn, _ := schema.PGNs.FindByPGN(126992)
	//x := fmt.Sprintf("%#v", pgn)
	//fmt.Printf(x)

	exampleData := test_test.LoadBytes(t, "../actisense/testdata/actisense-serial-ng1-cat-usb-2021-05-14-1005.bin")
	r := bytes.NewReader(exampleData)
	wr := bufio.NewReadWriter(bufio.NewReader(r), nil)

	device := actisense.NewNGT1Device(wr)
	for {
		packet, err := device.ReadRawMessage(context.Background())
		if err != nil {
			fmt.Println(err)
			break
		}

		pgn, ok := schema.PGNs.FindByPGN(packet.PGN)
		if !ok {
			fmt.Printf("could not find PGN: %v\n", packet.PGN)
			continue
		}
		result, err := nmea.ParsePGN(pgn, packet)
		fmt.Printf("%v\n", result)

		assert.NoError(t, err)
		if err != nil {
			break
		}
	}
}
