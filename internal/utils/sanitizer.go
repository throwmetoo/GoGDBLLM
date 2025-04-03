package utils

import (
	"regexp"
	"strings"
)

// ansiRegex defines the pattern for ANSI escape codes.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)

// gdbControlChars defines specific GDB control characters (SOH, STX) to remove.
// Using direct byte representation for clarity.
var gdbControlChars = []byte{0x01, 0x02} // Corresponds to \u0001 and \u0002

// StripAnsiAndControlChars removes ANSI escape codes and specific GDB control characters from a string.
func StripAnsiAndControlChars(str string) string {
	// Remove ANSI escape codes
	sanitized := ansiRegex.ReplaceAllString(str, "")

	// Remove specific GDB control characters (\u0001, \u0002)
	// We convert the string to bytes, filter, and convert back.
	// This is generally safer for handling potential multi-byte characters
	// if they were present, though less critical for control chars.
	var result strings.Builder
	result.Grow(len(sanitized)) // Pre-allocate roughly the needed size

	for _, r := range sanitized {
		isControlChar := false
		for _, controlByte := range gdbControlChars {
			// Check if the rune's byte representation matches a control character
			// This check is simplistic and assumes single-byte representation for control chars.
			// A more robust method might be needed if dealing with complex encodings,
			// but for ASCII/UTF-8 control chars 0x01/0x02, this works.
			if byte(r) == controlByte {
				isControlChar = true
				break
			}
		}
		if !isControlChar {
			result.WriteRune(r)
		}
	}

	return result.String()
}
