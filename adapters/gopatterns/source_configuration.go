package gopatterns

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/rstyled/gstyle/domain/pdg"
)

// Fix evidence includes module/workspace inputs as well as Go source bytes.
func captureFixConfiguration(sources *pdg.SourceContext, options sourceLoadOptions) error {
	if !options.tests || sources == nil {
		return nil
	}
	paths := append([]string{sources.Root}, sources.Queries...)
	for _, path := range paths {
		if err := captureConfigurationParents(sources, configurationDirectory(path)); err != nil {
			return err
		}
	}
	workspace := os.Getenv("GOWORK")
	if filepath.IsAbs(workspace) {
		return captureConfigurationFile(sources, workspace)
	}
	return nil
}

func configurationDirectory(query string) string {
	if strings.HasPrefix(query, "file=") {
		return filepath.Dir(strings.TrimPrefix(query, "file="))
	}
	return strings.TrimSuffix(query, string(filepath.Separator)+"...")
}

func captureConfigurationParents(sources *pdg.SourceContext, directory string) error {
	for {
		for _, name := range []string{"go.mod", "go.sum", "go.work", "go.work.sum"} {
			if err := captureConfigurationFile(sources, filepath.Join(directory, name)); err != nil {
				return err
			}
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return nil
		}
		directory = parent
	}
}

func captureConfigurationFile(sources *pdg.SourceContext, path string) error {
	if sources.Reference(path) != 0 {
		return nil
	}
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		sources.RecordMissing(path)
		return nil
	}
	if err != nil {
		return err
	}
	sources.Add(path, content, nil)
	return nil
}
