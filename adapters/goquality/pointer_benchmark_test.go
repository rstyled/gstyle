package goquality

import (
	"context"
	"os"
	"testing"
)

// The same archived source is supplied to baseline and candidate processes.
func BenchmarkHelperReturnExtraction(b *testing.B) {
	root := os.Getenv("GSTYLE_BENCH_ROOT")
	if root == "" {
		b.Skip("set GSTYLE_BENCH_ROOT to a fixed source corpus")
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := (Analyzer{Root: root}).Extract(context.Background(), nil); err != nil {
			b.Fatal(err)
		}
	}
}
