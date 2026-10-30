package quality

import (
	"slices"
	"strings"
)

func ReachableHelpers(name string, calls []Call, lookup func(string) (Helper, bool)) []Helper {
	seen := map[string]bool{NormalizeCallee(name): true}
	pending := slices.Clone(calls)
	var result []Helper
	for len(pending) > 0 {
		call := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if !call.Local || call.Dynamic {
			continue
		}
		key := NormalizeCallee(call.Callee)
		helper, found := lookup(key)
		if !found || seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, helper)
		pending = append(pending, helper.Calls...)
	}
	slices.SortFunc(result, func(a, b Helper) int { return strings.Compare(a.Name, b.Name) })
	return result
}
