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

type Access struct {
	Node, Symbol        uint32
	Index, UpperChecked bool
	Bounds              Range
}

func Select(accesses []Access) []Access {
	seen := make(map[uint32]bool)
	var out []Access
	for i := 0; i < len(accesses); i++ {
		site := accesses[i]
		if !NegativeIndex(site.Index, site.UpperChecked, site.Bounds) {
			continue
		}
		if seen[site.Symbol] {
			continue
		}
		seen[site.Symbol] = true
		out = append(out, site)
	}
	return out
}

type Advisory struct {
	Path                                    string
	Line, EndLine                           int
	ID, Name                                string
	Kind                                    string
	ScoreMilli                              uint32
	Observation, Inference, CounterEvidence string
}

func Describe(path, function, name string, line uint32) Advisory {
	return Advisory{
		Path: path, Line: int(line), EndLine: int(line), ID: function, Name: function,
		Kind: "negative_index", ScoreMilli: 800,
		Observation:     "upper-bound check on " + name + " has no lower-bound check; " + name + " indexes a collection",
		Inference:       "verify negative indices are rejected or impossible before the index is used",
		CounterEvidence: "intent unknown; no automatic fix — verify against the contract",
	}
}
