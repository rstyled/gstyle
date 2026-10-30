package qualitycheck

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/rstyled/gstyle/adapters/cpp"
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/quality"
)

type configFile struct {
	quality.Config
	Overrides []toml.Primitive `toml:"overrides"`
	Patterns  patterns.Config  `toml:"patterns"`
}

var repositoryKeys = []string{"guard", "risk", "prefixes", "grouping", "overrides", "patterns"}

func LoadConfig(path string) (quality.Config, error) {
	return LoadConfigWithDefaults(path, quality.DefaultConfig())
}

func LoadAnalysisConfig(path string, cpg bool) (quality.Config, error) {
	if cpg {
		return LoadConfigWithDefaults(path, cpp.DefaultQualityConfig())
	}
	return LoadConfig(path)
}

func LoadConfigWithDefaults(path string, config quality.Config) (quality.Config, error) {
	file, err := loadConfigFile(path, config)
	return file.Config, err
}

func LoadPatternConfig(path string) (patterns.Config, error) {
	file, err := loadConfigFile(path, quality.DefaultConfig())
	return file.Patterns, err
}

func loadConfigFile(path string, config quality.Config) (configFile, error) {
	file := configFile{Config: config, Patterns: patterns.DefaultConfig()}
	if path == "" {
		return file, nil
	}
	source, err := os.ReadFile(path)
	if err != nil {
		return file, fmt.Errorf("read quality config: %w", err)
	}
	// Decode into a config without prefixes so user prefixes extend the
	// defaults instead of replacing them.
	defaults := config.Prefixes
	config.Prefixes = nil
	file.Config = config
	metadata, err := toml.Decode(string(source), &file)
	if err != nil {
		return file, fmt.Errorf("decode quality config: %w", err)
	}
	config = file.Config
	config.Prefixes = append(defaults, config.Prefixes...)
	if config.Scopes, err = decodeOverrides(metadata, file.Overrides, config); err != nil {
		return file, err
	}
	if err := rejectUndecoded(metadata); err != nil {
		return file, err
	}
	file.Config = config
	return file, file.validate()
}

func (file configFile) validate() error {
	if err := file.Config.Validate(); err != nil {
		return err
	}
	return file.Patterns.Validate()
}

type override struct {
	settings toml.Primitive
	paths    []string
}

func (o override) covers(scope string) bool {
	return slices.ContainsFunc(o.paths, func(path string) bool { return quality.Covers(path, scope) })
}

func decodeOverrides(metadata toml.MetaData, entries []toml.Primitive, base quality.Config) ([]quality.Scope, error) {
	if err := rejectRepositoryOverrides(metadata); err != nil {
		return nil, err
	}
	overrides, err := readOverrides(metadata, entries)
	if err != nil {
		return nil, err
	}
	var scopes []quality.Scope
	for _, scope := range overrideScopes(overrides) {
		config, err := cascade(metadata, overrides, base, scope)
		if err != nil {
			return nil, err
		}
		scopes = append(scopes, quality.Scope{Path: scope, Config: config})
	}
	return scopes, nil
}

func readOverrides(metadata toml.MetaData, entries []toml.Primitive) ([]override, error) {
	overrides := make([]override, len(entries))
	for index, entry := range entries {
		var scope struct {
			Paths []string `toml:"paths"`
		}
		if err := metadata.PrimitiveDecode(entry, &scope); err != nil {
			return nil, fmt.Errorf("decode quality overrides[%d]: %w", index, err)
		}
		if len(scope.Paths) == 0 {
			return nil, fmt.Errorf("quality overrides[%d] needs paths", index)
		}
		overrides[index] = override{settings: entry, paths: scope.Paths}
	}
	return overrides, nil
}

func overrideScopes(overrides []override) []string {
	var paths []string
	for _, override := range overrides {
		paths = append(paths, override.paths...)
	}
	slices.Sort(paths)
	return slices.Compact(paths)
}

func cascade(metadata toml.MetaData, overrides []override, config quality.Config, scope string) (quality.Config, error) {
	for index, override := range overrides {
		if !override.covers(scope) {
			continue
		}
		if err := metadata.PrimitiveDecode(override.settings, &config); err != nil {
			return config, fmt.Errorf("decode quality overrides[%d]: %w", index, err)
		}
	}
	return config, nil
}

func rejectRepositoryOverrides(metadata toml.MetaData) error {
	for _, key := range metadata.Keys() {
		if len(key) > 1 && key[0] == "overrides" && slices.Contains(repositoryKeys, key[1]) {
			return fmt.Errorf("quality overrides cannot set %s", key[1])
		}
	}
	return nil
}

func rejectUndecoded(metadata toml.MetaData) error {
	keys := metadata.Undecoded()
	if len(keys) == 0 {
		return nil
	}
	names := make([]string, len(keys))
	for index, key := range keys {
		names[index] = key.String()
	}
	return fmt.Errorf("unknown quality config keys: %s", strings.Join(names, ", "))
}
