package gopatterns

import (
	"context"
	"fmt"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

type PackageTypes struct {
	Types   *types.Package
	FileSet *token.FileSet
	Files   []string
}

func LoadPackageTypes(ctx context.Context, root string) ([]PackageTypes, error) {
	config := &packages.Config{Context: ctx, Dir: root, Mode: packages.LoadSyntax, Fset: token.NewFileSet(), BuildFlags: []string{"-mod=readonly"}}
	loaded, err := loadSourcePackages(config, []string{"./..."}, nil)
	if err != nil {
		return nil, err
	}
	if err := patternPackageErrors(loaded, root); err != nil {
		return nil, err
	}
	return checkedPackageTypes(loaded)
}

func checkedPackageTypes(loaded []*packages.Package) ([]PackageTypes, error) {
	var result []PackageTypes
	for _, pkg := range loaded {
		if pkg.IllTyped || pkg.Types == nil || !pkg.Types.Complete() {
			return nil, fmt.Errorf("incomplete package types: %s", pkg.PkgPath)
		}
		if pkg.Name != "main" {
			result = append(result, PackageTypes{Types: pkg.Types, FileSet: pkg.Fset, Files: pkg.GoFiles})
		}
	}
	return result, nil
}
