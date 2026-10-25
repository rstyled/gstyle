package gorisk

import (
	"go/types"
	"path/filepath"
	"strings"

	"github.com/rstyled/gstyle/domain/changerisk"
)

func apiDeclaration(api API, change changerisk.APIChange) *changerisk.APIDeclaration {
	pkg := api.Packages[change.Package]
	if pkg == nil {
		return nil
	}
	source := api.Sources[change.Package]
	if strings.HasPrefix(change.Message, "package ") {
		return packageDeclaration(api.Root, source.Files, pkg)
	}
	object := apiObject(pkg, change.Message)
	if object == nil {
		return nil
	}
	result := &changerisk.APIDeclaration{Declaration: types.ObjectString(object, apiQualifier(pkg))}
	if source.FileSet == nil {
		return result
	}
	position := source.FileSet.PositionFor(object.Pos(), false)
	result.Path = repositoryPath(api.Root, position.Filename)
	if result.Path != "" {
		result.Line = position.Line
	}
	return result
}

func apiQualifier(current *types.Package) types.Qualifier {
	return func(pkg *types.Package) string {
		if pkg == current {
			return ""
		}
		return pkg.Name()
	}
}

func packageDeclaration(root string, files []string, pkg *types.Package) *changerisk.APIDeclaration {
	result := &changerisk.APIDeclaration{Declaration: "package " + pkg.Name()}
	if len(files) != 0 {
		result.Path = repositoryPath(root, filepath.Dir(files[0]))
	}
	for _, name := range pkg.Scope().Names() {
		if pkg.Scope().Lookup(name).Exported() {
			result.Exports++
		}
	}
	return result
}

func apiObject(pkg *types.Package, message string) types.Object {
	name, _, _ := strings.Cut(message, ":")
	name, _, _ = strings.Cut(name, ",")
	name = strings.NewReplacer("(*", "", ")", "").Replace(name)
	owner, member, found := strings.Cut(name, ".")
	owner, _, _ = strings.Cut(owner, "[")
	object := pkg.Scope().Lookup(owner)
	if !found || object == nil {
		return object
	}
	field, _, _ := types.LookupFieldOrMethod(object.Type(), true, pkg, member)
	return field
}

func repositoryPath(root, name string) string {
	if root == "" || name == "" {
		return ""
	}
	path, err := filepath.Rel(root, name)
	if err != nil || path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
		return ""
	}
	return filepath.ToSlash(path)
}
