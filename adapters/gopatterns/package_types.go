package gopatterns

import (
	"context"
	"fmt"
	"go/types"

	"golang.org/x/tools/go/packages"
)

func LoadPackageTypes(ctx context.Context, root string) ([]*types.Package, error) {
	config := &packages.Config{Context: ctx, Dir: root, Mode: packages.LoadTypes, BuildFlags: []string{"-mod=readonly"}}
	loaded, err := loadSourcePackages(config, []string{"./..."}, nil)
	if err != nil {
		return nil, err
	}
	if err := patternPackageErrors(loaded, root); err != nil {
		return nil, err
	}
	return checkedPackageTypes(loaded)
}

func checkedPackageTypes(loaded []*packages.Package) ([]*types.Package, error) {
	var result []*types.Package
	for _, pkg := range loaded {
		if pkg.IllTyped || pkg.Types == nil || !pkg.Types.Complete() {
			return nil, fmt.Errorf("incomplete package types: %s", pkg.PkgPath)
		}
		if pkg.Name != "main" {
			result = append(result, pkg.Types)
		}
	}
	return result, nil
}
