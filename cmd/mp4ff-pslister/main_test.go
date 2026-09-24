package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Eyevinn/mp4ff/avc"
	"github.com/Eyevinn/mp4ff/mp4"
)

const (
	avc_sps  = "6764001eacd940a02ff9610000030001000003003c8f162d96"
	avc_pps  = "68ebecb22c"
	hevc_vps = "40010c01ffff016000000300900000030000030078959809"
	hevc_sps = "420101016000000300900000030000030078a00502016965959a4932bc05a80808082000000300200000030321"
	hevc_pps = "4401c172b46240"
)

func TestCommandLines(t *testing.T) {
	cases := []struct {
		desc        string
		args        []string
		expectedErr bool
		goldenOut   string
	}{
		{desc: "h264 segment without PS", args: []string{appName, "-v", "-i", "../../mp4/testdata/1.m4s"},
			expectedErr: true},
		{desc: "help", args: []string{appName, "-h"}, expectedErr: false},
		{desc: "version", args: []string{appName, "-version"}, expectedErr: false},
		{desc: "no args", args: []string{appName}, expectedErr: true},
		{desc: "unknown args", args: []string{appName, "-x"}, expectedErr: true},
		{desc: "non-existing file", args: []string{appName, "-i", "infile.mp4"}, expectedErr: true},
		{desc: "bad file - no ps", args: []string{appName, "-i", "main.go"}, expectedErr: true},
		{desc: "segment wo ps", args: []string{appName, "-i", "../../mp4/testdata/1.m4s"}, expectedErr: true},
		{desc: "h264mp4", args: []string{appName, "-i", "../../mp4/testdata/init.mp4"},
			goldenOut: "testdata/golden_h264mp4.txt", expectedErr: false},
		{desc: "h264mp4 verbose", args: []string{appName, "-v", "-i", "../../mp4/testdata/init.mp4"},
			goldenOut: "testdata/golden_h264mp4_verbose.txt", expectedErr: false},
		{desc: "h264 sps+pps", args: []string{appName, "-sps", avc_sps, "-pps", avc_pps},
			goldenOut: "testdata/golden_avc_sps_pss.txt", expectedErr: false},
		{desc: "h264 annexb", args: []string{appName, "-i", "testdata/4pics.264"},
			goldenOut: "testdata/golden_annexb_h264.txt", expectedErr: false},
		{desc: "hevcmp4", args: []string{appName, "-i", "../../mp4/testdata/ed_hevc.mp4"},
			goldenOut: "testdata/golden_hevc_mp4.txt", expectedErr: false},
		{desc: "hevcmp4 verbose", args: []string{appName, "-v", "-i", "../../mp4/testdata/ed_hevc.mp4"},
			goldenOut: "testdata/golden_hevc_mp4_verbose.txt", expectedErr: false},
		{desc: "hevc vps+sps+pps", args: []string{appName, "-vps", hevc_vps, "-sps", hevc_sps, "-pps", hevc_pps},
			goldenOut: "testdata/golden_hevc_vps_sps_pps.txt", expectedErr: false},
		{desc: "hevc annexb", args: []string{appName, "-c", "hevc", "-i", "testdata/hevc.265"},
			goldenOut: "testdata/golden_hevc_265.txt", expectedErr: false},
		{desc: "av1mp4", args: []string{appName, "-i", "../../mp4/testdata/av1_init.mp4"},
			goldenOut: "testdata/golden_av1_mp4.txt", expectedErr: false},
		{desc: "av1mp4 verbose", args: []string{appName, "-v", "-i", "../../mp4/testdata/av1_init.mp4"},
			goldenOut: "testdata/golden_av1_mp4_verbose.txt", expectedErr: false},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			gotOut := bytes.Buffer{}
			err := run(c.args, &gotOut)
			if c.expectedErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %s", err)
				return
			}
			if c.goldenOut != "" {
				expectedString := getExpected(t, c.goldenOut)
				gotString := gotOut.String()
				if gotString != expectedString {
					t.Errorf("expected %s, got %s", expectedString, gotString)
				}
			}
		})
	}
}

func getExpected(t *testing.T, filename string) string {
	t.Helper()
	b, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("could not read golden file %s: %s", filename, err)
	}
	r := bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
	return string(r)
}

// TestShortNaluLengths checks that parameter sets are listed from the samples
// of files with 1- and 2-byte NALU lengths as from the same files with 4-byte
// lengths.
func TestShortNaluLengths(t *testing.T) {
	sps, _ := hex.DecodeString(avc_sps)
	pps, _ := hex.DecodeString(avc_pps)
	small := mp4.CreateEmptyInit()
	if err := small.AddEmptyTrack(90000, "video", "und").SetAVCDescriptor("avc3", [][]byte{sps}, [][]byte{pps}, false); err != nil {
		t.Fatal(err)
	}
	smallSamples := []mp4.FullSample{{Sample: mp4.Sample{Flags: mp4.SyncSampleFlags, Dur: 3000},
		Data: frame(t, 4, [][]byte{sps, pps, {0x65, 0x88, 0x84}})}}

	cases := []struct {
		desc       string
		inFiles    []string
		lengthSize int
	}{
		{desc: "avc3", inFiles: []string{"../../mp4/testdata/init.mp4", "../../mp4/testdata/1.m4s"}, lengthSize: 2},
		{desc: "hev1", inFiles: []string{"../../mp4/testdata/hvc1_init.mp4", "../../mp4/testdata/hvc1_seg_1.m4s"}, lengthSize: 2},
		{desc: "avc3 small sample", lengthSize: 1},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			init, samples := small, smallSamples
			if c.inFiles != nil {
				init, samples = readSamples(t, c.inFiles...)
				moveParameterSets(t, init, samples)
			}
			want4Frag, want4Prog := build(t, init, samples, 4)
			gotFrag, gotProg := build(t, init, samples, c.lengthSize)
			for i, files := range [][2]string{{want4Frag, gotFrag}, {want4Prog, gotProg}} {
				var outs [2]string
				for j, file := range files {
					out := bytes.Buffer{}
					if err := run([]string{appName, "-i", file}, &out); err != nil {
						t.Fatal(err)
					}
					outs[j] = out.String()
				}
				if !strings.Contains(outs[0], "SPS 1 len") || outs[1] != outs[0] {
					t.Errorf("file %d: got\n%s\nwant\n%s", i, outs[1], outs[0])
				}
			}
		})
	}
}

// moveParameterSets moves the parameter sets of the decoder configuration into
// the first sample, as in avc3 and hev1 files.
func moveParameterSets(t *testing.T, init *mp4.InitSegment, samples []mp4.FullSample) {
	stsd := init.Moov.Trak.Mdia.Minf.Stbl.Stsd
	var ps [][]byte
	if se := stsd.AvcX; se != nil {
		ps = append(se.AvcC.SPSnalus, se.AvcC.PPSnalus...)
		se.AvcC.SPSnalus, se.AvcC.PPSnalus = nil, nil
		se.SetType("avc3")
	} else {
		se := stsd.HvcX
		for _, array := range se.HvcC.NaluArrays {
			ps = append(ps, array.Nalus...)
		}
		se.HvcC.NaluArrays = nil
		se.SetType("hev1")
	}
	samples[0].Data = append(frame(t, 4, ps), samples[0].Data...)
}

// readSamples returns the init segment and the samples of fragmented files.
func readSamples(t *testing.T, inFiles ...string) (*mp4.InitSegment, []mp4.FullSample) {
	t.Helper()
	var init *mp4.InitSegment
	var samples []mp4.FullSample
	for _, inFile := range inFiles {
		f, err := mp4.ReadMP4File(inFile)
		if err != nil {
			t.Fatal(err)
		}
		if f.Init != nil {
			init = f.Init
		}
		for _, seg := range f.Segments {
			for _, frag := range seg.Fragments {
				fss, err := frag.GetFullSamples(init.Moov.Mvex.Trex)
				if err != nil {
					t.Fatal(err)
				}
				samples = append(samples, fss...)
			}
		}
	}
	return init, samples
}

// frame precedes each nalu with a length field of lengthSize bytes.
func frame(t *testing.T, lengthSize int, nalus [][]byte) []byte {
	t.Helper()
	var sample []byte
	for _, nalu := range nalus {
		if len(nalu) >= 1<<(8*lengthSize) {
			t.Fatalf("%d-byte nalu does not fit a %d-byte length field", len(nalu), lengthSize)
		}
		lengthField := binary.BigEndian.AppendUint32(nil, uint32(len(nalu)))
		sample = append(append(sample, lengthField[4-lengthSize:]...), nalu...)
	}
	return sample
}

// build writes init and the samples, re-framed from 4-byte to lengthSize-byte
// NALU lengths, as a fragmented file and as a progressive copy of it.
func build(t *testing.T, init *mp4.InitSegment, samples []mp4.FullSample, lengthSize int) (fragmented, progressive string) {
	t.Helper()
	stsd := init.Moov.Trak.Mdia.Minf.Stbl.Stsd
	if stsd.AvcX != nil {
		stsd.AvcX.AvcC.NaluLengthSize = byte(lengthSize)
	} else {
		stsd.HvcX.HvcC.LengthSizeMinusOne = byte(lengthSize - 1)
	}
	frag, err := mp4.CreateFragment(1, init.Moov.Trak.Tkhd.TrackID)
	if err != nil {
		t.Fatal(err)
	}
	for _, fs := range samples {
		nalus, err := avc.GetNalusFromSample(fs.Data)
		if err != nil {
			t.Fatal(err)
		}
		fs.Data = frame(t, lengthSize, nalus)
		fs.Size = uint32(len(fs.Data))
		frag.AddFullSample(fs)
	}
	f := mp4.NewFile()
	f.AddChild(init.Ftyp, 0)
	f.AddChild(init.Moov, 0)
	seg := mp4.NewMediaSegment()
	seg.AddFragment(frag)
	f.AddMediaSegment(seg)
	fragBuf, progBuf := bytes.Buffer{}, bytes.Buffer{}
	if err := f.Encode(&fragBuf); err != nil {
		t.Fatal(err)
	}
	decoded, err := mp4.DecodeFile(bytes.NewReader(fragBuf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if err := mp4.Defragment(decoded, bytes.NewReader(fragBuf.Bytes()), &progBuf); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	fragmented, progressive = filepath.Join(dir, "fragmented.mp4"), filepath.Join(dir, "progressive.mp4")
	if err := os.WriteFile(fragmented, fragBuf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(progressive, progBuf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return fragmented, progressive
}
