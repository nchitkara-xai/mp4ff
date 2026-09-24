// Package naluscan walks the NAL units of a sample in which each NAL unit is
// preceded by a big-endian length field.
package naluscan

import (
	"encoding/binary"
	"errors"
)

var (
	ErrLengthSize = errors.New("NALU length size must be 1, 2, or 4 bytes")
	ErrBadLength  = errors.New("bad NALU length field")
)

// Walk calls visit with each NALU in sample, in order, where each NALU is
// preceded by a big-endian lengthSize-byte (1, 2, or 4) length field. It stops
// when visit returns false or when no more than lengthSize bytes remain.
// It returns ErrLengthSize for any other lengthSize, and ErrBadLength at the
// first length field below minLen or reaching past the end of sample.
func Walk(sample []byte, lengthSize, minLen int, visit func(nalu []byte) bool) error {
	if !ValidLengthSize(lengthSize) {
		return ErrLengthSize
	}
	pos := 0
	for pos+lengthSize < len(sample) {
		var naluLength uint64
		switch lengthSize {
		case 1:
			naluLength = uint64(sample[pos])
		case 2:
			naluLength = uint64(binary.BigEndian.Uint16(sample[pos:]))
		case 4:
			naluLength = uint64(binary.BigEndian.Uint32(sample[pos:]))
		}
		pos += lengthSize
		if naluLength < uint64(minLen) || uint64(pos)+naluLength > uint64(len(sample)) {
			return ErrBadLength
		}
		end := pos + int(naluLength)
		nalu := sample[pos:end]
		pos = end
		if !visit(nalu) {
			return nil
		}
	}
	return nil
}

// ValidLengthSize reports whether lengthSize is 1, 2, or 4.
func ValidLengthSize(lengthSize int) bool {
	return lengthSize == 1 || lengthSize == 2 || lengthSize == 4
}
