package callflow

import (
	"slices"
	"strconv"
	"strings"
)

// AllocationHistory is an immutable finite set of receiving call sites. It is
// comparable and can be used in an abstract address. Repeated sites identify
// allocation families; they do not create a growing call-depth trace.
type AllocationHistory struct{ key string }

func (history AllocationHistory) Extend(site string) (AllocationHistory, bool) {
	sites, repeated := extendAllocationSites(history.Sites(), site)
	quoted := make([]string, len(sites))
	for i, site := range sites {
		quoted[i] = strconv.Quote(site)
	}
	return AllocationHistory{key: strings.Join(quoted, "\n")}, repeated
}

// Sites returns an owned, sorted set. Zero history has no sites.
func (history AllocationHistory) Sites() []string {
	if history.key == "" {
		return nil
	}
	quoted := strings.Split(history.key, "\n")
	for i, site := range quoted {
		value, err := strconv.Unquote(site)
		if err != nil {
			panic(err)
		}
		quoted[i] = value
	}
	return quoted
}

func extendAllocationSites(sites []string, site string) ([]string, bool) {
	repeated := slices.Contains(sites, site)
	result := append(slices.Clone(sites), site)
	slices.Sort(result)
	return slices.Compact(result), repeated
}
