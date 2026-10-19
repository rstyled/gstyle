package goquality

import (
	"go/types"

	"golang.org/x/tools/go/packages"
)

// A physical function can occur in production and augmented test builds.
// Retain only global-read proofs shared by every loaded variant of its package.
func stablePackageGlobals(pkgs []*packages.Package) map[*packages.Package]map[types.Object]bool {
	return packageProofs(pkgs, stableGlobals)
}

// Keep only proofs shared by all loaded variants of each package.
func packageProofs(pkgs []*packages.Package, prove func(*packages.Package) map[types.Object]bool) map[*packages.Package]map[types.Object]bool {
	proofs := map[*packages.Package]map[types.Object]bool{}
	common := map[string]map[string]bool{}
	for _, pkg := range pkgs {
		proof := prove(pkg)
		proofs[pkg] = proof
		intersectPackageProof(common, pkg, proof)
	}
	for pkg, proof := range proofs {
		for object := range proof {
			if !common[pkg.PkgPath][packageProofKey(pkg, object)] {
				delete(proof, object)
			}
		}
	}
	return proofs
}

func intersectPackageProof(common map[string]map[string]bool, pkg *packages.Package, proof map[types.Object]bool) {
	current := map[string]bool{}
	for object := range proof {
		current[packageProofKey(pkg, object)] = true
	}
	prior, found := common[pkg.PkgPath]
	if !found {
		common[pkg.PkgPath] = current
		return
	}
	for name := range prior {
		if !current[name] {
			delete(prior, name)
		}
	}
}

func packageProofKey(pkg *packages.Package, object types.Object) string {
	// Same-named fields belong to distinct records. Physical declarations also
	// match across test variants without depending on //line source mappings.
	if field, ok := object.(*types.Var); ok && field.IsField() {
		return physicalPosition(pkg.Fset, field.Pos()).String()
	}
	return resolvedVariable(object)
}
