/*
Package avc parses AVC (H.264) NAL unit headers, slice headers and complete SPS and PPS.

The ...WithLengthSize functions take the size in bytes of the NALU length fields in the sample
(1, 2, or 4) as returned by DecConfRec.LengthSize(), not the raw NaluLengthSize field, which is 0
for 4 bytes. Any other value gives an empty result, or an error from GetNalusFromSampleWithLengthSize.
*/
package avc
