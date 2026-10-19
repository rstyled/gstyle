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
		intersectPackageProof(common, pkg.PkgPath, proof)
	}
	for pkg, proof := range proofs {
		for object := range proof {
			if !common[pkg.PkgPath][resolvedVariable(object)] {
				delete(proof, object)
			}
		}
	}
	return proofs
}

func intersectPackageProof(common map[string]map[string]bool, path string, proof map[types.Object]bool) {
	current := map[string]bool{}
	for object := range proof {
		current[resolvedVariable(object)] = true
	}
	prior, found := common[path]
	if !found {
		common[path] = current
		return
	}
	for name := range prior {
		if !current[name] {
			delete(prior, name)
		}
	}
}
