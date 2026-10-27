package quality

import (
	"fmt"
	"strings"

	"github.com/rstyled/gstyle/domain/guard"
)

type Scope struct {
	Path   string
	Config Config
}

func Covers(scope, path string) bool {
	return path == scope || strings.HasPrefix(path, scope+"/")
}

func (c Config) For(path string) Config {
	result, depth := c, -1
	for _, scope := range c.Scopes {
		if Covers(scope.Path, path) && len(scope.Path) > depth {
			result, depth = scope.Config, len(scope.Path)
		}
	}
	return result
}

func (c Config) validateScopes() error {
	for _, scope := range c.Scopes {
		if err := scope.validate(); err != nil {
			return fmt.Errorf("overrides for %q: %w", scope.Path, err)
		}
	}
	return nil
}

func (s Scope) validate() error {
	if !guard.ValidRelativePath(s.Path) {
		return fmt.Errorf("paths must list repository-relative files or directories")
	}
	if len(s.Config.Scopes) != 0 {
		return fmt.Errorf("overrides cannot nest")
	}
	return s.Config.Validate()
}
