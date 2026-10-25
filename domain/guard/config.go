package guard

import (
	"fmt"
	"path"
	"slices"
	"strings"
)

type Comments struct {
	Mode string `toml:"mode"`
}

type FileRule struct {
	ID               string   `toml:"id"`
	Extensions       []string `toml:"extensions"`
	AllowDirectories []string `toml:"allow_directories"`
}

type Config struct {
	Comments Comments   `toml:"comments"`
	Files    []FileRule `toml:"files"`
}

func (c Config) Validate() error {
	if !slices.Contains([]string{"", "added", "all"}, c.Comments.Mode) {
		return fmt.Errorf("guard.comments.mode must be added or all")
	}
	ids := []string{"", "source_comment"}
	for _, rule := range c.Files {
		if slices.Contains(ids, rule.ID) {
			return fmt.Errorf("guard file rules require distinct IDs other than source_comment")
		}
		if err := rule.validate(); err != nil {
			return err
		}
		ids = append(ids, rule.ID)
	}
	return nil
}

func (r FileRule) validate() error {
	if len(r.Extensions) == 0 {
		return fmt.Errorf("guard file rule %q requires extensions", r.ID)
	}
	for _, extension := range r.Extensions {
		if !validExtension(extension) {
			return fmt.Errorf("guard file rule %q has invalid extension %q", r.ID, extension)
		}
	}
	for _, directory := range r.AllowDirectories {
		if !validDirectory(directory) {
			return fmt.Errorf("guard file rule %q requires repository-relative allowed directories", r.ID)
		}
	}
	return nil
}

func validExtension(extension string) bool {
	return strings.HasPrefix(extension, ".") && len(extension) > 1 && path.Ext(extension) == extension && !strings.ContainsAny(extension, "/\\*?[] \t\n")
}

func validDirectory(directory string) bool {
	if slices.Contains([]string{"", ".", ".."}, directory) || strings.ContainsAny(directory, "\\*?[]") {
		return false
	}
	return !path.IsAbs(directory) && path.Clean(directory) == directory && !strings.HasPrefix(directory, "../")
}
