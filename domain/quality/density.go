package quality

// Density counts each source call once per effect kind, retaining every
// underlying effect as evidence. Missing call identities stay independent.
type densitySite struct {
	site int
	kind Kind
}

type densitySites map[densitySite]bool

func (sites densitySites) counts(effect Effect) bool {
	if effect.Site == 0 {
		return true
	}
	key := densitySite{effect.Site, effect.Kind}
	if sites[key] {
		return false
	}
	sites[key] = true
	return true
}
