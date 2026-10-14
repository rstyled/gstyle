// Package linecount counts ordered, inclusive spans of code-bearing lines.
package linecount

// Counter is an immutable accumulator. Its zero value is ready for use.
// Token spans must arrive in source order; the next span starts at or after
// the preceding span's last line. Add rejects spans that violate this contract.
type Counter struct {
	Lines   int
	Through int
}

// Add counts a token's lines once, including a multiline literal's interior.
// A valid counter satisfies 0 <= Lines <= Through. The scanner supplies
// 1 <= first <= last and first >= Through. The second result reports whether
// the span and counter satisfy these conditions.
func (c Counter) Add(first, last int) (Counter, bool) {
	if !c.accepts(first, last) {
		return c, false
	}
	if last == c.Through {
		return c, true
	}
	if first == c.Through {
		first++
	}
	return Counter{Lines: c.Lines + (last - first + 1), Through: last}, true
}

func (c Counter) accepts(first, last int) bool {
	return c.Lines >= 0 && c.Through >= c.Lines && first >= 1 && last >= first && first >= c.Through
}

// Exceeds applies the strict threshold used by function-length diagnostics.
func Exceeds(actual, limit int64) bool {
	return actual > limit
}
