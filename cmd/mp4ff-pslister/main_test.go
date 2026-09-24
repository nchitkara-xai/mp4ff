package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

// TestShortNaluLengths checks that parameter sets in an avcC with 2-byte NALU
// lengths are listed, but that pslister refuses to look for them in samples
// with such lengths, while still finding them in samples with 4-byte lengths.
func TestShortNaluLengths(t *testing.T) {
	cases := []struct {
		desc           string
		inFile         string
		naluLengthSize byte
		stripPS        bool
		goldenOut      string
		wantOut        string
		wantErr        string
	}{
		{desc: "PS in avcC", inFile: "../../mp4/testdata/init.mp4", naluLengthSize: 2,
			goldenOut: "testdata/golden_h264mp4.txt"},
		{desc: "progressive, PS in 4-byte samples", inFile: "../../mp4/testdata/prog_8s.mp4", stripPS: true,
			wantOut: "SPS 1 len 25B: 6764001e"},
		{desc: "progressive, PS in 2-byte samples", inFile: "../../mp4/testdata/prog_8s.mp4", naluLengthSize: 2, stripPS: true,
			wantErr: "2-byte NALU lengths not supported"},
		{desc: "fragmented, PS in 2-byte samples", inFile: "../../mp4/testdata/prog_8s_dec_dashinit.mp4", naluLengthSize: 2,
			stripPS: true, wantErr: "2-byte NALU lengths not supported"},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			ifh, err := os.Open(c.inFile)
			if err != nil {
				t.Fatal(err)
			}
			f, err := mp4.DecodeFile(ifh)
			ifh.Close()
			if err != nil {
				t.Fatal(err)
			}
			moov := f.Moov
			if moov == nil {
				moov = f.Init.Moov
			}
			moovSize := moov.Size()
			var avcC *mp4.AvcCBox
			for _, trak := range moov.Traks {
				if trak.Mdia.Hdlr.HandlerType == "vide" {
					avcC = trak.Mdia.Minf.Stbl.Stsd.AvcX.AvcC
				}
			}
			avcC.NaluLengthSize = c.naluLengthSize
			if c.stripPS {
				avcC.SPSnalus, avcC.PPSnalus = nil, nil
			}
			if !f.IsFragmented() {
				// The mdat follows the moov, so its chunks move with the moov size
				shrink := moovSize - moov.Size()
				for _, trak := range moov.Traks {
					stbl := trak.Mdia.Minf.Stbl
					if stbl.Stco != nil {
						for i := range stbl.Stco.ChunkOffset {
							stbl.Stco.ChunkOffset[i] -= uint32(shrink)
						}
					}
					if stbl.Co64 != nil {
						for i := range stbl.Co64.ChunkOffset {
							stbl.Co64.ChunkOffset[i] -= shrink
						}
					}
				}
			}
			outFile := filepath.Join(t.TempDir(), "short_lengths.mp4")
			ofh, err := os.Create(outFile)
			if err != nil {
				t.Fatal(err)
			}
			err = f.Encode(ofh)
			ofh.Close()
			if err != nil {
				t.Fatal(err)
			}

			gotOut := bytes.Buffer{}
			err = run([]string{appName, "-i", outFile}, &gotOut)
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Errorf("got error %v, want %s", err, c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if c.goldenOut != "" && gotOut.String() != getExpected(t, c.goldenOut) {
				t.Errorf("got %s", gotOut.String())
			}
			if !strings.Contains(gotOut.String(), c.wantOut) {
				t.Errorf("got %s, want it to contain %s", gotOut.String(), c.wantOut)
			}
		})
	}
}
