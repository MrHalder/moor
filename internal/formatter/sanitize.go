package formatter

import "strings"

// SanitizeDisplay strips C0 control characters (U+0000–U+001F), DEL (U+007F),
// and C1 control characters (U+0080–U+009F) from a string to prevent terminal
// escape sequence injection via process names, command lines, or other OS-sourced data.
func SanitizeDisplay(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 32 || r == 127 || (r >= 0x80 && r <= 0x9F) {
			return -1
		}
		return r
	}, s)
}
