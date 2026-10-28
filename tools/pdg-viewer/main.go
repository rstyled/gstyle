// Command pdg-viewer renders gstyle PDG exports as a standalone page, or serves a repository atlas.
package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
	"strings"
)

//go:embed app.html
var appPage string

//go:embed graph-layout.js
var layoutScript string

//go:embed model.js
var modelScript string

//go:embed app.js
var appScript string

type node struct {
	ID string `json:"id"`
}
type edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind"`
}
type graph struct {
	Sources []source `json:"sources"`
	Nodes   []node   `json:"nodes"`
	Edges   []edge   `json:"edges"`
}

// gstyle-allow(side_effect_density): The command boundary reads flags and standard streams, logs errors, and sets the process exit code.
func main() {
	root := flag.String("source-root", "", "include source files from this directory")
	repo := flag.String("repo", "", "serve a codebase explorer for this Go repository")
	port := flag.Int("port", 8090, "localhost explorer port")
	flag.Parse()
	if err := execute(*repo, *port, *root); err != nil {
		slog.Error("PDG viewer failed", "error", err)
		os.Exit(1)
	}
}

// gstyle-allow(side_effect_density): This stream boundary reads a bounded export and writes its HTML document.
func run(input io.Reader, output io.Writer, root string) error {
	data, err := io.ReadAll(io.LimitReader(input, 32<<20+1))
	if err != nil {
		return fmt.Errorf("read export: %w", err)
	}
	if len(data) > 32<<20 {
		return fmt.Errorf("export exceeds 32 MiB")
	}
	document, err := buildDocument(data, root)
	if err != nil {
		return err
	}
	_, err = io.WriteString(output, document)
	return err
}

// page assembles the app. Go's JSON encoder escapes "<", so data cannot close its script.
// gstyle-allow(side_effect_density): Embedded assets are immutable inputs.
func page(data []byte) string {
	scripts := layoutScript + "\n" + modelScript + "\n" + appScript
	html := strings.Replace(appPage, "/* SCRIPTS */", scripts, 1)
	return strings.Replace(html, "__PDG_JSON__", string(data), 1)
}

// prepare validates a PDG array and optionally attaches source text from root.
// gstyle-allow(side_effect_density): This export boundary validates JSON and optionally reads confined source files; JSON encoding and IO are deliberate.
func prepare(data []byte, root string) ([]byte, error) {
	var docs []json.RawMessage
	if err := json.Unmarshal(data, &docs); err != nil {
		return nil, fmt.Errorf("decode PDG array: %w", err)
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("PDG array is empty")
	}
	if err := validateDocuments(docs); err != nil {
		return nil, err
	}
	if root != "" {
		return attachSources(data, root)
	}
	return json.Marshal(docs)
}

func validateDocuments(docs []json.RawMessage) error {
	for i, doc := range docs {
		var g graph
		if err := json.Unmarshal(doc, &g); err != nil {
			return fmt.Errorf("graph %d: %w", i, err)
		}
		if err := validateGraph(g); err != nil {
			return fmt.Errorf("graph %d: %w", i, err)
		}
	}
	return nil
}

func validateGraph(g graph) error {
	if g.Edges == nil {
		return fmt.Errorf("missing edges array")
	}
	if g.Nodes == nil {
		return fmt.Errorf("missing nodes array")
	}
	ids := make(map[string]bool, len(g.Nodes))
	for _, n := range g.Nodes {
		if n.ID == "" || ids[n.ID] {
			return fmt.Errorf("empty or duplicate node ID %q", n.ID)
		}
		ids[n.ID] = true
	}
	return validateEdges(g.Edges, ids)
}

func validateEdges(edges []edge, ids map[string]bool) error {
	for _, e := range edges {
		if !ids[e.Source] || !ids[e.Target] {
			return fmt.Errorf("edge has missing endpoint: %s → %s", e.Source, e.Target)
		}
		if !slices.Contains([]string{"data", "control", "execution"}, e.Kind) {
			return fmt.Errorf("unknown edge kind %q", e.Kind)
		}
	}
	return nil
}

// gstyle-allow(side_effect_density): This command coordinator optionally loads source text; required file IO propagates through helper summaries.
func buildDocument(data []byte, root string) (string, error) {
	safe, err := prepare(data, root)
	if err != nil {
		return "", err
	}
	return page(safe), nil
}

// gstyle-allow(side_effect_density): Selects the authorized command IO boundary: repository server or standalone export renderer.
func execute(repo string, port int, root string) error {
	if repo != "" {
		return serveRepository(repo, port)
	}
	return run(os.Stdin, os.Stdout, root)
}
