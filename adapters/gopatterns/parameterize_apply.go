package gopatterns

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"

	"github.com/rstyled/gstyle/domain/patterns"
)

// A proof is tied to the complete source snapshot. Earlier fixes must re-mine
// before parameterization; stale evidence cannot authorize another pair.
func applyParameterizeFix(spec *patterns.FixSpec, src []byte) ([]byte, error) {
	a, b, signatures, err := parameterizationProof(spec, src)
	if err != nil {
		return nil, err
	}
	fset, file, err := parseSpec(spec, src)
	if err != nil {
		return nil, err
	}
	normalizeParameterizationFile(spec, fset, file)
	candidates := []patterns.Candidate{{Kind: patterns.Parameterize, Sites: []patterns.Site{{Name: a}, {Name: b}}}}
	out, _, err := fixParameterizeWithTypes(fset, file, src, parameterizeConfig{typedInterfaces: true, candidates: candidates, signatures: signatures, helperName: spec.Params["helper_name"]})
	return out, err
}

func parameterizationProof(spec *patterns.FixSpec, src []byte) (string, string, map[string]string, error) {
	if spec.Params["substitution_policy"] != substitutionPolicy || !validHelperIdentifier(spec.Params["helper_name"]) {
		return "", "", nil, fmt.Errorf("parameterization requires a current substitution proof; re-run patterns")
	}
	if spec.Params["source_sha256"] != fmt.Sprintf("%x", sha256.Sum256(src)) {
		return "", "", nil, fmt.Errorf("parameterization requires a current typed source proof; re-run patterns")
	}
	a, b := spec.Params["pair_a"], spec.Params["pair_b"]
	if !validParameterizationPair(a, b) {
		return "", "", nil, fmt.Errorf("parameterization requires one explicit function pair")
	}
	var signatures map[string]string
	if err := json.Unmarshal([]byte(spec.Params["callee_types"]), &signatures); err != nil {
		return "", "", nil, fmt.Errorf("parameterization callable types: %w", err)
	}

	return a, b, signatures, nil
}

func validParameterizationPair(a, b string) bool { return a != "" && b != "" && a != b }

func normalizeParameterizationFile(spec *patterns.FixSpec, fset *token.FileSet, file *ast.File) {
	first, second := namedFreeFunction(file, spec.Params["pair_a"]), namedFreeFunction(file, spec.Params["pair_b"])
	if first == nil || second == nil {
		return
	}
	path := fixSourcePath(spec.Context.Root, spec.File)
	snapshot, _ := spec.Context.Snapshot(spec.Context.Reference(path))
	source := &parameterizationSource{snapshot: snapshot, fset: fset, file: file}
	normalizeParameterizationImports(&parameterizationPair{first: source, second: source, a: first, b: second})
}
