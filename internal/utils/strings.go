package utils

import "strings"

func FormatSpaces(s []byte) string {
	buf := strings.Builder{}
	for _, c := range s {
		switch c {
		case '\t':
			buf.WriteString(`\t`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\v':
			buf.WriteString(`\v`)
		case '\f':
			buf.WriteString(`\f`)
		default:
			buf.WriteByte(c)
		}
	}
	return buf.String()
}
