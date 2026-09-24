/*
Package hevc -  parsing of HEVC(H.265) NAL unit headers, slice headers, VPS, SPS, and PPS.

The ...WithLengthSize functions take the size in bytes of the NALU length fields in the sample
(1, 2, or 4) as returned by DecConfRec.LengthSize(), not the raw LengthSizeMinusOne field.
Any other value gives an empty result.
*/
package hevc
