// Package query provides checked access to canonical PDG evidence records.
package query

// ReferenceIndex resolves a one-based reference without indexing a table.
// Zero, negative table lengths and references beyond the table are unavailable.
func ReferenceIndex(ref Ref, count int) (int, bool) {
	if count < 0 || ref == 0 || uint64(ref) > uint64(count) {
		return 0, false
	}
	return int(ref) - 1, true
}

// EvidenceAt resolves an actual record and preserves absence separately from a
// present record whose status is Unknown. The evidence table is borrowed.
func EvidenceAt(evidence []Capability, ref Ref) (Capability, bool) {
	index, ok := ReferenceIndex(ref, len(evidence))
	if !ok {
		return Capability{}, false
	}
	return evidence[index], true
}

// EvidenceStatus preserves unavailable evidence as Unknown.
func EvidenceStatus(evidence []Capability, ref Ref) Status {
	record, ok := EvidenceAt(evidence, ref)
	if !ok {
		return Unknown
	}
	return record.Status
}

// CoverageSupported admits an explicitly inapplicable capability, but never
// unknown, unsupported or approximate evidence.
func CoverageSupported(status Status) bool {
	return status == Supported || status == Known || status == NotApplicable
}

// DependencySupported requires affirmative evidence for an existing edge.
// NotApplicable cannot certify an edge that is actually present.
func DependencySupported(status Status) bool {
	return status == Supported || status == Known
}

// DependencyAdmitted is the edge-evidence rule used by semantic patterns.
// Missing records and non-affirmative statuses cannot certify a dependency.
func DependencyAdmitted(evidence []Capability, ref Ref) bool {
	return DependencySupported(EvidenceStatus(evidence, ref))
}

// BooleanValue keeps a known false distinct from unavailable evidence. Unknown
// payload bits are not semantic values, even if a malformed record sets them.
func BooleanValue(fact BooleanFact) (bool, bool) {
	if fact.Status != Known {
		return false, false
	}
	return fact.Value, true
}

// KnownFalse is the checked Boolean gate used for non-variadic interfaces.
func KnownFalse(fact BooleanFact) bool {
	value, known := BooleanValue(fact)
	return known && !value
}
