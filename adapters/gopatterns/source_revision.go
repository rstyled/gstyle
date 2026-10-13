package gopatterns

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rstyled/gstyle/domain/pdg"
)

func revisionSource(path string, sources *pdg.SourceContext) ([]byte, error) {
	if sources == nil {
		return os.ReadFile(path)
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(sources.Root, path)
	}
	_, text, ok := sources.Source(sources.Reference(path))
	if !ok {
		return nil, fmt.Errorf("source evidence is missing: %s", path)
	}
	return []byte(text), nil
}
