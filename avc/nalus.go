package avc

import (
	"fmt"

	"github.com/Eyevinn/mp4ff/internal/naluscan"
)

// GetNalusFromSample - get nalus by following 4 byte length fields
func GetNalusFromSample(sample []byte) ([][]byte, error) {
	return GetNalusFromSampleWithLengthSize(sample, 4)
}

// GetNalusFromSampleWithLengthSize - get nalus by following lengthSize-byte length fields
// This function is codec agnostic.
func GetNalusFromSampleWithLengthSize(sample []byte, lengthSize int) ([][]byte, error) {
	if !naluscan.ValidLengthSize(lengthSize) {
		return nil, fmt.Errorf("%w, not %d", ErrLengthSize, lengthSize)
	}
	if len(sample) < lengthSize {
		return nil, fmt.Errorf("less than %d bytes, No NALUs", lengthSize)
	}
	naluList := make([][]byte, 0, 2)
	err := naluscan.Walk(sample, lengthSize, 0, func(nalu []byte) bool {
		naluList = append(naluList, nalu)
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("NALU length fields are bad. Not video?")
	}
	return naluList, nil
}
