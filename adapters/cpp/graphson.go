package cpp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/rstyled/gstyle/domain/pdg"
)

type vertex struct {
	ID       uint64
	Label    string
	Props    map[string]any
	Parent   *vertex
	Children []*vertex
	Owner    *vertex
	Out      map[string][]*vertex
}

type edge struct {
	ID             uint64
	Label          string
	Source, Target *vertex
	Variable       string
	Properties     string
}

type document struct {
	Nodes       []*vertex
	Edges       []edge
	Metadata    *vertex
	Methods     []*vertex
	MethodEdges map[*vertex][]edge
	Files       map[string]*sourceFile
	Sources     *pdg.SourceContext
	Callables   map[string][]*vertex
	Captures    map[string][]*vertex
	Referrers   map[*vertex][]*vertex
	UnboundUses map[*vertex]map[string]bool
	Records     map[string]*cppRecord
	Enums       map[string]bool
}

type wireNode struct {
	ID         json.RawMessage            `json:"id"`
	Label      string                     `json:"label"`
	Properties map[string]json.RawMessage `json:"properties"`
}

type wireEdge struct {
	wireNode
	Source json.RawMessage `json:"outV"`
	Target json.RawMessage `json:"inV"`
}

type wireGraph struct {
	Type  string `json:"@type"`
	Value struct {
		Vertices []wireNode `json:"vertices"`
		Edges    []wireEdge `json:"edges"`
	} `json:"@value"`
}

func Load(ctx context.Context, path string) (*Extraction, error) {
	return LoadFrom(ctx, path, "")
}

func LoadFrom(ctx context.Context, path, sourceRoot string) (*Extraction, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		path = filepath.Join(path, "export.json")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return readFrom(ctx, file, sourceRoot)
}

func Read(ctx context.Context, input io.Reader) (*Extraction, error) {
	return readFrom(ctx, input, "")
}

func readFrom(ctx context.Context, input io.Reader, sourceRoot string) (*Extraction, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(input)
	var wire wireGraph
	if err := decoder.Decode(&wire); err != nil {
		return nil, fmt.Errorf("read C++ GraphSON: %w", err)
	}
	if err := jsonEnd(decoder); err != nil {
		return nil, err
	}
	if !completeGraph(wire) {
		return nil, fmt.Errorf("require full C++ CPG export with --repr=all --format=graphson")
	}
	doc, err := indexWithSources(ctx, wire, sourceRoot)
	if err != nil {
		return nil, err
	}
	return convert(ctx, doc)
}

func indexWithSources(ctx context.Context, wire wireGraph, sourceRoot string) (*document, error) {
	doc, err := indexGraph(ctx, wire)
	if err != nil {
		return nil, err
	}
	if err := doc.supplementSources(sourceRoot); err != nil {
		return nil, err
	}
	return doc, nil
}

func completeGraph(wire wireGraph) bool {
	return wire.Type == "tinker:graph" && wire.Value.Vertices != nil && wire.Value.Edges != nil
}

func jsonEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("C++ export must contain one complete JSON graph: %v", err)
	}
	return nil
}

func value(raw json.RawMessage) (any, error) {
	var parsed any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&parsed); err != nil {
		return nil, err
	}
	return unwrap(parsed)
}

func unwrap(parsed any) (any, error) {
	if wrapper, ok := parsed.(map[string]any); ok {
		inner, exists := wrapper["@value"]
		if !exists {
			return nil, fmt.Errorf("GraphSON property has no @value")
		}
		return unwrap(inner)
	}
	if items, ok := parsed.([]any); ok {
		for i, item := range items {
			decoded, err := unwrap(item)
			if err != nil {
				return nil, err
			}
			items[i] = decoded
		}
	}
	return parsed, nil
}

func identifier(raw json.RawMessage) (uint64, error) {
	if err := identifierType(raw); err != nil {
		return 0, err
	}
	parsed, err := value(raw)
	if err != nil {
		return 0, err
	}
	number, ok := parsed.(json.Number)
	if !ok {
		return 0, fmt.Errorf("GraphSON ID must be an integer")
	}
	id, err := strconv.ParseUint(string(number), 10, 63)
	if err != nil {
		return 0, fmt.Errorf("invalid GraphSON ID: %w", err)
	}
	return id, nil
}

func identifierType(raw json.RawMessage) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil
	}
	var typed struct {
		Type string `json:"@type"`
	}
	if err := json.Unmarshal(trimmed, &typed); err != nil {
		return err
	}
	if typed.Type != "g:Int64" {
		return fmt.Errorf("GraphSON ID requires g:Int64, got %q", typed.Type)
	}
	return nil
}

func properties(raw map[string]json.RawMessage) (map[string]any, error) {
	result := make(map[string]any, len(raw))
	for name, data := range raw {
		decoded, err := value(data)
		if err != nil {
			return nil, fmt.Errorf("property %s: %w", name, err)
		}
		decoded, err = scalarProperty(name, decoded)
		if err != nil {
			return nil, err
		}
		result[name] = decoded
	}
	return result, validateProperties(result)
}

func scalarProperty(name string, decoded any) (any, error) {
	if !scalarNames[name] {
		return decoded, nil
	}
	items, ok := decoded.([]any)
	if !ok {
		return decoded, nil
	}
	if len(items) == 0 {
		return nil, nil
	}
	if len(items) != 1 {
		return nil, fmt.Errorf("%s has multiple scalar values", name)
	}
	return items[0], nil
}

var scalarNames = scalarPropertyNames()

func scalarPropertyNames() map[string]bool {
	result := map[string]bool{"IS_EXTERNAL": true, "IS_VARIADIC": true}
	for _, name := range append(textProperties, numberProperties...) {
		result[name] = true
	}
	return result
}
