// Package sanitization provides bounded, log-safe representations of untrusted input.
package sanitization

import (
	"strings"
	"unicode/utf8"
)

const MaxUserAgentLength = 256

// UserAgent removes line breaks and bounds the value before it reaches logs.
func UserAgent(value string) string {
	value = strings.NewReplacer("\r", " ", "\n", " ").Replace(value)
	if utf8.RuneCountInString(value) <= MaxUserAgentLength {
		return value
	}
	return string([]rune(value)[:MaxUserAgentLength])
}
