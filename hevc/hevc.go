package hevc

import (
	"fmt"

	"github.com/Eyevinn/mp4ff/internal/naluscan"
)

// NaluType - HEVC nal type according to ISO/IEC 23008-2 Table 7.1
type NaluType uint16

// HEVC NALU types
const (
	NALU_TRAIL_N = NaluType(0)
	NALU_TRAIL_R = NaluType(1)
	NALU_TSA_N   = NaluType(2)
	NALU_TSA_R   = NaluType(3)
	NALU_STSA_N  = NaluType(4)
	NALU_STSA_R  = NaluType(5)
	NALU_RADL_N  = NaluType(6)
	NALU_RADL_R  = NaluType(7)
	NALU_RASL_N  = NaluType(8)
	NALU_RASL_R  = NaluType(9)
	// BLA_W_LP and the following types are Random Access
	NALU_BLA_W_LP   = NaluType(16)
	NALU_BLA_W_RADL = NaluType(17)
	NALU_BLA_N_LP   = NaluType(18)
	NALU_IDR_W_RADL = NaluType(19)
	NALU_IDR_N_LP   = NaluType(20)
	NALU_CRA        = NaluType(21)
	// Reserved IRAP VCL NAL unit types
	NALU_IRAP_VCL22 = NaluType(22)
	NALU_IRAP_VCL23 = NaluType(23)
	// NALU_VPS - VideoParameterSet NAL Unit
	NALU_VPS = NaluType(32)
	// NALU_SPS - SequenceParameterSet NAL Unit
	NALU_SPS = NaluType(33)
	// NALU_PPS - PictureParameterSet NAL Unit
	NALU_PPS = NaluType(34)
	// NALU_AUD - AccessUnitDelimiter NAL Unit
	NALU_AUD = NaluType(35)
	//NALU_EOS - End of Sequence NAL Unit
	NALU_EOS = NaluType(36)
	//NALU_EOB - End of Bitstream NAL Unit
	NALU_EOB = NaluType(37)
	//NALU_FD - Filler data NAL Unit
	NALU_FD = NaluType(38)
	//NALU_SEI_PREFIX - Prefix SEI NAL Unit
	NALU_SEI_PREFIX = NaluType(39)
	//NALU_SEI_SUFFIX - Suffix SEI NAL Unit
	NALU_SEI_SUFFIX = NaluType(40)

	highestVideoNaluType = 31
)

func (n NaluType) String() string {
	switch n {
	case NALU_TRAIL_N, NALU_TRAIL_R:
		return fmt.Sprintf("NonRAP_Trail_%d", n)
	case NALU_TSA_N, NALU_TSA_R:
		return fmt.Sprintf("NonRAP_TSA_%d", n)
	case NALU_STSA_N, NALU_STSA_R:
		return fmt.Sprintf("NonRAP_STSA_%d", n)
	case NALU_RADL_N, NALU_RADL_R:
		return fmt.Sprintf("NonRAP_RADL_%d", n)
	case NALU_RASL_N, NALU_RASL_R:
		return fmt.Sprintf("NonRAP_RASL_%d", n)
	case NALU_BLA_N_LP, NALU_BLA_W_LP, NALU_BLA_W_RADL:
		return fmt.Sprintf("RAP_BLA_%d", n)
	case NALU_IDR_N_LP, NALU_IDR_W_RADL:
		return fmt.Sprintf("RAP_IDR_%d", n)
	case NALU_CRA:
		return fmt.Sprintf("RAP_CRA_%d", n)
	case NALU_VPS:
		return fmt.Sprintf("VPS_%d", n)
	case NALU_SPS:
		return fmt.Sprintf("SPS_%d", n)
	case NALU_PPS:
		return fmt.Sprintf("PPS_%d", n)
	case NALU_AUD:
		return fmt.Sprintf("AUD_%d", n)
	case NALU_SEI_PREFIX, NALU_SEI_SUFFIX:
		return fmt.Sprintf("SEI_%d", n)
	default:
		return fmt.Sprintf("Other_%d", n)
	}
}

// GetNaluType - extract NALU type from first byte of NALU Header
func GetNaluType(naluHeaderStart byte) NaluType {
	return NaluType((naluHeaderStart >> 1) & 0x3f)
}

// GetNaluLayerID extracts nuh_layer_id (6 bits) from the 2-byte HEVC NAL unit header.
// HEVC NAL header: forbidden(1) | nal_unit_type(6) | nuh_layer_id(6) | nuh_temporal_id_plus1(3)
func GetNaluLayerID(naluHeader []byte) byte {
	return ((naluHeader[0] & 0x01) << 5) | ((naluHeader[1] >> 3) & 0x1f)
}

// GetNaluTemporalID extracts nuh_temporal_id (nuh_temporal_id_plus1 - 1) from the 2-byte HEVC NAL unit header.
func GetNaluTemporalID(naluHeader []byte) byte {
	return (naluHeader[1] & 0x07) - 1
}

// NaluInfo holds parsed information from a HEVC NAL unit header.
type NaluInfo struct {
	Type       NaluType
	LayerID    byte
	TemporalID byte
}

// ParseNaluHeader parses a 2-byte HEVC NAL unit header.
func ParseNaluHeader(naluHeader []byte) NaluInfo {
	return NaluInfo{
		Type:       GetNaluType(naluHeader[0]),
		LayerID:    GetNaluLayerID(naluHeader),
		TemporalID: GetNaluTemporalID(naluHeader),
	}
}

// SplitNalusByLayerID splits length-prefixed NALUs in a sample by nuh_layer_id.
// The lengthSize is the length field size in bytes, as returned by DecConfRec.LengthSize().
func SplitNalusByLayerID(sample []byte, lengthSize int) map[byte][][]byte {
	result := make(map[byte][][]byte)
	_ = naluscan.Walk(sample, lengthSize, 2, func(nalu []byte) bool {
		layerID := GetNaluLayerID(nalu)
		result[layerID] = append(result[layerID], nalu)
		return true
	})
	return result
}

// FindNaluTypes - find list of nalu types in sample with 4-byte NALU lengths
func FindNaluTypes(sample []byte) []NaluType {
	return FindNaluTypesWithLengthSize(sample, 4)
}

// FindNaluTypesWithLengthSize - find list of nalu types in sample with lengthSize-byte NALU lengths
func FindNaluTypesWithLengthSize(sample []byte, lengthSize int) []NaluType {
	naluList := make([]NaluType, 0)
	_ = naluscan.Walk(sample, lengthSize, 1, func(nalu []byte) bool {
		naluList = append(naluList, GetNaluType(nalu[0]))
		return true
	})
	return naluList
}

// FindNaluTypesUpToFirstVideoNalu - all nalu types up to first video nalu in sample with 4-byte NALU lengths
func FindNaluTypesUpToFirstVideoNalu(sample []byte) []NaluType {
	return FindNaluTypesUpToFirstVideoNaluWithLengthSize(sample, 4)
}

// FindNaluTypesUpToFirstVideoNaluWithLengthSize - all nalu types up to first video nalu in sample with lengthSize-byte NALU lengths
func FindNaluTypesUpToFirstVideoNaluWithLengthSize(sample []byte, lengthSize int) []NaluType {
	naluList := make([]NaluType, 0)
	_ = naluscan.Walk(sample, lengthSize, 1, func(nalu []byte) bool {
		naluType := GetNaluType(nalu[0])
		naluList = append(naluList, naluType)
		return !IsVideoNaluType(naluType)
	})
	return naluList
}

// IsVideoNaluType returns true if NaluType is a video type (<= 31)
func IsVideoNaluType(naluType NaluType) bool {
	return naluType <= highestVideoNaluType
}

// ContainsNaluType - is specific NaluType present in sample with 4-byte NALU lengths
func ContainsNaluType(sample []byte, specificNaluType NaluType) bool {
	return ContainsNaluTypeWithLengthSize(sample, specificNaluType, 4)
}

// ContainsNaluTypeWithLengthSize - is specific NaluType present in sample with lengthSize-byte NALU lengths
func ContainsNaluTypeWithLengthSize(sample []byte, specificNaluType NaluType, lengthSize int) bool {
	found := false
	_ = naluscan.Walk(sample, lengthSize, 1, func(nalu []byte) bool {
		found = GetNaluType(nalu[0]) == specificNaluType
		return !found
	})
	return found
}

// IsRAPSample - is Random Access picture (NALU 16-23) in sample with 4-byte NALU lengths
func IsRAPSample(sample []byte) bool {
	return IsRAPSampleWithLengthSize(sample, 4)
}

// IsRAPSampleWithLengthSize - is Random Access picture (NALU 16-23) in sample with lengthSize-byte NALU lengths
func IsRAPSampleWithLengthSize(sample []byte, lengthSize int) bool {
	for _, naluType := range FindNaluTypesWithLengthSize(sample, lengthSize) {
		if 16 <= naluType && naluType <= 23 {
			return true
		}
	}
	return false
}

// IsIDRSample - is IDR picture (NALU 19-20) in sample with 4-byte NALU lengths
func IsIDRSample(sample []byte) bool {
	return IsIDRSampleWithLengthSize(sample, 4)
}

// IsIDRSampleWithLengthSize - is IDR picture (NALU 19-20) in sample with lengthSize-byte NALU lengths
func IsIDRSampleWithLengthSize(sample []byte, lengthSize int) bool {
	for _, naluType := range FindNaluTypesWithLengthSize(sample, lengthSize) {
		if 19 <= naluType && naluType <= 20 {
			return true
		}
	}
	return false
}

// HasParameterSets - Check if HEVC VPS, SPS and PPS are present in sample with 4-byte NALU lengths
func HasParameterSets(b []byte) bool {
	return HasParameterSetsWithLengthSize(b, 4)
}

// HasParameterSetsWithLengthSize - Check if HEVC VPS, SPS and PPS are present in sample with lengthSize-byte NALU lengths
func HasParameterSetsWithLengthSize(b []byte, lengthSize int) bool {
	naluTypeList := FindNaluTypesUpToFirstVideoNaluWithLengthSize(b, lengthSize)
	var hasVPS, hasSPS, hasPPS bool
	for _, naluType := range naluTypeList {
		switch naluType {
		case NALU_VPS:
			hasVPS = true
		case NALU_SPS:
			hasSPS = true
		case NALU_PPS:
			hasPPS = true
		}
		if hasVPS && hasSPS && hasPPS {
			return true
		}
	}
	return false
}

// GetParameterSets - get (multiple) VPS,  SPS, and PPS from a sample with 4-byte NALU lengths
func GetParameterSets(sample []byte) (vps, sps, pps [][]byte) {
	return GetParameterSetsWithLengthSize(sample, 4)
}

// GetParameterSetsWithLengthSize - get (multiple) VPS,  SPS, and PPS from a sample with lengthSize-byte NALU lengths
func GetParameterSetsWithLengthSize(sample []byte, lengthSize int) (vps, sps, pps [][]byte) {
	_ = naluscan.Walk(sample, lengthSize, 1, func(nalu []byte) bool {
		switch naluType := GetNaluType(nalu[0]); {
		case naluType == NALU_VPS:
			vps = append(vps, nalu)
		case naluType == NALU_SPS:
			sps = append(sps, nalu)
		case naluType == NALU_PPS:
			pps = append(pps, nalu)
		case naluType <= highestVideoNaluType:
			return false
		}
		return true
	})
	return vps, sps, pps
}
