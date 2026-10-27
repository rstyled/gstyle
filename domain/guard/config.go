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
	Mode             string   `toml:"mode"`
	Names            []string `toml:"names"`
	Extensions       []string `toml:"extensions"`
	AllowPaths       []string `toml:"allow_paths"`
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
	if len(r.Extensions) == 0 && len(r.Names) == 0 {
		return fmt.Errorf("guard file rule %q requires extensions or names", r.ID)
	}
	if !slices.Contains([]string{"", "added"}, r.Mode) {
		return fmt.Errorf("guard file rule %q mode must be added or omitted", r.ID)
	}
	for _, validate := range []func() error{r.validateMatches, r.validateAllowances} {
		if err := validate(); err != nil {
			return err
		}
	}
	return nil
}

func (r FileRule) validateMatches() error {
	for _, extension := range r.Extensions {
		if !validExtension(extension) {
			return fmt.Errorf("guard file rule %q has invalid extension %q", r.ID, extension)
		}
	}
	for _, name := range r.Names {
		if !validName(name) {
			return fmt.Errorf("guard file rule %q has invalid name %q", r.ID, name)
		}
	}
	return nil
}

func (r FileRule) validateAllowances() error {
	for _, paths := range [][]string{r.AllowDirectories, r.AllowPaths} {
		for _, name := range paths {
			if !ValidRelativePath(name) {
				return fmt.Errorf("guard file rule %q requires repository-relative allowed directories and paths", r.ID)
			}
		}
	}
	return nil
}

func (c Config) HasAddedFileRules() bool {
	return slices.ContainsFunc(c.Files, func(rule FileRule) bool { return rule.Mode == "added" })
}

func validName(name string) bool {
	return ValidRelativePath(name) && path.Base(name) == name
}

func validExtension(extension string) bool {
	return strings.HasPrefix(extension, ".") && len(extension) > 1 && path.Ext(extension) == extension && !strings.ContainsAny(extension, "/\\*?[] \t\n")
}

func ValidRelativePath(directory string) bool {
	if slices.Contains([]string{"", ".", ".."}, directory) || strings.ContainsAny(directory, "\\*?[]") {
		return false
	}
	return !path.IsAbs(directory) && path.Clean(directory) == directory && !strings.HasPrefix(directory, "../")
}
