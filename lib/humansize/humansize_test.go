package humansize

/*
 * humansize_test.go
 * Tests for humansize.go
 * By J. Stuart McMurray
 * Created 20231210
 * Last Modified 20231210
 */

import (
	"strconv"
	"testing"
)

func TestMustNew(t *testing.T) {
	for _, c := range []struct {
		have string
		want uint64
	}{
		{"", 0},
		{"0", 0},
		{"1", 1},
		{"100", 100},
		{"1K", 1024},
		{"1K", 1024},
		{"2K", 2048},
		{"3K", 3072},
		{"65535", 65535},
		{"65K", 66560},
		{"1G", 1024 * 1024 * 1024},
		{"1234T", 1234 * 1024 * 1024 * 1024 * 1024},
	} {
		c := c /* :( */
		t.Run(c.have, func(t *testing.T) {
			if got := uint64(MustNew(c.have)); got != c.want {
				t.Fatalf(
					"Incorrect value:\n"+
						"have: %s\n"+
						" got: %d\n"+
						"want: %d",
					c.have,
					got,
					c.want,
				)
			}
		})
	}
}

func TestSizeUnmarshalText(t *testing.T) {
	for _, c := range []struct {
		have string
		want uint64
	}{
		{"", 0},
		{"0", 0},
		{"1", 1},
		{"100", 100},
		{"1K", 1024},
		{"1K", 1024},
		{"2K", 2048},
		{"3K", 3072},
		{"65535", 65535},
		{"65K", 66560},
		{"1G", 1024 * 1024 * 1024},
		{"1234T", 1234 * 1024 * 1024 * 1024 * 1024},
	} {
		c := c /* :( */
		t.Run(c.have, func(t *testing.T) {
			var s Size
			if err := s.UnmarshalText([]byte(c.have)); nil != err {
				t.Fatalf("Error: %s", err)
			}
			if got := uint64(s); got != c.want {
				t.Fatalf(
					"Incorrect parse:\n"+
						"have: %s\n"+
						" got: %d\n"+
						"want: %d",
					c.have,
					got,
					c.want,
				)
			}
		})
	}
}

func TestSizeMarshalText(t *testing.T) {
	for _, c := range []struct {
		have uint64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{100, "100"},
		{1024, "1K"},
		{1025, "1025"},
		{2048, "2K"},
		{3072, "3K"},
		{65535, "65535"},
		{66560, "65K"},
		{134221824, "131076K"},
		{23 * 1024 * 1024, "23M"},
		{1024 * 1024 * 1024, "1G"},
		{1234 * 1024 * 1024 * 1024 * 1024, "1234T"},
	} {
		c := c /* :( */
		t.Run(strconv.FormatUint(c.have, 10), func(t *testing.T) {
			s := Size(c.have)
			b, err := s.MarshalText()
			if nil != err {
				t.Fatalf("Error: %s", err)
			}
			if got := string(b); got != c.want {
				t.Fatalf(
					"Incorrect MarshalText result:\n"+
						"have: %d\n"+
						" got: %s\n"+
						"want: %s",
					c.have,
					got,
					c.want,
				)
			}
		})
	}
}

func TestSizeString(t *testing.T) {
	for _, c := range []struct {
		have uint64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{100, "100"},
		{1024, "1K"},
		{1025, "1025"},
		{2048, "2K"},
		{3072, "3K"},
		{65535, "65535"},
		{66560, "65K"},
		{134221824, "131076K"},
		{23 * 1024 * 1024, "23M"},
		{1024 * 1024 * 1024, "1G"},
		{1234 * 1024 * 1024 * 1024 * 1024, "1234T"},
	} {
		c := c /* :( */
		t.Run(strconv.FormatUint(c.have, 10), func(t *testing.T) {
			s := Size(c.have)
			if got := s.String(); got != c.want {
				t.Fatalf(
					"Incorrect MarshalText result:\n"+
						"have: %d\n"+
						" got: %s\n"+
						"want: %s",
					c.have,
					got,
					c.want,
				)
			}
		})
	}
}
