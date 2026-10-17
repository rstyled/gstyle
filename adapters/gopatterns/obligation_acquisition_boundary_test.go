package gopatterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"io/fs"
	"runtime"
	"slices"
	"testing"
)

func TestAcquisitionBindingAndPriorDefersAffectRelease(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		released   bool
	}{

		{"argument fault precedes acquisition", `file,err:=namedPair(files,names[index]);if err!=nil{return nil,err};defer file.Close();return nil,nil`, true},
		{"var tuple binding", `var file,err=pair(files);if err!=nil{return nil,err};defer file.Close();return nil,nil`, true},
		{"partial tuple store", `var file fs.File;var failures []error;file,failures[0]=pair(files);file.Close();return nil,nil`, false},

		{"conditional prior panic", `if choose {defer panic(7)};file,err:=pair(files);if err!=nil{return nil,err};return file,nil`, false},
		{"identity defers in prefix loop", `for choose {defer println("wait")};file,err:=pair(files);if err!=nil{return nil,err};return file,nil`, true},
		{"pure prefix loop", `for choose {};file,err:=pair(files);if err!=nil{return nil,err};return file,nil`, true},
		{"unrelated nonreturning branch", `if choose {for {defer println("wait")}};file,err:=pair(files);if err!=nil{return nil,err};return file,nil`, true},
		{"new close after registration loop", `for choose {defer panic(7)};file,err:=pair(files);if err!=nil{return nil,err};defer file.Close();return nil,nil`, true},
		{"old capture does not close new file", `file:=other;defer file.Close();file,err:=pair(files);if err!=nil{return nil,err};return nil,nil`, false},
		{"prior Goexit cancels transfer", `defer runtime.Goexit();file,err:=pair(files);if err!=nil{return nil,err};return file,nil`, false},
		{"prior panic cancels transfer", `defer panic(7);file,err:=pair(files);if err!=nil{return nil,err};return file,nil`, false},
		{"prior print preserves transfer", `defer println("exit");file,err:=pair(files);if err!=nil{return nil,err};return file,nil`, true},
		{"new close precedes prior Goexit", `defer runtime.Goexit();file,err:=pair(files);if err!=nil{return nil,err};defer file.Close();return nil,nil`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pkg := loadTestPackage(t, `package p;import("io/fs";"runtime");var _=runtime.Goexit;var choose bool;var other fs.File;var names []string;var index int;func caller(files fs.FS)(fs.File,error){`+tc.body+`};func pair(files fs.FS)(fs.File,error){return files.Open("file")};func namedPair(files fs.FS,name string)(fs.File,error){return files.Open(name)}`)
			summaries, err := packageResourceSummaries(pkg)
			if err != nil {
				t.Fatal(err)
			}
			fn := findFunc(pkg.Syntax[0], "caller")
			graph, err := buildGraph(pkg, fn, "p.go", pdg.NewBuilder())
			if err != nil {
				t.Fatal(err)
			}
			ctx := extractCtx{fileSources: summaries.sources, fileFailures: summaries.failures, helpers: summaries.helpers, exits: summaries.exits}
			released, _, err := fileObligationFacts(pkg, fn, ctx, &graph)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(released) > 0; got != tc.released {
				t.Fatalf("released=%v want %v", released, tc.released)
			}
		})
	}
}

type partialBindingFile struct {
	fs.File
	closed bool
}

func (file *partialBindingFile) Close() error { file.closed = true; return nil }
func TestNativePartialTupleStoreLeaksReturnedResource(t *testing.T) {
	file := &partialBindingFile{}
	var bound fs.File
	acquired, panicked := false, false
	acquire := func() (fs.File, error) { acquired = true; return file, nil }
	func() {
		defer func() { panicked = recover() != nil }()
		var failures []error
		bound, failures[0] = acquire()
		_ = bound.Close()
	}()
	if !acquired || !panicked || bound != file || file.closed {
		t.Fatalf("acquired=%v panic=%v first bound=%v closed=%v", acquired, panicked, bound == file, file.closed)
	}
}

func TestNativePriorExitsCancelAcquiredReturn(t *testing.T) {
	for _, action := range []func(){runtime.Goexit, func() { panic(7) }} {
		file := &partialBindingFile{}
		done := make(chan bool, 1)
		go func() {
			returned := false
			defer func() { _ = recover(); done <- returned }()
			func() fs.File { defer action(); return file }()
			returned = true
		}()
		if <-done || file.closed {
			t.Fatal("exceptional prior defer transfers or closes acquired resource")
		}
	}
}
func TestNativeEarlierCaptureRetainsOldFile(t *testing.T) {
	oldFile, newFile := &partialBindingFile{}, &partialBindingFile{}
	var file fs.File = oldFile
	func() {
		defer file.Close()
		file = newFile
	}()
	if !oldFile.closed || newFile.closed {
		t.Fatal("earlier deferred receiver follows later binding")
	}
}

func TestAcquisitionBoundaryRequiresCanonicalBindingFacts(t *testing.T) {
	for _, fact := range []string{"gstyle.go:value-operands", "gstyle.go:evaluation-region"} {
		t.Run(fact, func(t *testing.T) {
			pkg := loadTestPackage(t, `package p;import "io/fs";func caller(files fs.FS)error{file,err:=pair(files);if err!=nil{return err};defer file.Close();return nil};func pair(files fs.FS)(fs.File,error){return files.Open("file")}`)
			summaries, err := packageResourceSummaries(pkg)
			if err != nil {
				t.Fatal(err)
			}
			fn := findFunc(pkg.Syntax[0], "caller")
			graph, err := buildGraph(pkg, fn, "p.go", pdg.NewBuilder())
			if err != nil {
				t.Fatal(err)
			}
			ctx := extractCtx{fileSources: summaries.sources, fileFailures: summaries.failures, helpers: summaries.helpers, exits: summaries.exits}
			released, _, err := fileObligationFacts(pkg, fn, ctx, &graph)
			if err != nil || len(released) != 1 {
				t.Fatalf("baseline release=%v %v", released, err)
			}
			reference, err := newReleaseGraph(&graph, pkg.Fset)
			if err != nil {
				t.Fatal(err)
			}
			owner := reference.ref(fn.Body.List[0])
			attrs := graph.NodeAttributes(graph.Nodes[owner-1])
			set := graph.Attributes[graph.Nodes[owner-1].Attributes-1]
			changed := false
			graph.AttributeTables.Extensions[set.Extensions-1] = slices.DeleteFunc(slices.Clone(attrs.Extensions), func(extension pdg.Extension) bool {
				if graph.Tables.Text(extension.Name) == fact {
					changed = true
					return true
				}
				return false
			})
			if !changed {
				t.Fatal("missing fact to remove")
			}
			released, _, err = fileObligationFacts(pkg, fn, ctx, &graph)
			if err != nil {
				t.Fatal(err)
			}
			if len(released) != 0 {
				t.Fatal("source syntax replaces missing binding facts")
			}
		})
	}
}

func TestLaterHandoffCannotHideAcquisitionBindingFault(t *testing.T) {
	pkg := loadTestPackage(t, `package p;import "io/fs";func caller(files fs.FS){var file fs.File;var failures []error;file,failures[0]=pair(files);consume(file)};func pair(files fs.FS)(fs.File,error){return files.Open("file")};func consume(fs.File){}`)
	summaries, err := packageResourceSummaries(pkg)
	if err != nil {
		t.Fatal(err)
	}
	fn := findFunc(pkg.Syntax[0], "caller")
	graph, err := buildGraph(pkg, fn, "p.go", pdg.NewBuilder())
	if err != nil {
		t.Fatal(err)
	}
	ctx := extractCtx{fileSources: summaries.sources, fileFailures: summaries.failures, helpers: summaries.helpers, exits: summaries.exits}
	released, sites, err := fileObligationFacts(pkg, fn, ctx, &graph)
	if err != nil {
		t.Fatal(err)
	}
	if len(released) != 0 || len(sites) != 1 {
		t.Fatalf("released=%v sites=%v; later handoff hides earlier lost result", released, sites)
	}
}

func TestNonreturningHelperPreventsLaterHandoff(t *testing.T) {
	pkg := loadTestPackage(t, `package p;import("io/fs";"runtime");func caller(files fs.FS){file,err:=pair(files);if err!=nil{return};stops();consume(file)};func pair(files fs.FS)(fs.File,error){return files.Open("file")};func stops(){runtime.Goexit()};func consume(fs.File){}`)
	summaries, err := packageResourceSummaries(pkg)
	if err != nil {
		t.Fatal(err)
	}
	fn := findFunc(pkg.Syntax[0], "caller")
	graph, err := buildGraph(pkg, fn, "p.go", pdg.NewBuilder())
	if err != nil {
		t.Fatal(err)
	}
	ctx := extractCtx{fileSources: summaries.sources, fileFailures: summaries.failures, helpers: summaries.helpers, exits: summaries.exits}
	released, sites, err := fileObligationFacts(pkg, fn, ctx, &graph)
	if err != nil {
		t.Fatal(err)
	}
	if len(released) != 0 || len(sites) != 1 {
		t.Fatalf("released=%v sites=%v; unreachable handoff suppresses ownership", released, sites)
	}
}
