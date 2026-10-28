package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type source struct {
	ID              string          `json:"id"`
	Path            string          `json:"path"`
	Content         string          `json:"content,omitempty"`
	ContentIdentity json.RawMessage `json:"content_identity,omitempty"`
}

// attachSources reads only files contained in the explicit source root.
// gstyle-allow(side_effect_density): Source enrichment coordinates bounded file reads; the dependency summary includes that required IO.
func attachSources(data []byte, root string) ([]byte, error) {
	var docs []map[string]json.RawMessage
	if err := json.Unmarshal(data, &docs); err != nil {
		return nil, err
	}
	for _, doc := range docs {
		sources, err := readSources(doc["sources"], root)
		if err != nil {
			return nil, err
		}
		doc["sources"] = sources
	}
	return json.Marshal(docs)
}

// gstyle-allow(side_effect_density): This source-loading boundary reads each listed file through the confined root.
func readSources(raw json.RawMessage, root string) (json.RawMessage, error) {
	var sources []source
	if err := json.Unmarshal(raw, &sources); err != nil {
		return nil, fmt.Errorf("decode sources: %w", err)
	}
	for i := range sources {
		content, err := readSource(root, sources[i].Path)
		if err != nil {
			return nil, err
		}
		if err := verifyIdentity(sources[i], content); err != nil {
			return nil, err
		}
		sources[i].Content = string(content)
	}
	return json.Marshal(sources)
}

// gstyle-allow(side_effect_density): This file boundary uses os.Root to prevent path and symlink escapes; file IO is its purpose.
func readSource(root, path string) ([]byte, error) {
	base, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	rel, err := sourceRelative(base, path)
	if err != nil {
		return nil, err
	}
	dir, err := os.OpenRoot(base)
	if err != nil {
		return nil, fmt.Errorf("open source root: %w", err)
	}
	defer dir.Close()
	content, err := dir.ReadFile(rel)
	if err != nil {
		return nil, fmt.Errorf("read source %q: %w", path, err)
	}
	return content, nil
}

func sourceRelative(root, path string) (string, error) {
	if !filepath.IsAbs(path) {
		return path, nil
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", err
	}
	if !filepath.IsLocal(rel) {
		return "", fmt.Errorf("source %q is outside root", path)
	}
	return rel, nil
}

func verifyIdentity(s source, content []byte) error {
	if len(s.ContentIdentity) == 0 {
		return nil
	}
	var identity struct {
		Status string `json:"status"`
		Value  string `json:"value"`
	}
	if err := json.Unmarshal(s.ContentIdentity, &identity); err != nil {
		return err
	}
	if identity.Status != "known" {
		return nil
	}
	if identity.Value != fmt.Sprintf("%x", sha256.Sum256(content)) {
		return fmt.Errorf("source %q changed since export; regenerate the PDG", s.Path)
	}
	return nil
}
