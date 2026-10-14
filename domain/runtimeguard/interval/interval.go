package interval

type Range struct {
	Lower, Upper       int64
	HasLower, HasUpper bool
}

func Join(a, b Range) Range {
	r := Range{Lower: min(a.Lower, b.Lower), Upper: max(a.Upper, b.Upper),
		HasLower: a.HasLower && b.HasLower, HasUpper: a.HasUpper && b.HasUpper}
	if !r.HasLower {
		r.Lower = 0
	}
	if !r.HasUpper {
		r.Upper = 0
	}
	return r
}

func NegativeIndex(index, upperChecked bool, r Range) bool {
	if !index || !upperChecked {
		return false
	}
	return !r.HasLower || r.Lower < 0
}
