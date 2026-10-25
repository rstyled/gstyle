package application_test

import (
	"reflect"
	"testing"
)

func TestControlSearchPreservesSharedViewAndReturnsUsableIndexes(t *testing.T) {
	server, _ := startControlTestServer(t)
	before := rpcResult(t, server.URL(), "getState", nil)
	result := rpcResult(t, server.URL(), "search", map[string]any{"query": "SIMPLE", "limit": 1})
	if result["total"] != float64(1) || result["revision"] != before["revision"] || result["refreshing"] != false {
		t.Fatalf("search result = %v", result)
	}
	match := result["matches"].([]any)[0].(map[string]any)
	if match["fileIndex"] != float64(0) || match["functionIndex"] != float64(1) {
		t.Fatalf("search indexes = %v", match)
	}
	after := rpcResult(t, server.URL(), "getState", nil)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("search changes shared state")
	}
	rpcResult(t, server.URL(), "selectFile", map[string]any{"index": match["fileIndex"]})
	state := rpcResult(t, server.URL(), "selectFunction", map[string]any{"index": match["functionIndex"]})
	assertSelection(t, state, "hot.go", "Simple")
}

func TestControlSearchValidatesParametersAndDefaults(t *testing.T) {
	server, _ := startControlTestServer(t)
	result := rpcResult(t, server.URL(), "search", nil)
	if result["limit"] != float64(20) || result["total"] != float64(3) {
		t.Fatalf("defaults = %v", result)
	}
	for _, params := range []any{
		map[string]any{"limit": 0}, map[string]any{"limit": 201}, map[string]any{"offset": -1},
		map[string]any{"minComplexity": -1}, map[string]any{"minCognitiveComplexity": -1},
		map[string]any{"query": 1}, map[string]any{"findingsOnly": "yes"},
		map[string]any{"unknown": true}, []any{},
	} {
		if got := rpcError(t, server.URL(), "search", params)["code"]; got != float64(-32602) {
			t.Fatalf("code = %v", got)
		}
	}
}
