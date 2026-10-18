package deferflow

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/pdg"
	"slices"
)

// Index owns validated registration metadata, independent of dynamic execution.
// Its zero value describes no registrations. Returned sites are owned copies.
type Index struct {
	sites  []Site
	lookup map[pdg.Ref]Site
}

func NewIndex(graph *pdg.Graph) (Index, error) {
	if graph == nil {
		return Index{}, fmt.Errorf("deferred index requires a canonical graph")
	}
	sites, err := Sites(graph)
	if err != nil {
		return Index{}, err
	}
	index := Index{sites: sites, lookup: make(map[pdg.Ref]Site, len(sites))}
	for _, site := range sites {
		index.lookup[site.Registration] = site
	}
	return index, nil
}
func (index Index) Sites() []Site {
	sites := slices.Clone(index.sites)
	for i := range sites {
		sites[i].Captures = slices.Clone(sites[i].Captures)
	}
	return sites
}
