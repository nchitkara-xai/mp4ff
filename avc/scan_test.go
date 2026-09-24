package avc_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Eyevinn/mp4ff/avc"
	"github.com/go-test/deep"
)

// field returns n as a big-endian length field of lengthSize bytes.
func field(t *testing.T, lengthSize, n int) []byte {
	t.Helper()
	if n >= 1<<(8*lengthSize) {
		t.Fatalf("%d does not fit in a %d-byte length field", n, lengthSize)
	}
	b := make([]byte, lengthSize)
	for i := lengthSize - 1; i >= 0; i-- {
		b[i] = byte(n)
		n >>= 8
	}
	return b
}

// frame precedes each nalu with a length field of lengthSize bytes.
func frame(t *testing.T, lengthSize int, nalus ...[]byte) []byte {
	t.Helper()
	var sample []byte
	for _, nalu := range nalus {
		sample = append(sample, field(t, lengthSize, len(nalu))...)
		sample = append(sample, nalu...)
	}
	return sample
}

func cat(parts ...[]byte) []byte {
	return bytes.Join(parts, nil)
}

var (
	spsNalu    = []byte{0x67, 0x42, 0x00}
	ppsNalu    = []byte{0x68, 0xce}
	seiNalu    = []byte{0x06, 0x05}
	idrNalu    = []byte{0x65, 0x88}
	nonIDRNalu = []byte{0x41, 0x9a}
	longNalu   = append([]byte{0x65}, make([]byte, 299)...) // longer than 255 bytes
)

type scanResult struct {
	Types, TypesUpToVideo     []avc.NaluType
	ContainsSPS, IsIDR, HasPS bool
	SPS, PPS                  [][]byte
	Nalus                     [][]byte
	NalusErr                  bool
}

// scanCases build each sample for a given length size, and hold for each of
// lengthSizes, by default 1, 2, and 4.
var scanCases = []struct {
	desc        string
	lengthSizes []int
	sample      func(t *testing.T, ls int) []byte
	want        scanResult
	// wantByteStream is the ConvertSampleToByteStream output for ls 4.
	wantByteStream string
}{
	{
		desc:           "empty",
		sample:         func(t *testing.T, ls int) []byte { return []byte{} },
		want:           scanResult{NalusErr: true},
		wantByteStream: "",
	},
	{
		desc:           "shorter than a length field",
		sample:         func(t *testing.T, ls int) []byte { return make([]byte, ls-1) },
		want:           scanResult{NalusErr: true},
		wantByteStream: "000000",
	},
	{
		desc:           "length field only",
		sample:         func(t *testing.T, ls int) []byte { return field(t, ls, ls) },
		want:           scanResult{Types: []avc.NaluType{}, TypesUpToVideo: []avc.NaluType{}, Nalus: [][]byte{}},
		wantByteStream: "00000004",
	},
	{
		desc:           "zero length field only",
		sample:         func(t *testing.T, ls int) []byte { return field(t, ls, 0) },
		want:           scanResult{Types: []avc.NaluType{}, TypesUpToVideo: []avc.NaluType{}, Nalus: [][]byte{}},
		wantByteStream: "00000001",
	},
	{
		desc:   "trailing zero length field",
		sample: func(t *testing.T, ls int) []byte { return cat(frame(t, ls, spsNalu), field(t, ls, 0)) },
		want: scanResult{Types: []avc.NaluType{7}, TypesUpToVideo: []avc.NaluType{7}, ContainsSPS: true,
			SPS: [][]byte{spsNalu}, Nalus: [][]byte{spsNalu}},
		wantByteStream: "0000000167420000000001",
	},
	{
		desc:   "trailing nonzero length field",
		sample: func(t *testing.T, ls int) []byte { return cat(frame(t, ls, spsNalu), field(t, ls, 9)) },
		want: scanResult{Types: []avc.NaluType{7}, TypesUpToVideo: []avc.NaluType{7}, ContainsSPS: true,
			SPS: [][]byte{spsNalu}, Nalus: [][]byte{spsNalu}},
		wantByteStream: "0000000167420000000009",
	},
	{
		desc:   "trailing partial length field",
		sample: func(t *testing.T, ls int) []byte { return cat(frame(t, ls, spsNalu), make([]byte, ls-1)) },
		want: scanResult{Types: []avc.NaluType{7}, TypesUpToVideo: []avc.NaluType{7}, ContainsSPS: true,
			SPS: [][]byte{spsNalu}, Nalus: [][]byte{spsNalu}},
		wantByteStream: "00000001674200000000",
	},
	{
		desc:   "zero length nalu first",
		sample: func(t *testing.T, ls int) []byte { return cat(field(t, ls, 0), frame(t, ls, spsNalu)) },
		want: scanResult{Types: []avc.NaluType{}, TypesUpToVideo: []avc.NaluType{},
			Nalus: [][]byte{{}, spsNalu}},
		wantByteStream: "0000000100000001674200",
	},
	{
		desc:   "length beyond sample",
		sample: func(t *testing.T, ls int) []byte { return cat(frame(t, ls, spsNalu), field(t, ls, 16), ppsNalu) },
		want: scanResult{Types: []avc.NaluType{7}, TypesUpToVideo: []avc.NaluType{7}, ContainsSPS: true,
			SPS: [][]byte{spsNalu}, NalusErr: true},
		wantByteStream: "000000016742000000001068ce",
	},
	{
		desc: "maximum length field",
		sample: func(t *testing.T, ls int) []byte {
			return cat(frame(t, ls, spsNalu), bytes.Repeat([]byte{0xff}, ls), ppsNalu)
		},
		want: scanResult{Types: []avc.NaluType{7}, TypesUpToVideo: []avc.NaluType{7}, ContainsSPS: true,
			SPS: [][]byte{spsNalu}, NalusErr: true},
		wantByteStream: "00000001674200ffffffff68ce",
	},
	{
		desc:        "length beyond sample in the upper byte",
		lengthSizes: []int{2, 4},
		sample: func(t *testing.T, ls int) []byte {
			return cat(frame(t, ls, spsNalu), field(t, ls, map[int]int{2: 0x0103, 4: 0x00010003}[ls]), ppsNalu, []byte{0})
		},
		want: scanResult{Types: []avc.NaluType{7}, TypesUpToVideo: []avc.NaluType{7}, ContainsSPS: true,
			SPS: [][]byte{spsNalu}, NalusErr: true},
		wantByteStream: "000000016742000001000368ce00",
	},
	{
		desc:   "access unit",
		sample: func(t *testing.T, ls int) []byte { return frame(t, ls, seiNalu, spsNalu, ppsNalu, idrNalu, nonIDRNalu) },
		want: scanResult{Types: []avc.NaluType{6, 7, 8, 5, 1}, TypesUpToVideo: []avc.NaluType{6, 7, 8, 5},
			ContainsSPS: true, IsIDR: true, HasPS: true, SPS: [][]byte{spsNalu}, PPS: [][]byte{ppsNalu},
			Nalus: [][]byte{seiNalu, spsNalu, ppsNalu, idrNalu, nonIDRNalu}},
		wantByteStream: "000000010605" + "00000001674200" + "0000000168ce" + "000000016588" + "00000001419a",
	},
	{
		desc:        "long nalu",
		lengthSizes: []int{2, 4},
		sample:      func(t *testing.T, ls int) []byte { return frame(t, ls, spsNalu, longNalu, nonIDRNalu) },
		want: scanResult{Types: []avc.NaluType{7, 5, 1}, TypesUpToVideo: []avc.NaluType{7, 5}, ContainsSPS: true,
			IsIDR: true, SPS: [][]byte{spsNalu}, Nalus: [][]byte{spsNalu, longNalu, nonIDRNalu}},
		wantByteStream: "00000001674200" + "0000000165" + strings.Repeat("00", 299) + "00000001419a",
	},
	{
		desc:   "parameter set after video",
		sample: func(t *testing.T, ls int) []byte { return frame(t, ls, idrNalu, spsNalu) },
		want: scanResult{Types: []avc.NaluType{5, 7}, TypesUpToVideo: []avc.NaluType{5}, ContainsSPS: true,
			IsIDR: true, Nalus: [][]byte{idrNalu, spsNalu}},
		wantByteStream: "00000001658800000001674200",
	},
}

// TestScan checks the scanning functions for each length size, and pins the
// 4-byte functions, including how they handle bad length fields.
func TestScan(t *testing.T) {
	for _, c := range scanCases {
		lengthSizes := c.lengthSizes
		if lengthSizes == nil {
			lengthSizes = []int{1, 2, 4}
		}
		for _, ls := range lengthSizes {
			t.Run(fmt.Sprintf("%s %d-byte", c.desc, ls), func(t *testing.T) {
				sample := c.sample(t, ls)
				got := scanResult{
					Types:          avc.FindNaluTypesWithLengthSize(sample, ls),
					TypesUpToVideo: avc.FindNaluTypesUpToFirstVideoNALUWithLengthSize(sample, ls),
					ContainsSPS:    avc.ContainsNaluTypeWithLengthSize(sample, avc.NALU_SPS, ls),
					IsIDR:          avc.IsIDRSampleWithLengthSize(sample, ls),
					HasPS:          avc.HasParameterSetsWithLengthSize(sample, ls),
				}
				got.SPS, got.PPS = avc.GetParameterSetsWithLengthSize(sample, ls)
				var err error
				got.Nalus, err = avc.GetNalusFromSampleWithLengthSize(sample, ls)
				got.NalusErr = err != nil
				if diff := deep.Equal(got, c.want); diff != nil {
					t.Error(diff)
				}
				if ls != 4 {
					return
				}
				got4 := scanResult{
					Types:          avc.FindNaluTypes(sample),
					TypesUpToVideo: avc.FindNaluTypesUpToFirstVideoNALU(sample),
					ContainsSPS:    avc.ContainsNaluType(sample, avc.NALU_SPS),
					IsIDR:          avc.IsIDRSample(sample),
					HasPS:          avc.HasParameterSets(sample),
				}
				got4.SPS, got4.PPS = avc.GetParameterSets(sample)
				got4.Nalus, err = avc.GetNalusFromSample(sample)
				got4.NalusErr = err != nil
				if diff := deep.Equal(got4, c.want); diff != nil {
					t.Errorf("4-byte functions: %v", diff)
				}
				if bs := hex.EncodeToString(avc.ConvertSampleToByteStream(sample)); bs != c.wantByteStream {
					t.Errorf("ConvertSampleToByteStream: got %s, want %s", bs, c.wantByteStream)
				}
			})
		}
	}
}

func TestScanWithBadLengthSize(t *testing.T) {
	for _, sample := range [][]byte{frame(t, 4, seiNalu, spsNalu, ppsNalu, idrNalu), {0x67}} {
		for _, ls := range []int{-1, 0, 3, 5} {
			spss, ppss := avc.GetParameterSetsWithLengthSize(sample, ls)
			if avc.FindNaluTypesWithLengthSize(sample, ls) != nil ||
				avc.FindNaluTypesUpToFirstVideoNALUWithLengthSize(sample, ls) != nil ||
				avc.ContainsNaluTypeWithLengthSize(sample, avc.NALU_SPS, ls) ||
				avc.IsIDRSampleWithLengthSize(sample, ls) ||
				avc.HasParameterSetsWithLengthSize(sample, ls) ||
				spss != nil || ppss != nil {
				t.Errorf("length size %d: got a non-empty result", ls)
			}
			if _, err := avc.GetNalusFromSampleWithLengthSize(sample, ls); !errors.Is(err, avc.ErrLengthSize) {
				t.Errorf("length size %d: got error %v, want %v", ls, err, avc.ErrLengthSize)
			}
		}
	}
}
