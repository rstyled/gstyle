package cpp

import "strings"

func codeLines(source string) int {
	line, previous, count := 1, 0, 0
	for len(source) != 0 {
		size, code := cppToken(source)
		last := line + strings.Count(source[:size], "\n")
		if code {
			count += last - max(previous+1, line) + 1
			previous = last
		}
		line, source = last, source[size:]
	}
	return count
}

func cppToken(source string) (int, bool) {
	switch {
	case strings.ContainsRune(" \t\r\n\v\f", rune(source[0])):
		return 1, false
	case strings.HasPrefix(source, "//"):
		return lineCommentLength(source), false
	case strings.HasPrefix(source, "/*"):
		return throughDelimiter(source, 2, "*/"), false
	case strings.HasPrefix(source, "R\""):
		return rawStringLength(source), true
	default:
		return cppValueLength(source), true
	}
}

func cppValueLength(source string) int {
	if source[0] >= '0' && source[0] <= '9' {
		return cppNumberLength(source)
	}
	if source[0] == '\'' || source[0] == '"' {
		return quotedLength(source)
	}
	return 1
}

func cppNumberLength(source string) int {
	i := 1
	for i < len(source) && strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_.'", rune(source[i])) {
		i++
	}
	return i
}

func throughDelimiter(source string, start int, delimiter string) int {
	end := strings.Index(source[start:], delimiter)
	if end < 0 {
		return len(source)
	}
	return start + end + len(delimiter)
}

func lineCommentLength(source string) int {
	for i := 2; i < len(source); i++ {
		if source[i] != '\n' {
			continue
		}
		prefix := strings.TrimSuffix(source[:i], "\r")
		if !strings.HasSuffix(prefix, "\\") {
			return i
		}
	}
	return len(source)
}

func rawStringLength(source string) int {
	open := strings.IndexByte(source[2:], '(')
	if open < 0 || open > 16 {
		return 1
	}
	return throughDelimiter(source, open+3, ")"+source[2:open+2]+"\"")
}

func quotedLength(source string) int {
	for i := 1; i < len(source); i++ {
		if source[i] == '\\' {
			i++
			continue
		}
		if source[i] == source[0] {
			return i + 1
		}
	}
	return len(source)
}
