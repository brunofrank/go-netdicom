package fuzzpdu

import (
	"bytes"
	"encoding/binary"
	"flag"
	"github.com/yasushi-saito/go-dicom/dicomio"
	"github.com/yasushi-saito/go-netdicom/dimse"
	"github.com/yasushi-saito/go-netdicom/pdu"
)

func init() {
	flag.Parse()
}

func Fuzz(data []byte) int {
	in := bytes.NewBuffer(data)
	if len(data) == 0 || data[0] <= 0xc0 {
		pdu.ReadPDU(in, 4<<20)
	} else {
		d := dicomio.NewDecoder(in, int64(len(data)), binary.LittleEndian, dicomio.ExplicitVR)
		dimse.ReadMessage(d)
	}
	return 0
}
