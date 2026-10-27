package quality

import (
	"fmt"
	"slices"
	"strings"

	"github.com/rstyled/gstyle/domain/guard"
)

type Override struct {
	Paths  []string
	Config Config
}

func (c Config) For(path string) Config {
	for _, override := range slices.Backward(c.Overrides) {
		if override.matches(path) {
			return override.Config
		}
	}
	return c
}

func (o Override) matches(path string) bool {
	return slices.ContainsFunc(o.Paths, func(scope string) bool {
		return path == scope || strings.HasPrefix(path, scope+"/")
	})
}

func (c Config) validateOverrides() error {
	for index, override := range c.Overrides {
		if err := override.validate(); err != nil {
			return fmt.Errorf("overrides[%d]: %w", index, err)
		}
	}
	return nil
}

func (o Override) validate() error {
	if len(o.Paths) == 0 || slices.ContainsFunc(o.Paths, invalidScope) {
		return fmt.Errorf("paths must list repository-relative files or directories")
	}
	if len(o.Config.Overrides) != 0 {
		return fmt.Errorf("overrides cannot nest")
	}
	return o.Config.Validate()
}

func invalidScope(scope string) bool { return !guard.ValidRelativePath(scope) }
