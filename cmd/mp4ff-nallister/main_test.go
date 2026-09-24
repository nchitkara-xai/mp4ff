package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/Eyevinn/mp4ff/avc"
	"github.com/Eyevinn/mp4ff/mp4"
)

func TestOptions(t *testing.T) {
	cases := []struct {
		desc        string
		args        []string
		expectedErr bool
		goldenOut   string
	}{
		{desc: "no args", args: []string{appName}, expectedErr: true},
		{desc: "unknown args", args: []string{appName, "-x"}, expectedErr: true},
		{desc: "non-existing file", args: []string{appName, "infile.mp4"}, expectedErr: true},
		{desc: "bad file", args: []string{appName, "main.go"}, expectedErr: true},
		{desc: "annexB, bad file", args: []string{appName, "-annexb", "main.go"}, expectedErr: true},
		{desc: "annexB, non-existing file", args: []string{appName, "-annexb", "none.264"}, expectedErr: true},
		{desc: "annexBH264", args: []string{appName, "-annexb", "-ps", "testdata/4pics.264"},
			goldenOut: "testdata/golden_4pics_h264.txt", expectedErr: false},
		{desc: "annexBBadCodec", args: []string{appName, "-annexb", "-c", "av1", "testdata/4pics.264"},
			expectedErr: true},
		{desc: "initFile", args: []string{appName, "../../mp4/testdata/init.mp4"}, expectedErr: false},
		{desc: "progH264", args: []string{appName, "-ps", "-m", "4", "../../mp4/testdata/prog_8s.mp4"},
			goldenOut: "testdata/golden_prot_h264_4pics.txt", expectedErr: false},
		{desc: "mp4H264", args: []string{appName, "testdata/h264.mp4"},
			goldenOut: "testdata/golden_h264_mp4.txt", expectedErr: false},
		{desc: "annexBHEVC", args: []string{appName, "-annexb", "-c", "hevc", "-ps", "testdata/hevc.265"},
			goldenOut: "testdata/golden_hevc_265.txt", expectedErr: false},
		{desc: "annexBHEVC with SEI", args: []string{appName, "-annexb", "-c", "hevc", "-sei", "2", "testdata/hevc.265"},
			goldenOut: "", expectedErr: false},
		{desc: "mp4HEVC", args: []string{appName, "testdata/hevc.mp4"},
			goldenOut: "testdata/golden_hevc_mp4.txt", expectedErr: false},
		{desc: "h264 frag mp4 raw", args: []string{appName, "-m", "6", "-raw", "4", "../../mp4/testdata/prog_8s_dec_dashinit.mp4"},
			goldenOut: "testdata/golden_h264_frag_raw.txt", expectedErr: false},
		{desc: "avcSeiTime", args: []string{appName, "-sei", "2", "-annexb", "testdata/4pics.264"},
			goldenOut: "testdata/golden_4pic_sei_264.txt", expectedErr: false},
		{desc: "vvc 2s", args: []string{appName, "../../mp4/testdata/vvc_400kbps_2s.mp4"}, expectedErr: false,
			goldenOut: "testdata/golden_vvc_2s.txt"},
		{desc: "vvc annexB", args: []string{appName, "-annexb", "-ps", "-c", "vvc", "testdata/annexb.vvc"}, expectedErr: false,
			goldenOut: "testdata/golden_vvc_annexb.txt"},
		{desc: "version", args: []string{appName, "-version"}, expectedErr: false},
		{desc: "help", args: []string{appName, "-h"}, expectedErr: false},
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

func MakeByteStream(t *testing.T, inFile, outFile string) {
	t.Helper()
	ifd, err := os.Open(inFile)
	if err != nil {
		t.Fatalf("could not open file %s: %s", inFile, err)
	}
	d, err := mp4.DecodeFile(ifd)
	if err != nil {
		t.Fatalf("could not decode file %s: %s", inFile, err)
	}
	fullSamples, err := d.Segments[0].Fragments[0].GetFullSamples(nil)
	if err != nil {
		t.Fatalf("could not get full samples: %s", err)
	}
	byteStream := make([]byte, 0, 1024)
	for i := 0; i <= 5; i++ {
		fs := fullSamples[i]
		bs := avc.ConvertSampleToByteStream(fs.Data)
		byteStream = append(byteStream, bs...)
	}
	err = os.WriteFile(outFile, byteStream, 0644)
	if err != nil {
		t.Fatalf("could not write file %s: %s", outFile, err)
	}
}

// TestBadChunkOffsets checks that a progressive file whose chunk offsets point
// outside the mdat data gives an error instead of a panic.
func TestBadChunkOffsets(t *testing.T) {
	raw, err := os.ReadFile("testdata/h264.mp4")
	if err != nil {
		t.Fatal(err)
	}
	// Move every chunk offset in the first stco far beyond the end of the file.
	stcoPos := bytes.Index(raw, []byte("stco"))
	if stcoPos < 0 {
		t.Fatal("no stco box found")
	}
	entryCount := binary.BigEndian.Uint32(raw[stcoPos+8 : stcoPos+12])
	for e := 0; e < int(entryCount); e++ {
		p := stcoPos + 12 + e*4
		offset := binary.BigEndian.Uint32(raw[p : p+4])
		binary.BigEndian.PutUint32(raw[p:p+4], offset+1<<28)
	}
	badFile := filepath.Join(t.TempDir(), "bad_stco.mp4")
	if err := os.WriteFile(badFile, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := run([]string{appName, badFile}, &bytes.Buffer{}); err == nil {
		t.Error("expected error for chunk offsets outside mdat, got nil")
	}
}

// TestShortNaluLengths checks that samples with 2-byte NALU lengths are listed
// like the same samples with 4-byte lengths, apart from their smaller sizes.
func TestShortNaluLengths(t *testing.T) {
	sampleSize := regexp.MustCompile(`\(\d+B\):`)
	cases := []struct {
		desc    string
		inFiles []string
	}{
		{desc: "avc", inFiles: []string{"../../mp4/testdata/init.mp4", "../../mp4/testdata/1.m4s"}},
		{desc: "hevc", inFiles: []string{"../../mp4/testdata/hvc1_init.mp4", "../../mp4/testdata/hvc1_seg_1.m4s"}},
		{desc: "vvc", inFiles: []string{"../../mp4/testdata/vvc_400kbps_2s.mp4"}},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			init, samples := readSamples(t, c.inFiles...)
			want4Frag, want4Prog, _ := build(t, init, samples, 4)
			got2Frag, got2Prog, sizes := build(t, init, samples, 2)
			for i, files := range [][2]string{{want4Frag, got2Frag}, {want4Prog, got2Prog}} {
				var outs [2]string
				for j, file := range files {
					out := bytes.Buffer{}
					if err := run([]string{appName, file}, &out); err != nil {
						t.Fatal(err)
					}
					outs[j] = out.String()
				}
				nr := 0
				want := sampleSize.ReplaceAllStringFunc(outs[0], func(string) string {
					nr++
					return fmt.Sprintf("(%dB):", sizes[nr-1])
				})
				if nr != len(samples) || outs[1] != want {
					t.Errorf("file %d: got\n%s\nwant\n%s", i, outs[1], want)
				}
			}
		})
	}
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
// NALU lengths, as a fragmented file and as a progressive copy of it, and
// returns the new sample sizes.
func build(t *testing.T, init *mp4.InitSegment, samples []mp4.FullSample, lengthSize int) (fragmented, progressive string,
	sizes []uint32) {
	t.Helper()
	se := init.Moov.Trak.Mdia.Minf.Stbl.Stsd.Children[0].(*mp4.VisualSampleEntryBox)
	switch {
	case se.AvcC != nil:
		se.AvcC.NaluLengthSize = byte(lengthSize)
	case se.HvcC != nil:
		se.HvcC.LengthSizeMinusOne = byte(lengthSize - 1)
	default:
		se.VvcC.LengthSizeMinusOne = byte(lengthSize - 1)
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
		sizes = append(sizes, fs.Size)
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
	return fragmented, progressive, sizes
}
