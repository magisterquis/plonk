package opshell

import "fmt"

/*
 * colors.go
 * Color text
 * By J. Stuart McMurray
 * Created 20231130
 * Last Modified 20240119
 */

// Color is used ot select a color from terminal.EscapeCodes.  Please see
// term.EscapeCodes for more information.
type Color string

const (
	ColorBlack   Color = "black"
	ColorRed     Color = "red"
	ColorGreen   Color = "green"
	ColorYellow  Color = "yellow"
	ColorBlue    Color = "blue"
	ColorMagenta Color = "magenta"
	ColorCyan    Color = "cyan"
	ColorWhite   Color = "white"
	ColorReset   Color = "reset"
)

// Color wraps t with escape codes to set the color and then reset it after s.
// if c isn't one of the Color* constants, t is returned.
func (s *Shell[T]) Color(c Color, t string) string {
	cc := s.ColorCode(c)
	if "" == cc {
		return t
	}
	return fmt.Sprintf("%s%s%s", cc, t, s.ColorCode(ColorReset))
}

// ColorCode returns a terminal sequence for the given color.  Please see
// term.EscapeCodes for more information.  This is a wraper around
// s.Escape.<c>.  If c doesn't reprsent a known color, ColorCode returns the
// empty string.
func (s *Shell[T]) ColorCode(c Color) string {
	switch c {
	case ColorBlack:
		return string(s.Escape().Black)
	case ColorRed:
		return string(s.Escape().Red)
	case ColorGreen:
		return string(s.Escape().Green)
	case ColorYellow:
		return string(s.Escape().Yellow)
	case ColorBlue:
		return string(s.Escape().Blue)
	case ColorMagenta:
		return string(s.Escape().Magenta)
	case ColorCyan:
		return string(s.Escape().Cyan)
	case ColorWhite:
		return string(s.Escape().White)
	case ColorReset:
		return string(s.Escape().Reset)
	default:
		return ""
	}
}
