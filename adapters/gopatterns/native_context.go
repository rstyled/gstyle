package gopatterns

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/rstyled/gstyle/adapters/cfgjson"
	"github.com/rstyled/gstyle/adapters/pdgjson"
	"github.com/rstyled/gstyle/domain/pdg"
)

// WriteContext streams exact source artifacts and paired schema documents.
// Reports stay outside the repository; no expanded extraction-wide copy exists.
func (n *NativeMatching) WriteContext(w io.Writer) error {
	encoder := json.NewEncoder(w)
	if err := n.writeSourceSnapshots(encoder); err != nil {
		return err
	}
	for i := range n.Graphs {
		graph := &n.Graphs[i]
		source, err := n.contextSource(graph)
		if err != nil {
			return err
		}
		if err := writePairedContext(encoder, graph, source); err != nil {
			return err
		}
	}
	return nil
}

func (n *NativeMatching) contextSource(graph *pdg.Graph) ([]byte, error) {
	path := filepath.Join(n.root, graph.Tables.Text(graph.Sources[0].Path))
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(source))
	if digest != graph.Tables.Text(graph.Sources[0].ContentIdentity.Value) {
		return nil, fmt.Errorf("source changes after compiler snapshot: %s", path)
	}
	return source, nil
}

func (n *NativeMatching) writeSourceSnapshots(encoder *json.Encoder) error {
	paths := make([]string, 0, len(n.SourceSnapshots))
	for path := range n.SourceSnapshots {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	for _, path := range paths {
		source, err := os.ReadFile(filepath.Join(n.root, path))
		if err != nil {
			return err
		}
		if digest := fmt.Sprintf("%x", sha256.Sum256(source)); digest != n.SourceSnapshots[path] {
			return fmt.Errorf("compiler source snapshot changes: %s", path)
		}
		if err := encoder.Encode(map[string]any{"type": "source", "path": path, "content_identity": n.SourceSnapshots[path], "content": string(source)}); err != nil {
			return err
		}
	}
	return nil
}

func writePairedContext(encoder *json.Encoder, graph *pdg.Graph, source []byte) error {
	var document bytes.Buffer
	if err := pdgjson.Write(&document, graph); err != nil {
		return err
	}
	if err := encoder.Encode(map[string]any{"type": "pdg", "function": graph.Tables.Text(graph.Function.ID), "document": json.RawMessage(document.Bytes())}); err != nil {
		return err
	}
	document.Reset()
	if err := cfgjson.Write(&document, graph, source); err != nil {
		return err
	}
	return encoder.Encode(map[string]any{"type": "cfg", "function": graph.Tables.Text(graph.Function.ID), "document": json.RawMessage(document.Bytes())})
}
