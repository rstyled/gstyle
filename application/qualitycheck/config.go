package qualitycheck

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/rstyled/gstyle/domain/quality"
)

type configFile struct {
	quality.Config
	Overrides []toml.Primitive `toml:"overrides"`
}

var repositoryKeys = []string{"guard", "risk", "prefixes", "grouping", "overrides"}

func LoadConfig(path string) (quality.Config, error) {
	config := quality.DefaultConfig()
	if path == "" {
		return config, nil
	}
	source, err := os.ReadFile(path)
	if err != nil {
		return config, fmt.Errorf("read quality config: %w", err)
	}
	// Decode into a config without prefixes so user prefixes extend the
	// defaults instead of replacing them.
	defaults := config.Prefixes
	config.Prefixes = nil
	file := configFile{Config: config}
	metadata, err := toml.Decode(string(source), &file)
	if err != nil {
		return config, fmt.Errorf("decode quality config: %w", err)
	}
	config = file.Config
	config.Prefixes = append(defaults, config.Prefixes...)
	if config.Scopes, err = decodeOverrides(metadata, file.Overrides, config); err != nil {
		return config, err
	}
	if err := rejectUndecoded(metadata); err != nil {
		return config, err
	}
	return config, config.Validate()
}

func decodeOverrides(metadata toml.MetaData, entries []toml.Primitive, base quality.Config) ([]quality.Scope, error) {
	if err := rejectRepositoryOverrides(metadata); err != nil {
		return nil, err
	}
	paths, err := overridePaths(metadata, entries)
	if err != nil {
		return nil, err
	}
	var scopes []quality.Scope
	for _, scope := range slices.Compact(slices.Sorted(slices.Values(slices.Concat(paths...)))) {
		config, err := cascade(metadata, entries, paths, base, scope)
		if err != nil {
			return nil, err
		}
		scopes = append(scopes, quality.Scope{Path: scope, Config: config})
	}
	return scopes, nil
}

func overridePaths(metadata toml.MetaData, entries []toml.Primitive) ([][]string, error) {
	paths := make([][]string, len(entries))
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
		paths[index] = scope.Paths
	}
	return paths, nil
}

func cascade(metadata toml.MetaData, entries []toml.Primitive, paths [][]string, config quality.Config, scope string) (quality.Config, error) {
	for index, entry := range entries {
		if !slices.ContainsFunc(paths[index], func(path string) bool { return quality.Covers(path, scope) }) {
			continue
		}
		if err := metadata.PrimitiveDecode(entry, &config); err != nil {
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
