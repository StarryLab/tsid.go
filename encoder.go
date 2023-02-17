package tsid

import (
	"bytes"
	"errors"
	"fmt"
	"math/bits"
	"strconv"
	"strings"
)

type Encoder interface {
	Encode(id *ID) string
	Decode(no string) (id *ID, err error)
}

const (
	base64Digits   = "0xHqN63nKLpM1hJRwZ9jklm.Y4aPoIiQA2DrsVB5Ob7CzcFGdv8U-EefgWXtuSTy"
	base64Signed   = '!'
	base64Widths   = 11
	base64Paddings = "00000000000000000000"
)

type Base64 struct {
	Aligned bool
}

type DecodeError struct {
	No  string
	Err error
}

func (e *DecodeError) Error() string {
	return fmt.Sprintf(`tsid.Base64.Decode: parsing %s error, reason: %s`, strconv.Quote(e.No), e.Err)
}

func (e *DecodeError) Unwrap() error {
	return e.Err
}

func decodeError(num, reason string) *DecodeError {
	return &DecodeError{No: num, Err: errors.New(reason)}
}

func (e *Base64) Encode(id *ID) string {
	s := [2]struct {
		buf []byte // string buffers
		len int    // string length
		pad int    // padding count
	}{}
	// g is the capacity of the builder's underlying byte slice.
	g := 0
	if id.Ext > 0 {
		s[0].buf = formatBits(id.Ext)
		s[0].len = len(s[0].buf)
		if e.Aligned && base64Widths > s[0].len {
			s[0].pad = base64Widths - s[0].len
		}
		g += s[0].len + s[0].pad
	}
	if id.Main != 0 {
		s[1].buf = formatBits(id.Main)
		s[1].len = len(s[1].buf)
		if (id.Ext > 0 || e.Aligned) && base64Widths > s[1].len {
			s[1].pad = base64Widths - s[1].len
		}
		g += s[1].len + s[1].pad
	} else if id.Ext > 0 || e.Aligned {
		// zero with padding
		s[1].pad = base64Widths
		g += base64Widths
	} else {
		// zero without padding
		s[1].buf = []byte{base64Digits[0]}
		s[1].len = 1
		g += 1
	}
	if id.Signed && (id.Main > 0 || id.Ext > 0) {
		g += 1
	}
	// build string
	b := strings.Builder{}
	b.Grow(g)
	if id.Signed && (id.Main > 0 || id.Ext > 0) {
		b.WriteByte(base64Signed)
	}
	if s[0].pad > 0 {
		b.Write([]byte(base64Paddings)[:s[0].pad])
	}
	if s[0].len > 0 {
		b.Write(s[0].buf)
	}
	if s[1].pad > 0 {
		b.Write([]byte(base64Paddings)[:s[1].pad])
	}
	if s[1].len > 0 {
		b.Write(s[1].buf)
	}
	return b.String()
}

func (e *Base64) Decode(no string) (id *ID, err error) {
	w := len(no)
	if w < 1 {
		return nil, decodeError(no, "number cannot empty")
	}
	i := 0
	s := no[0] == base64Signed
	if s {
		i++
		w--
	}
	if w < 1 {
		return nil, decodeError(no, "invalid base64 number")
	}
	var m, x string
	if w > base64Widths {
		m = no[w+i-base64Widths:]
		// extension part
		x = no[i : w+i-base64Widths]
	} else if s {
		m = no[1:]
	} else {
		m = no
	}
	var main, ext int64
	main, err = parseBits(m)
	if err != nil {
		return nil, err
	}
	if len(x) > 0 {
		ext, err = parseBits(x)
		if err != nil {
			return nil, err
		}
	}
	id = &ID{
		Main:   main,
		Ext:    ext,
		Signed: s,
	}
	return id, nil
}

// formatBits computes the string representation of u.
// If neg is set, u is treated as negative int64 value.
// From: `$GOROOT/src/strconv/itoa.go`
// [https://cs.opensource.google/go/go/+/refs/tags/go1.16:src/strconv/itoa.go]
func formatBits(u int64) []byte {
	var a [64]byte
	i := len(a)
	s := u < 0
	v := uint64(u)
	if s {
		v = -v
	}
	shift := uint(bits.TrailingZeros(uint(64))) & 7
	m := uint(64) - 1 // == 1<<shift - 1
	for v >= 64 {
		i--
		a[i] = base64Digits[uint(v)&m]
		v >>= shift
	}
	// u < base
	i--
	a[i] = base64Digits[uint(v)]
	return a[i:]
}

// From: `$GOROOT/src/strconv/atoi.go`
// [https://cs.opensource.google/go/go/+/refs/tags/go1.16:src/strconv/atoi.go]
func parseBits(s string) (v int64, err error) {
	if s == "" {
		return 0, decodeError(s, "number cannot empty")
	}
	b := 64
	maxVal := uint64(1)<<uint(b) - 1
	var n uint64
	for _, c := range []byte(s) {
		d := bytes.IndexByte([]byte(base64Digits), c)
		if d < 0 {
			return 0, decodeError(s, "invalid digit")
		}
		if n >= cutoff {
			// n*base overflows
			n = maxVal
			err = decodeError(s, "number overflows")
			break
		}
		n *= 64
		n1 := n + uint64(d)
		if n1 < n || n1 > maxVal {
			// n+d overflows
			n = maxVal
			err = decodeError(s, "number overflows")
			break
		}
		n = n1
	}

	co := uint64(1 << uint(63))
	if n >= co {
		return int64(co - 1), decodeError(s, "value out of range")
	}

	return int64(n), nil
}
