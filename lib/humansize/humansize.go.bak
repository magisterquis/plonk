// Package humansize - Human-readable data sizes
package humansize

/*
 * humansize.go
 * Human-readable data sizes
 * By J. Stuart McMurray
 * Created 20231210
 * Last Modified 20231210
 */

import (
	"fmt"
	"strconv"
	"unicode"
)

// suffixes holds the SI prefixes (as suffixes) and their bit shift distances.
var suffixes = []struct {
	suffix rune
	bits   uint64
}{
	{suffix: 'E', bits: 60},
	{suffix: 'P', bits: 50},
	{suffix: 'T', bits: 40},
	{suffix: 'G', bits: 30},
	{suffix: 'M', bits: 20},
	{suffix: 'K', bits: 10},
}

// Size allows us to give users the option to use suffixes like g and K.  It
// is also suitable for passing to flag.TextVar.
type Size uint64

// MustNew retruns a new Size into which is unmarshalled s.  If Unmarshalling
// fails, MustNew panics.
func MustNew(s string) Size {
	var sz Size
	if err := sz.UnmarshalText([]byte(s)); nil != err {
		panic(err)
	}
	return sz
}

// UnmarshalText implements encoding.TextUnmarshaler.  A nil or empty slices
// unmarshals to 0.
func (s *Size) UnmarshalText(text []byte) error {
	ns := []rune(string(text))

	/* If we have nothing, we have 0. */
	if 0 == len(ns) {
		*s = 0
		return nil
	}

	/* Work out our suffix. */
	/* Suffixes mooched from https://en.wikipedia.org/wiki/Binary_prefix */
	var shift uint64
	sc := unicode.ToUpper(ns[len(ns)-1])
	for _, v := range suffixes {
		if sc == v.suffix {
			shift = v.bits
			break
		}
	}

	/* What are we shifting? */
	if 0 != shift {
		ns = ns[:len(ns)-1]
	}
	n, err := strconv.ParseUint(string(ns), 0, 64)
	if nil != err {
		return err
	}

	/* Make a beeeeeeg number. */
	*s = Size(n << shift)

	return nil
}

// MarshalText implements encoding.TextMarshaler.  It is just s.String with
// extra steps.
func (s *Size) MarshalText() (text []byte, err error) {
	return []byte(s.String()), nil
}

// String implements fmt.Stringer and is more or less the inverse of
// UnmarshalText.
func (s Size) String() string {
	/* 0 is always easy. */
	if 0 == s {
		return "0"
	}

	/* Work out how much we can get away with shifting. */
	n := uint64(s)
	for _, v := range suffixes {
		if 0 == n&^(0xFFFFFFFFFFFFFFFF<<v.bits) { /* Fits :) */
			return fmt.Sprintf(
				"%d%c",
				n>>v.bits,
				v.suffix,
			)
		}
	}

	return fmt.Sprintf("%d", s)
}
