package patterns

// Signature/graph compatibility alone does not show that introducing an
// abstraction improves this code. Only parameterization has a source verifier.
func UnsupportedAbstraction(candidate Candidate) bool {
	switch candidate.Kind {
	case Parameterize:
		return candidate.FixSpec == nil
	case TraitMethod, CapabilitySet, GenericFn:
		return true
	default:
		return false
	}
}
