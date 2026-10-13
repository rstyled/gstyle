package patternfix

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/adapters/gopatterns"
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

func TestUnsupportedSignatureDoesNotBlockOtherFixes(t *testing.T) {
	path := writeTempGo(t, `package sample
func Exported(id int, name string) {}
func guard(x int) int {
 if x > 0 {
  println(x)
  println("positive")
 } else { return -1 }
 return x
}
`)
	ext, err := gopatterns.ExtractForFix(context.Background(), "", []string{path})
	if err != nil {
		t.Fatal(err)
	}
	bad := &patterns.FixSpec{Kind: patterns.ValueObject, File: path, Context: ext.Sources, Params: map[string]string{"group": "id:int,name:string"}}
	good := &patterns.FixSpec{Kind: patterns.GuardClause, File: path, Line: 4, Context: ext.Sources, Params: map[string]string{"if_line": "4"}}
	var output bytes.Buffer
	count, _, err := applyAndTrack(context.Background(), []*patterns.FixSpec{bad, good}, options{}, &output)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || !strings.Contains(output.String(), "skipped value_object") || !strings.Contains(output.String(), "if x <= 0") {
		t.Fatalf("unsupported candidate blocks valid fix: %d %s", count, &output)
	}
}

func TestSignatureOverlapDoesNotPartiallyAppendFiles(t *testing.T) {
	sources := pdg.NewSourceContext(t.TempDir(), nil)
	first, second := filepath.Join(sources.Root, "first.go"), filepath.Join(sources.Root, "second.go")
	sources.Add(first, []byte("first"), nil)
	sources.Add(second, []byte("second"), nil)
	plan := fixPlan{sources: sources, files: []filePlan{{path: first, after: []byte("accepted")}}}
	overlay := map[string][]byte{first: []byte("conflict"), second: []byte("partial")}
	if err := appendSignatureFiles(&plan, overlay, first); !errors.Is(err, errOverlappingPlans) {
		t.Fatalf("expected overlap: %v", err)
	}
	if len(plan.files) != 1 || string(plan.files[0].after) != "accepted" {
		t.Fatal("rejected plan left partial edits")
	}
}

func TestMissingSignatureEvidenceIsStillFatal(t *testing.T) {
	plan := fixPlan{}
	spec := &patterns.FixSpec{Kind: patterns.ValueObject}
	if err := addSignaturePlans(&plan, []*patterns.FixSpec{spec}); err == nil || len(plan.skipped) != 0 {
		t.Fatalf("missing evidence became a skip: %v", err)
	}
}

func TestUnknownSignatureTargetIsNotSilentlySkipped(t *testing.T) {
	sources := pdg.NewSourceContext(t.TempDir(), nil)
	path := filepath.Join(sources.Root, "code.go")
	sources.Add(path, []byte("package sample\n"), nil)
	plan := fixPlan{sources: sources}
	spec := &patterns.FixSpec{Kind: patterns.ValueObject, File: path, Context: sources, Targets: []string{"missing.target"}, Params: map[string]string{"group": "id:int,name:string"}}
	if err := addSignaturePlans(&plan, []*patterns.FixSpec{spec}); err == nil || len(plan.skipped) != 0 {
		t.Fatalf("unknown target became a skip: %v", err)
	}
}
