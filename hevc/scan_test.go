package hevc_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/Eyevinn/mp4ff/hevc"
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
	vpsNalu   = []byte{0x40, 0x01}
	spsNalu   = []byte{0x42, 0x01, 0x01}
	ppsNalu   = []byte{0x44, 0x01}
	seiNalu   = []byte{0x4e, 0x01}
	idrNalu   = []byte{0x26, 0x01}
	trailNalu = []byte{0x02, 0x01}
	longNalu  = append([]byte{0x26, 0x01}, make([]byte, 298)...) // longer than 255 bytes
)

type scanResult struct {
	Types, TypesUpToVideo            []hevc.NaluType
	ContainsSPS, IsRAP, IsIDR, HasPS bool
	VPS, SPS, PPS                    [][]byte
	ByLayer                          map[byte][][]byte
}

// scanCases build each sample for a given length size, and hold for each of
// lengthSizes, by default 1, 2, and 4.
var scanCases = []struct {
	desc        string
	lengthSizes []int
	sample      func(t *testing.T, ls int) []byte
	want        scanResult
}{
	{
		desc:   "empty",
		sample: func(t *testing.T, ls int) []byte { return []byte{} },
		want:   scanResult{Types: []hevc.NaluType{}, TypesUpToVideo: []hevc.NaluType{}, ByLayer: map[byte][][]byte{}},
	},
	{
		desc:   "shorter than a length field",
		sample: func(t *testing.T, ls int) []byte { return make([]byte, ls-1) },
		want:   scanResult{Types: []hevc.NaluType{}, TypesUpToVideo: []hevc.NaluType{}, ByLayer: map[byte][][]byte{}},
	},
	{
		desc:   "length field only",
		sample: func(t *testing.T, ls int) []byte { return field(t, ls, ls) },
		want:   scanResult{Types: []hevc.NaluType{}, TypesUpToVideo: []hevc.NaluType{}, ByLayer: map[byte][][]byte{}},
	},
	{
		desc:   "zero length field only",
		sample: func(t *testing.T, ls int) []byte { return field(t, ls, 0) },
		want:   scanResult{Types: []hevc.NaluType{}, TypesUpToVideo: []hevc.NaluType{}, ByLayer: map[byte][][]byte{}},
	},
	{
		desc:   "trailing zero length field",
		sample: func(t *testing.T, ls int) []byte { return cat(frame(t, ls, spsNalu), field(t, ls, 0)) },
		want: scanResult{Types: []hevc.NaluType{33}, TypesUpToVideo: []hevc.NaluType{33}, ContainsSPS: true,
			SPS: [][]byte{spsNalu}, ByLayer: map[byte][][]byte{0: {spsNalu}}},
	},
	{
		desc:   "trailing nonzero length field",
		sample: func(t *testing.T, ls int) []byte { return cat(frame(t, ls, spsNalu), field(t, ls, 9)) },
		want: scanResult{Types: []hevc.NaluType{33}, TypesUpToVideo: []hevc.NaluType{33}, ContainsSPS: true,
			SPS: [][]byte{spsNalu}, ByLayer: map[byte][][]byte{0: {spsNalu}}},
	},
	{
		desc:   "trailing partial length field",
		sample: func(t *testing.T, ls int) []byte { return cat(frame(t, ls, spsNalu), make([]byte, ls-1)) },
		want: scanResult{Types: []hevc.NaluType{33}, TypesUpToVideo: []hevc.NaluType{33}, ContainsSPS: true,
			SPS: [][]byte{spsNalu}, ByLayer: map[byte][][]byte{0: {spsNalu}}},
	},
	{
		desc:   "zero length nalu first",
		sample: func(t *testing.T, ls int) []byte { return cat(field(t, ls, 0), frame(t, ls, spsNalu)) },
		want:   scanResult{Types: []hevc.NaluType{}, TypesUpToVideo: []hevc.NaluType{}, ByLayer: map[byte][][]byte{}},
	},
	{
		desc:   "one byte nalu",
		sample: func(t *testing.T, ls int) []byte { return frame(t, ls, spsNalu[:1], spsNalu) },
		want: scanResult{Types: []hevc.NaluType{33, 33}, TypesUpToVideo: []hevc.NaluType{33, 33}, ContainsSPS: true,
			SPS: [][]byte{spsNalu[:1], spsNalu}, ByLayer: map[byte][][]byte{}},
	},
	{
		desc:   "length beyond sample",
		sample: func(t *testing.T, ls int) []byte { return cat(frame(t, ls, spsNalu), field(t, ls, 16), ppsNalu) },
		want: scanResult{Types: []hevc.NaluType{33}, TypesUpToVideo: []hevc.NaluType{33}, ContainsSPS: true,
			SPS: [][]byte{spsNalu}, ByLayer: map[byte][][]byte{0: {spsNalu}}},
	},
	{
		desc: "maximum length field",
		sample: func(t *testing.T, ls int) []byte {
			return cat(frame(t, ls, spsNalu), bytes.Repeat([]byte{0xff}, ls), ppsNalu)
		},
		want: scanResult{Types: []hevc.NaluType{33}, TypesUpToVideo: []hevc.NaluType{33}, ContainsSPS: true,
			SPS: [][]byte{spsNalu}, ByLayer: map[byte][][]byte{0: {spsNalu}}},
	},
	{
		desc:        "length beyond sample in the upper byte",
		lengthSizes: []int{2, 4},
		sample: func(t *testing.T, ls int) []byte {
			return cat(frame(t, ls, spsNalu), field(t, ls, map[int]int{2: 0x0103, 4: 0x00010003}[ls]), ppsNalu, []byte{0})
		},
		want: scanResult{Types: []hevc.NaluType{33}, TypesUpToVideo: []hevc.NaluType{33}, ContainsSPS: true,
			SPS: [][]byte{spsNalu}, ByLayer: map[byte][][]byte{0: {spsNalu}}},
	},
	{
		desc:        "long nalu",
		lengthSizes: []int{2, 4},
		sample:      func(t *testing.T, ls int) []byte { return frame(t, ls, spsNalu, longNalu, trailNalu) },
		want: scanResult{Types: []hevc.NaluType{33, 19, 1}, TypesUpToVideo: []hevc.NaluType{33, 19}, ContainsSPS: true,
			IsRAP: true, IsIDR: true, SPS: [][]byte{spsNalu}, ByLayer: map[byte][][]byte{0: {spsNalu, longNalu, trailNalu}}},
	},
	{
		desc: "access unit",
		sample: func(t *testing.T, ls int) []byte {
			return frame(t, ls, seiNalu, vpsNalu, spsNalu, ppsNalu, idrNalu, trailNalu)
		},
		want: scanResult{Types: []hevc.NaluType{39, 32, 33, 34, 19, 1}, TypesUpToVideo: []hevc.NaluType{39, 32, 33, 34, 19},
			ContainsSPS: true, IsRAP: true, IsIDR: true, HasPS: true,
			VPS: [][]byte{vpsNalu}, SPS: [][]byte{spsNalu}, PPS: [][]byte{ppsNalu},
			ByLayer: map[byte][][]byte{0: {seiNalu, vpsNalu, spsNalu, ppsNalu, idrNalu, trailNalu}}},
	},
	{
		desc:   "parameter set after video",
		sample: func(t *testing.T, ls int) []byte { return frame(t, ls, trailNalu, spsNalu) },
		want: scanResult{Types: []hevc.NaluType{1, 33}, TypesUpToVideo: []hevc.NaluType{1}, ContainsSPS: true,
			ByLayer: map[byte][][]byte{0: {trailNalu, spsNalu}}},
	},
}

// TestScan checks the scanning functions and SplitNalusByLayerID for each
// length size, and pins the 4-byte functions, including how they handle bad
// length fields.
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
					Types:          hevc.FindNaluTypesWithLengthSize(sample, ls),
					TypesUpToVideo: hevc.FindNaluTypesUpToFirstVideoNaluWithLengthSize(sample, ls),
					ContainsSPS:    hevc.ContainsNaluTypeWithLengthSize(sample, hevc.NALU_SPS, ls),
					IsRAP:          hevc.IsRAPSampleWithLengthSize(sample, ls),
					IsIDR:          hevc.IsIDRSampleWithLengthSize(sample, ls),
					HasPS:          hevc.HasParameterSetsWithLengthSize(sample, ls),
					ByLayer:        hevc.SplitNalusByLayerID(sample, ls),
				}
				got.VPS, got.SPS, got.PPS = hevc.GetParameterSetsWithLengthSize(sample, ls)
				if diff := deep.Equal(got, c.want); diff != nil {
					t.Error(diff)
				}
				if ls != 4 {
					return
				}
				got4 := scanResult{
					Types:          hevc.FindNaluTypes(sample),
					TypesUpToVideo: hevc.FindNaluTypesUpToFirstVideoNalu(sample),
					ContainsSPS:    hevc.ContainsNaluType(sample, hevc.NALU_SPS),
					IsRAP:          hevc.IsRAPSample(sample),
					IsIDR:          hevc.IsIDRSample(sample),
					HasPS:          hevc.HasParameterSets(sample),
					ByLayer:        got.ByLayer,
				}
				got4.VPS, got4.SPS, got4.PPS = hevc.GetParameterSets(sample)
				if diff := deep.Equal(got4, c.want); diff != nil {
					t.Errorf("4-byte functions: %v", diff)
				}
			})
		}
	}
}

func TestScanWithBadLengthSize(t *testing.T) {
	sample := frame(t, 4, vpsNalu, spsNalu, ppsNalu, idrNalu)
	for _, ls := range []int{-1, 0, 3, 5} {
		vpss, spss, ppss := hevc.GetParameterSetsWithLengthSize(sample, ls)
		if len(hevc.FindNaluTypesWithLengthSize(sample, ls)) != 0 ||
			len(hevc.FindNaluTypesUpToFirstVideoNaluWithLengthSize(sample, ls)) != 0 ||
			hevc.ContainsNaluTypeWithLengthSize(sample, hevc.NALU_SPS, ls) ||
			hevc.IsRAPSampleWithLengthSize(sample, ls) ||
			hevc.IsIDRSampleWithLengthSize(sample, ls) ||
			hevc.HasParameterSetsWithLengthSize(sample, ls) ||
			len(vpss) != 0 || len(spss) != 0 || len(ppss) != 0 {
			t.Errorf("length size %d: got a non-empty result", ls)
		}
	}
}
