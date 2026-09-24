package mp4_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"

	"github.com/Eyevinn/mp4ff/bits"
	"github.com/Eyevinn/mp4ff/hevc"
	"github.com/Eyevinn/mp4ff/mp4"
)

const (
	vpsHex = "40010c01ffff022000000300b0000003000003007b18b024"
	spsHex = "420101022000000300b0000003000003007ba0078200887db6718b92448053888892cf24a69272c9124922dc91aa48fca223ff000100016a02020201"
	ppsHex = "4401c0252f053240"
)

func TestHvcC(t *testing.T) {
	vpsNalu, err := hex.DecodeString(vpsHex)
	if err != nil {
		t.Error(err)
	}
	spsNalu, err := hex.DecodeString(spsHex)
	if err != nil {
		t.Error(err)
	}
	ppsNalu, err := hex.DecodeString(ppsHex)
	if err != nil {
		t.Error(err)
	}
	includePS := true
	hvcC, err := mp4.CreateHvcC([][]byte{vpsNalu}, [][]byte{spsNalu}, [][]byte{ppsNalu}, true, true, true, includePS)
	if err != nil {
		t.Error(err)
	}
	boxDiffAfterEncodeAndDecode(t, hvcC)
}

func TestHvcCLengthSize(t *testing.T) {
	vpsNalu, _ := hex.DecodeString(vpsHex)
	spsNalu, _ := hex.DecodeString(spsHex)
	ppsNalu, _ := hex.DecodeString(ppsHex)
	createHvcC := func(t *testing.T, lengthSizeMinusOne byte) *mp4.HvcCBox {
		t.Helper()
		hvcC, err := mp4.CreateHvcC([][]byte{vpsNalu}, [][]byte{spsNalu}, [][]byte{ppsNalu}, true, true, true, true)
		if err != nil {
			t.Fatal(err)
		}
		hvcC.LengthSizeMinusOne = lengthSizeMinusOne
		return hvcC
	}
	for _, lengthSizeMinusOne := range []byte{0, 1, 3} {
		t.Run(fmt.Sprintf("%d-byte", lengthSizeMinusOne+1), func(t *testing.T) {
			boxDiffAfterEncodeAndDecode(t, createHvcC(t, lengthSizeMinusOne))
		})
	}

	t.Run("3-byte lengths rejected", func(t *testing.T) {
		buf := bytes.Buffer{}
		if err := createHvcC(t, 3).Encode(&buf); err != nil {
			t.Fatal(err)
		}
		data := buf.Bytes()
		data[8+21] = data[8+21]&0xfc | 2 // lengthSizeMinusOne, after the 8-byte box header
		if _, err := mp4.DecodeBox(0, bytes.NewReader(data)); !errors.Is(err, hevc.ErrLengthSize) {
			t.Errorf("DecodeBox: got error %v, want %v", err, hevc.ErrLengthSize)
		}
		if _, err := mp4.DecodeBoxSR(0, bits.NewFixedSliceReader(data)); !errors.Is(err, hevc.ErrLengthSize) {
			t.Errorf("DecodeBoxSR: got error %v, want %v", err, hevc.ErrLengthSize)
		}
	})
}
