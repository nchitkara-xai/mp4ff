package mp4_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"

	"github.com/Eyevinn/mp4ff/avc"
	"github.com/Eyevinn/mp4ff/bits"
	"github.com/Eyevinn/mp4ff/mp4"
)

func TestAvcCNaluLengthSize(t *testing.T) {
	sps, _ := hex.DecodeString("6764001eacd940a02ff9610000030001000003003c8f162d96")
	pps, _ := hex.DecodeString("68ebecb22c")
	createAvcC := func(t *testing.T, naluLengthSize byte) *mp4.AvcCBox {
		t.Helper()
		avcC, err := mp4.CreateAvcC([][]byte{sps}, [][]byte{pps}, true /* includePS */)
		if err != nil {
			t.Fatal(err)
		}
		avcC.NaluLengthSize = naluLengthSize
		return avcC
	}
	for _, naluLengthSize := range []byte{0, 1, 2} {
		t.Run(fmt.Sprintf("NaluLengthSize %d", naluLengthSize), func(t *testing.T) {
			boxDiffAfterEncodeAndDecode(t, createAvcC(t, naluLengthSize))
		})
	}

	t.Run("3-byte lengths rejected", func(t *testing.T) {
		buf := bytes.Buffer{}
		if err := createAvcC(t, 0).Encode(&buf); err != nil {
			t.Fatal(err)
		}
		data := buf.Bytes()
		data[8+4] = 0xfc | 2 // lengthSizeMinusOne, after the 8-byte box header
		if _, err := mp4.DecodeBox(0, bytes.NewReader(data)); !errors.Is(err, avc.ErrLengthSize) {
			t.Errorf("DecodeBox: got error %v, want %v", err, avc.ErrLengthSize)
		}
		if _, err := mp4.DecodeBoxSR(0, bits.NewFixedSliceReader(data)); !errors.Is(err, avc.ErrLengthSize) {
			t.Errorf("DecodeBoxSR: got error %v, want %v", err, avc.ErrLengthSize)
		}
	})
}
