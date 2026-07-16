// Package widgets — shared utility functions.
package widgets

import "strings"

// stripTags removes HTML tags for plain-text display.
func stripTags(s string) string {
	var buf strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			buf.WriteRune(r)
		}
	}
	result := strings.TrimSpace(buf.String())
	parts := strings.Fields(result)
	return strings.Join(parts, " ")
}
