package text

import "strings"

// Dedent un-indents a block of multi-line text, using the first non-empty
// line to detect the indent level.
func Dedent(s string) string {
	lines := trimEmptyLeadingLines(strings.Split(s, "\n"))
	in := indent(lines[0])
	for i := 0; i < len(lines); i++ {
		lines[i] = strings.TrimPrefix(lines[i], in)
	}
	return strings.Join(lines, "\n")
}

// trimEmptyLeadingLines returns a subslice eliding any prefix of empty
// strings at the head of the given lines slice.
func trimEmptyLeadingLines(lines []string) []string {
	i := 0
	for ; i < len(lines); i++ {
		if len(lines[i]) > 0 {
			return lines[i:]
		}
	}
	return nil
}

// indent returns any space or tab indentation prefix substring from s.
func indent(s string) string {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ' ', '\t':
		default:
			return s[:i]
		}
	}
	return ""
}
