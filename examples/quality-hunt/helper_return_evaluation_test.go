package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Fixed external-source helpers from three packages/projects. The method is an
// upstream test fixture, not a production-project sample. Bodies stay unchanged.
func TestHelperReturnExternalEvaluation(t *testing.T) {
	definitions := []assuranceFamily{
		{"external-ptr-scalar", "local := Ptr(1)\n*local = 2", 0, true},
		{"external-ptr-shared-child", "local := Ptr(shared)\n(*local).Count++", 1, true},
		{"external-borrowed-method", "local := Type(0).Method(&shared.Count)\n*local += 1", 1, true},
		{"external-method-expression", "local := Type.Method(Type(0), &shared.Count)\n*local += 1", 1, true},
		{"external-mixed-borrowed", "local := MaxPtr(&shared.Count, &shared.Array[0])\n*local += 1", 1, true},
		{"external-nil-result", "_ = MaxPtr(nil, nil)", 0, true},
		{"external-literal-record", "local := NewIdent(\"value\")\nlocal.NamePos++", 0, true},
	}
	names := []string{"ExternalPtrScalar", "ExternalPtrChild", "ExternalMethod", "ExternalExpression", "ExternalMixed", "ExternalNil", "ExternalIdent"}
	var cases []specimen
	families := map[string]assuranceFamily{}
	for index, family := range definitions {
		name := names[index]
		cases = append(cases, specimen{Name: name, Base: name, Family: family.name, Variant: "original", Body: family.body})
		families[name] = family
	}
	source := bytes.Replace(program(cases), []byte("package repro\n"), []byte("package repro\nimport(\"go/ast\";\"go/token\")\ntype Ident = ast.Ident\ntype Type int\n"), 1)
	directory := "testdata/helper-return-holdout"
	data, err := os.ReadFile(filepath.Join(directory, "sources.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Sources []struct {
			Name string `json:"name"`
			Hash string `json:"excerpt_sha256"`
		} `json:"sources"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Sources) != 4 {
		t.Fatal("external source manifest must retain its four fixed helpers")
	}
	for _, entry := range manifest.Sources {
		body, err := os.ReadFile(filepath.Join(directory, entry.Name+".go.txt"))
		if err != nil || assuranceHash(body) != entry.Hash {
			t.Fatalf("external source changed: %s: %v", entry.Name, err)
		}
		source = append(source, body...)
	}
	assuranceAssess(t, cases, families, source)
}
