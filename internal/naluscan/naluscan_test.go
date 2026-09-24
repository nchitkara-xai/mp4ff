package naluscan

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/go-test/deep"
)

func field(t *testing.T, lengthSize, n int) []byte {
	t.Helper()
	if n >= 1<<(8*lengthSize) {
		t.Fatalf("%d does not fit in a %d-byte length field", n, lengthSize)
	}
	b := make([]byte, lengthSize)
	for i := lengthSize - 1; i >= 0; i-- {
		b[i] = byte(n)
		n >>= 8
	}
	return b
}

func frame(t *testing.T, lengthSize int, nalus ...[]byte) []byte {
	t.Helper()
	var sample []byte
	for _, nalu := range nalus {
		sample = append(sample, field(t, lengthSize, len(nalu))...)
		sample = append(sample, nalu...)
	}
	return sample
}

func TestWalk(t *testing.T) {
	a, b := []byte{0x67, 0x42}, []byte{0x68}
	long := make([]byte, 300)
	cases := []struct {
		desc        string
		lengthSizes []int // 1, 2, and 4 if nil
		sample      func(t *testing.T, ls int) []byte
		minLen      int
		maxVisit    int // stop after this many visits, 0 for no limit
		want        [][]byte
		wantErr     error
	}{
		{desc: "empty", sample: func(t *testing.T, ls int) []byte { return nil }},
		{desc: "length field only", sample: func(t *testing.T, ls int) []byte { return field(t, ls, 1) }},
		{desc: "two nalus", sample: func(t *testing.T, ls int) []byte { return frame(t, ls, a, b) }, want: [][]byte{a, b}},
		{desc: "stopped by visit", sample: func(t *testing.T, ls int) []byte { return frame(t, ls, a, b) }, maxVisit: 1, want: [][]byte{a}},
		{desc: "trailing partial length field",
			sample: func(t *testing.T, ls int) []byte { return append(frame(t, ls, a), make([]byte, ls-1)...) },
			want:   [][]byte{a}},
		{desc: "zero length nalu allowed", sample: func(t *testing.T, ls int) []byte { return frame(t, ls, nil, a) },
			want: [][]byte{{}, a}},
		{desc: "zero length nalu below minLen", sample: func(t *testing.T, ls int) []byte { return frame(t, ls, nil, a) }, minLen: 1,
			wantErr: ErrBadLength},
		{desc: "nalu below minLen", sample: func(t *testing.T, ls int) []byte { return frame(t, ls, a, b) }, minLen: 2, want: [][]byte{a},
			wantErr: ErrBadLength},
		{desc: "length beyond sample",
			sample: func(t *testing.T, ls int) []byte { return append(frame(t, ls, a), append(field(t, ls, 3), b...)...) },
			want:   [][]byte{a}, wantErr: ErrBadLength},
		{desc: "trailing length field ignored",
			sample: func(t *testing.T, ls int) []byte { return append(frame(t, ls, a), field(t, ls, 3)...) },
			want:   [][]byte{a}},
		{desc: "long nalu", lengthSizes: []int{2, 4}, sample: func(t *testing.T, ls int) []byte { return frame(t, ls, a, long, b) },
			want: [][]byte{a, long, b}},
		{desc: "length beyond sample in the upper byte", lengthSizes: []int{2, 4},
			sample: func(t *testing.T, ls int) []byte {
				return append(field(t, ls, map[int]int{2: 0x0103, 4: 0x00010003}[ls]), a[0], a[1], b[0])
			}, wantErr: ErrBadLength},
		{desc: "maximum length field", sample: func(t *testing.T, ls int) []byte { return append(bytes.Repeat([]byte{0xff}, ls), a...) },
			wantErr: ErrBadLength},
	}
	for _, c := range cases {
		lengthSizes := c.lengthSizes
		if lengthSizes == nil {
			lengthSizes = []int{1, 2, 4}
		}
		for _, ls := range lengthSizes {
			t.Run(fmt.Sprintf("%s %d-byte", c.desc, ls), func(t *testing.T) {
				var got [][]byte
				err := Walk(c.sample(t, ls), ls, c.minLen, func(nalu []byte) bool {
					got = append(got, nalu)
					return c.maxVisit == 0 || len(got) < c.maxVisit
				})
				if !errors.Is(err, c.wantErr) {
					t.Errorf("got error %v, want %v", err, c.wantErr)
				}
				if diff := deep.Equal(got, c.want); diff != nil {
					t.Error(diff)
				}
			})
		}
	}
}

func TestWalkLengthSize(t *testing.T) {
	for _, ls := range []int{-1, 0, 3, 5, 8} {
		visited := false
		err := Walk(frame(t, 4, []byte{0x67}), ls, 0, func(nalu []byte) bool {
			visited = true
			return true
		})
		if !errors.Is(err, ErrLengthSize) || visited {
			t.Errorf("length size %d: got error %v and visited %t, want %v and no visit", ls, err, visited, ErrLengthSize)
		}
	}
}
