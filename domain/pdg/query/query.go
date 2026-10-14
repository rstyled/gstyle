package query

func ReferenceIndex(ref Ref, count int) (int, bool) {
	if count < 0 || ref == 0 || uint64(ref) > uint64(count) {
		return 0, false
	}
	return int(ref) - 1, true
}

func EvidenceAt(evidence []Capability, ref Ref) (Capability, bool) {
	index, ok := ReferenceIndex(ref, len(evidence))
	if !ok {
		return Capability{}, false
	}
	return evidence[index], true
}

func EvidenceStatus(evidence []Capability, ref Ref) Status {
	record, ok := EvidenceAt(evidence, ref)
	if !ok {
		return Unknown
	}
	return record.Status
}

func CoverageSupported(status Status) bool {
	return status == Supported || status == Known || status == NotApplicable
}

func DependencySupported(status Status) bool {
	return status == Supported || status == Known
}

func DependencyAdmitted(evidence []Capability, ref Ref) bool {
	return DependencySupported(EvidenceStatus(evidence, ref))
}

func BooleanValue(fact BooleanFact) (bool, bool) {
	if fact.Status != Known {
		return false, false
	}
	return fact.Value, true
}

func KnownFalse(fact BooleanFact) bool {
	value, known := BooleanValue(fact)
	return known && !value
}
