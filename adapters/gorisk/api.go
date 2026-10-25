package gorisk

import (
	"cmp"
	"context"
	"go/types"
	"maps"
	"slices"

	"github.com/rstyled/gstyle/adapters/gopatterns"
	"github.com/rstyled/gstyle/domain/changerisk"
	"golang.org/x/exp/apidiff"
)

type API struct {
	Packages map[string]*types.Package
	Error    string
}

func ExtractAPI(ctx context.Context, root string) API {
	packages, err := gopatterns.LoadPackageTypes(ctx, root)
	if err != nil {
		return API{Error: "Go API type loading failed for the root module"}
	}
	result := map[string]*types.Package{}
	for _, pkg := range packages {
		result[pkg.Path()] = pkg
	}
	return API{Packages: result}
}

func CompareAPI(base, head API) changerisk.APIReport {
	for _, inventory := range []API{base, head} {
		if inventory.Error != "" {
			return changerisk.APIReport{Scope: changerisk.APIScope, Status: changerisk.APIUnavailable, Error: inventory.Error}
		}
		if inventory.Packages == nil {
			return changerisk.APIReport{Scope: changerisk.APIScope, Status: changerisk.APIUnavailable, Error: "Go API inventory unavailable"}
		}
	}
	changes := packageAPIChanges(base.Packages, head.Packages)
	limits := apiLimits(base.Packages, head.Packages)
	slices.SortFunc(changes, func(a, b changerisk.APIChange) int {
		return cmp.Or(cmp.Compare(a.Package, b.Package), cmp.Compare(a.Message, b.Message))
	})
	return changerisk.APIReport{Scope: changerisk.APIScope, Status: apiStatus(changes, limits), Changes: changes, Limits: limits, PackagesBefore: len(base.Packages), PackagesAfter: len(head.Packages)}
}

func packageAPIChanges(base, head map[string]*types.Package) []changerisk.APIChange {
	names := maps.Clone(base)
	maps.Copy(names, head)
	changes := []changerisk.APIChange{}
	for _, name := range slices.Sorted(maps.Keys(names)) {
		changes = append(changes, comparePackageAPI(name, base[name], head[name])...)
	}
	return changes
}

func comparePackageAPI(name string, base, head *types.Package) []changerisk.APIChange {
	if base == nil {
		return []changerisk.APIChange{{Package: name, Message: "package added", Compatible: true}}
	}
	if head == nil {
		return []changerisk.APIChange{{Package: name, Message: "package removed", Compatible: false}}
	}
	var changes []changerisk.APIChange
	if base.Name() != head.Name() {
		changes = append(changes, changerisk.APIChange{Package: name, Message: "package name changed from " + base.Name() + " to " + head.Name()})
	}
	for _, change := range apidiff.Changes(base, head).Changes {
		changes = append(changes, changerisk.APIChange{Package: name, Message: change.Message, Compatible: change.Compatible})
	}
	return changes
}

func apiStatus(changes []changerisk.APIChange, limits []string) changerisk.APIStatus {
	if slices.ContainsFunc(changes, func(change changerisk.APIChange) bool { return !change.Compatible }) {
		return changerisk.APIBreaking
	}
	if len(limits) != 0 {
		return changerisk.APIPartial
	}
	if len(changes) != 0 {
		return changerisk.APICompatible
	}
	return changerisk.APIUnchanged
}
