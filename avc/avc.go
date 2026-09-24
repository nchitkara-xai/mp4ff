package avc

import (
	"fmt"

	"github.com/Eyevinn/mp4ff/internal/naluscan"
)

// NaluType - AVC NAL unit type
type NaluType uint16

const (
	// NALU_NON_IDR - Non-IDR Slice NAL unit
	NALU_NON_IDR = NaluType(1)
	// NALU_IDR - IDR Random Access Slice NAL Unit
	NALU_IDR = NaluType(5)
	// NALU_SEI - Supplementary Enhancement Information NAL Unit
	NALU_SEI = NaluType(6)
	// NALU_SPS - SequenceParameterSet NAL Unit
	NALU_SPS = NaluType(7)
	// NALU_PPS - PictureParameterSet NAL Unit
	NALU_PPS = NaluType(8)
	// NALU_AUD - AccessUnitDelimiter NAL Unit
	NALU_AUD = NaluType(9)
	// NALU_EO_SEQ - End of Sequence NAL Unit
	NALU_EO_SEQ = NaluType(10)
	// NALU_EO_STREAM - End of Stream NAL Unit
	NALU_EO_STREAM = NaluType(11)
	// NALU_FILL - Filler NAL Unit
	NALU_FILL = NaluType(12)
)

func (a NaluType) String() string {
	switch a {
	case NALU_NON_IDR:
		return "NonIDR_1"
	case NALU_IDR:
		return "IDR_5"
	case NALU_SEI:
		return "SEI_6"
	case NALU_SPS:
		return "SPS_7"
	case NALU_PPS:
		return "PPS_8"
	case NALU_AUD:
		return "AUD_9"
	case NALU_EO_SEQ:
		return "EndOfSequence_10"
	case NALU_EO_STREAM:
		return "EndOfStream_11"
	case NALU_FILL:
		return "FILL_12"
	default:
		return fmt.Sprintf("Other_%d", a)
	}
}

// GetNaluType - get NALU type from  NALU Header byte
func GetNaluType(naluHeader byte) NaluType {
	return NaluType(naluHeader & 0x1f)
}

// FindNaluTypes - find list of NAL unit types in sample with 4-byte NALU lengths
func FindNaluTypes(sample []byte) []NaluType {
	return FindNaluTypesWithLengthSize(sample, 4)
}

// FindNaluTypesWithLengthSize - find list of NAL unit types in sample with lengthSize-byte NALU lengths
func FindNaluTypesWithLengthSize(sample []byte, lengthSize int) []NaluType {
	if !naluscan.ValidLengthSize(lengthSize) || len(sample) < lengthSize {
		return nil
	}
	naluList := make([]NaluType, 0, 2)
	_ = naluscan.Walk(sample, lengthSize, 1, func(nalu []byte) bool {
		naluList = append(naluList, GetNaluType(nalu[0]))
		return true
	})
	return naluList
}

// FindNaluTypesUpToFirstVideoNALU - find list of NAL unit types in sample with 4-byte NALU lengths
func FindNaluTypesUpToFirstVideoNALU(sample []byte) []NaluType {
	return FindNaluTypesUpToFirstVideoNALUWithLengthSize(sample, 4)
}

// FindNaluTypesUpToFirstVideoNALUWithLengthSize - find list of NAL unit types in sample with lengthSize-byte NALU lengths
func FindNaluTypesUpToFirstVideoNALUWithLengthSize(sample []byte, lengthSize int) []NaluType {
	if !naluscan.ValidLengthSize(lengthSize) || len(sample) < lengthSize {
		return nil
	}
	naluList := make([]NaluType, 0)
	_ = naluscan.Walk(sample, lengthSize, 1, func(nalu []byte) bool {
		naluType := GetNaluType(nalu[0])
		naluList = append(naluList, naluType)
		return !IsVideoNaluType(naluType)
	})
	return naluList
}

// IsIDRSample - does sample with 4-byte NALU lengths contain IDR NALU
func IsIDRSample(sample []byte) bool {
	return IsIDRSampleWithLengthSize(sample, 4)
}

// IsIDRSampleWithLengthSize - does sample with lengthSize-byte NALU lengths contain IDR NALU
func IsIDRSampleWithLengthSize(sample []byte, lengthSize int) bool {
	return ContainsNaluTypeWithLengthSize(sample, NALU_IDR, lengthSize)
}

// ContainsNaluType - is specific NaluType present in sample with 4-byte NALU lengths
func ContainsNaluType(sample []byte, specificNalType NaluType) bool {
	return ContainsNaluTypeWithLengthSize(sample, specificNalType, 4)
}

// ContainsNaluTypeWithLengthSize - is specific NaluType present in sample with lengthSize-byte NALU lengths
func ContainsNaluTypeWithLengthSize(sample []byte, specificNalType NaluType, lengthSize int) bool {
	found := false
	_ = naluscan.Walk(sample, lengthSize, 1, func(nalu []byte) bool {
		found = GetNaluType(nalu[0]) == specificNalType
		return !found
	})
	return found
}

// HasParameterSets - Check if H.264 SPS and PPS are present in sample with 4-byte NALU lengths
func HasParameterSets(b []byte) bool {
	return HasParameterSetsWithLengthSize(b, 4)
}

// HasParameterSetsWithLengthSize - Check if H.264 SPS and PPS are present in sample with lengthSize-byte NALU lengths
func HasParameterSetsWithLengthSize(b []byte, lengthSize int) bool {
	naluTypeList := FindNaluTypesUpToFirstVideoNALUWithLengthSize(b, lengthSize)
	hasSPS := false
	hasPPS := false
	for _, naluType := range naluTypeList {
		if naluType == NALU_SPS {
			hasSPS = true
		}
		if naluType == NALU_PPS {
			hasPPS = true
		}
		if hasSPS && hasPPS {
			return true
		}
	}
	return false
}

// GetParameterSets - get (multiple) SPS and PPS from a sample with 4-byte NALU lengths
func GetParameterSets(sample []byte) (sps [][]byte, pps [][]byte) {
	return GetParameterSetsWithLengthSize(sample, 4)
}

// GetParameterSetsWithLengthSize - get (multiple) SPS and PPS from a sample with lengthSize-byte NALU lengths
func GetParameterSetsWithLengthSize(sample []byte, lengthSize int) (sps [][]byte, pps [][]byte) {
	_ = naluscan.Walk(sample, lengthSize, 1, func(nalu []byte) bool {
		switch naluType := GetNaluType(nalu[0]); {
		case naluType == NALU_SPS:
			sps = append(sps, nalu)
		case naluType == NALU_PPS:
			pps = append(pps, nalu)
		case IsVideoNaluType(naluType):
			return false //SPS and PPS must come before video
		}
		return true
	})
	return sps, pps
}

// IsVideoNaluType returns true if nalu type is a VCL nalu.
func IsVideoNaluType(naluType NaluType) bool {
	const highestVideoNaluType = 5
	return naluType <= highestVideoNaluType
}
