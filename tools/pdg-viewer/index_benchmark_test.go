package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func BenchmarkLargeRepositoryIndex(b *testing.B) {
	root := b.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module scale.test/repo\ngo 1.26.0\n"), 0600); err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 334; i++ {
		dir := filepath.Join(root, fmt.Sprintf("pkg%02d", i%20))
		if err := os.MkdirAll(dir, 0700); err != nil {
			b.Fatal(err)
		}
		var content strings.Builder
		fmt.Fprintf(&content, "package pkg%02d\n", i%20)
		for j := 0; j < 60; j++ {
			fmt.Fprintf(&content, "func F%d_%d(n int)int{\n%sreturn n\n}\n", i, j, strings.Repeat("n++\n", 17))
		}
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("file%d.go", i)), []byte(content.String()), 0600); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index, err := indexRepository(root)
		if err != nil {
			b.Fatal(err)
		}
		if len(index.Symbols) != 20040 {
			b.Fatal("incomplete index")
		}
		b.ReportMetric(float64(index.Lines), "source_lines")
	}
}
