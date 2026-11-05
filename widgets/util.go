// Package widgets — shared utility functions.
package widgets

import "strings"

// htmlEscape escapes the characters that are unsafe in HTML text and
// quoted attribute values. Used by the remaining string-built widget
// chrome and actions; template-rendered fields rely on html/template.
func htmlEscape(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '&':
			result = append(result, '&', 'a', 'm', 'p', ';')
		case '<':
			result = append(result, '&', 'l', 't', ';')
		case '>':
			result = append(result, '&', 'g', 't', ';')
		case '"':
			result = append(result, '&', 'q', 'u', 'o', 't', ';')
		case '\'':
			result = append(result, '&', '#', '3', '9', ';')
		default:
			result = append(result, s[i])
		}
	}
	return string(result)
}

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
