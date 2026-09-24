package avc

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/go-test/deep"
)

const avcDecoderConfigRecord = "0164001effe100196764001eacd940a02ff9610000030001000003003c8f162d9601000568ebecb22cfdf8f800"
const sps = "6764001eacd940a02ff9610000030001000003003c8f162d96"
const pps = "68ebecb22c"

func TestAvcDecoderConfigRecord(t *testing.T) {
	byteData, _ := hex.DecodeString(avcDecoderConfigRecord)
	spsBytes, _ := hex.DecodeString(sps)
	ppsBytes, _ := hex.DecodeString(pps)

	wanted := DecConfRec{
		AVCProfileIndication: 100,
		ProfileCompatibility: 0,
		AVCLevelIndication:   30,
		SPSnalus:             [][]byte{spsBytes},
		PPSnalus:             [][]byte{ppsBytes},
		ChromaFormat:         1,
		BitDepthLumaMinus1:   0,
		BitDepthChromaMinus1: 0,
		NumSPSExt:            0,
	}

	got, err := DecodeAVCDecConfRec(byteData)
	if err != nil {
		t.Error("Error parsing AVCDecoderConfigurationRecord")
	}
	if diff := deep.Equal(got, wanted); diff != nil {
		t.Error(diff)
	}

	enc := bytes.Buffer{}
	err = got.Encode(&enc)
	if err != nil {
		t.Error("Error encoding AVCDecoderConfigurationRecord")
	}
	if !bytes.Equal(enc.Bytes(), byteData) {
		t.Error("Error encoding AVCDecoderConfigurationRecord")
	}
}

func TestAvcDecoderConfigRecordWithExtraBytes(t *testing.T) {
	//nolint: lll
	byteData, _ := hex.DecodeString("014d4029ffe10026674d40299e5281e022fde028404040500000030010000003032e04000ea6000057e43f13e0a001000468ef752000")
	spsBytes, _ := hex.DecodeString("674d40299e5281e022fde028404040500000030010000003032e04000ea6000057e43f13e0a0")
	ppsBytes, _ := hex.DecodeString("68ef7520")

	wanted := DecConfRec{
		AVCProfileIndication: 77,
		ProfileCompatibility: 64,
		AVCLevelIndication:   41,
		SPSnalus:             [][]byte{spsBytes},
		PPSnalus:             [][]byte{ppsBytes},
		ChromaFormat:         0,
		BitDepthLumaMinus1:   0,
		BitDepthChromaMinus1: 0,
		NumSPSExt:            0,
		NoTrailingInfo:       true,
		SkipBytes:            1,
	}

	got, err := DecodeAVCDecConfRec(byteData)
	if err != nil {
		t.Error("Error parsing AVCDecoderConfigurationRecord")
	}
	if diff := deep.Equal(got, wanted); diff != nil {
		t.Error(diff)
	}

	enc := bytes.Buffer{}
	err = got.Encode(&enc)
	if err != nil {
		t.Error("Error encoding AVCDecoderConfigurationRecord")
	}
	if !bytes.Equal(enc.Bytes(), byteData) {
		t.Error("Error encoding AVCDecoderConfigurationRecord")
	}
}

func TestAvcDecoderConfigRecordWithExtraBytesProfileHigh(t *testing.T) {
	//nolint: lll
	byteData, _ := hex.DecodeString("01640029ffe1002667640029ac3ca5014016ec050808080a00000300020000030065c0c000b71a00022551f89f0501000468ef752500")
	spsBytes, _ := hex.DecodeString("67640029ac3ca5014016ec050808080a00000300020000030065c0c000b71a00022551f89f05")
	ppsBytes, _ := hex.DecodeString("68ef7525")

	wanted := DecConfRec{
		AVCProfileIndication: 100,
		ProfileCompatibility: 0,
		AVCLevelIndication:   41,
		SPSnalus:             [][]byte{spsBytes},
		PPSnalus:             [][]byte{ppsBytes},
		ChromaFormat:         0,
		BitDepthLumaMinus1:   0,
		BitDepthChromaMinus1: 0,
		NumSPSExt:            0,
		NoTrailingInfo:       true,
		SkipBytes:            1,
	}

	got, err := DecodeAVCDecConfRec(byteData)
	if err != nil {
		t.Error("Error parsing AVCDecoderConfigurationRecord")
	}
	if diff := deep.Equal(got, wanted); diff != nil {
		t.Error(diff)
	}

	enc := bytes.Buffer{}
	err = got.Encode(&enc)
	if err != nil {
		t.Error("Error encoding AVCDecoderConfigurationRecord")
	}
	if !bytes.Equal(enc.Bytes(), byteData) {
		t.Error("Error encoding AVCDecoderConfigurationRecord")
	}
}

func TestCreateAVCDecConfRec(t *testing.T) {
	data, err := os.ReadFile("testdata/blackframe.264")
	if err != nil {
		t.Error("Error reading file")
	}
	spss := ExtractNalusOfTypeFromByteStream(NALU_SPS, data, true)
	ppss := ExtractNalusOfTypeFromByteStream(NALU_PPS, data, true)
	if len(spss) != 1 || len(ppss) != 1 {
		t.Error("Error extracting SPS/PPS")
	}
	_, err = CreateAVCDecConfRec(spss, ppss, true)
	if err != nil {
		t.Error("Error creating AVCDecoderConfigurationRecord")
	}
}

func TestAvcDecoderConfigRecordProfile244(t *testing.T) {
	// Same record as in TestAvcDecoderConfigRecord, but with profile 244
	byteData, _ := hex.DecodeString("01f4001effe100196764001eacd940a02ff9610000030001000003003c8f162d9601000568ebecb22cfdf8f800")

	got, err := DecodeAVCDecConfRec(byteData)
	if err != nil {
		t.Error("Error parsing AVCDecoderConfigurationRecord")
	}
	if got.AVCProfileIndication != 244 {
		t.Errorf("got profile %d instead of 244", got.AVCProfileIndication)
	}
	if got.Size() != uint64(len(byteData)) {
		t.Errorf("Size() = %d, but the record is %d bytes", got.Size(), len(byteData))
	}

	enc := bytes.Buffer{}
	err = got.Encode(&enc)
	if err != nil {
		t.Error("Error encoding AVCDecoderConfigurationRecord")
	}
	if !bytes.Equal(enc.Bytes(), byteData) {
		t.Errorf("encoded record differs from input:\n got %s\nwant %s",
			hex.EncodeToString(enc.Bytes()), hex.EncodeToString(byteData))
	}
}

func TestAvcDecoderConfigRecordTrailingBytes(t *testing.T) {
	// Extra bytes after the trailing info as written by some muxers
	cases := []struct {
		name     string
		trailing string
	}{
		{"single zero byte", "00"},
		{"encoder options blob", hex.EncodeToString([]byte("x264 - options: cabac=1 ref=2"))},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			byteData, _ := hex.DecodeString(avcDecoderConfigRecord + c.trailing)
			trailing, _ := hex.DecodeString(c.trailing)

			got, err := DecodeAVCDecConfRec(byteData)
			if err != nil {
				t.Fatalf("Error parsing AVCDecoderConfigurationRecord: %v", err)
			}
			if !bytes.Equal(got.TrailingBytes, trailing) {
				t.Errorf("TrailingBytes = %x, want %x", got.TrailingBytes, trailing)
			}
			if got.NoTrailingInfo {
				t.Error("NoTrailingInfo set although trailing info was parsed")
			}
			if got.Size() != uint64(len(byteData)) {
				t.Errorf("Size() = %d, but the record is %d bytes", got.Size(), len(byteData))
			}

			enc := bytes.Buffer{}
			err = got.Encode(&enc)
			if err != nil {
				t.Fatalf("Error encoding AVCDecoderConfigurationRecord: %v", err)
			}
			if !bytes.Equal(enc.Bytes(), byteData) {
				t.Errorf("encoded record differs from input:\n got %s\nwant %s",
					hex.EncodeToString(enc.Bytes()), hex.EncodeToString(byteData))
			}
		})
	}
}

func TestAvcDecoderConfigRecordLengthSize(t *testing.T) {
	// Same record as in TestAvcDecoderConfigRecord, but with each lengthSizeMinusOne value
	cases := []struct {
		lengthSizeMinusOne byte
		naluLengthSize     byte
		wantErr            error
	}{
		{lengthSizeMinusOne: 0, naluLengthSize: 1},
		{lengthSizeMinusOne: 1, naluLengthSize: 2},
		{lengthSizeMinusOne: 2, wantErr: ErrLengthSize},
		{lengthSizeMinusOne: 3, naluLengthSize: 0},
	}
	for _, c := range cases {
		lengthSize := int(c.lengthSizeMinusOne) + 1
		t.Run(fmt.Sprintf("%d-byte", lengthSize), func(t *testing.T) {
			byteData, _ := hex.DecodeString(avcDecoderConfigRecord)
			byteData[4] = 0xfc | c.lengthSizeMinusOne

			got, err := DecodeAVCDecConfRec(byteData)
			if c.wantErr != nil {
				if !errors.Is(err, c.wantErr) {
					t.Errorf("got error %v, want %v", err, c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Error parsing AVCDecoderConfigurationRecord: %v", err)
			}
			if got.NaluLengthSize != c.naluLengthSize {
				t.Errorf("NaluLengthSize = %d, want %d", got.NaluLengthSize, c.naluLengthSize)
			}
			if got.LengthSize() != lengthSize {
				t.Errorf("LengthSize() = %d, want %d", got.LengthSize(), lengthSize)
			}

			enc := bytes.Buffer{}
			err = got.Encode(&enc)
			if err != nil {
				t.Fatalf("Error encoding AVCDecoderConfigurationRecord: %v", err)
			}
			if !bytes.Equal(enc.Bytes(), byteData) {
				t.Errorf("encoded record differs from input:\n got %s\nwant %s",
					hex.EncodeToString(enc.Bytes()), hex.EncodeToString(byteData))
			}
		})
	}
}

func TestAvcDecoderConfigRecordEncodeLengthSize(t *testing.T) {
	spsBytes, _ := hex.DecodeString(sps)
	ppsBytes, _ := hex.DecodeString(pps)
	created, err := CreateAVCDecConfRec([][]byte{spsBytes}, [][]byte{ppsBytes}, true)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name    string
		rec     DecConfRec
		wantErr error
	}{
		{name: "zero value", rec: DecConfRec{}},
		{name: "CreateAVCDecConfRec", rec: *created},
		{name: "explicit 4 bytes", rec: DecConfRec{NaluLengthSize: 4}},
		{name: "3 bytes", rec: DecConfRec{NaluLengthSize: 3}, wantErr: ErrLengthSize},
		{name: "5 bytes", rec: DecConfRec{NaluLengthSize: 5}, wantErr: ErrLengthSize},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			enc := bytes.Buffer{}
			err := c.rec.Encode(&enc)
			if c.wantErr != nil {
				if !errors.Is(err, c.wantErr) {
					t.Errorf("got error %v, want %v", err, c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Error encoding AVCDecoderConfigurationRecord: %v", err)
			}
			if got := enc.Bytes()[4]; got != 0xff {
				t.Errorf("got lengthSizeMinusOne byte %#02x, want 0xff (4-byte lengths)", got)
			}
		})
	}
}
