package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/rstyled/gstyle/adapters/gopatterns"
	"github.com/rstyled/gstyle/adapters/pdgjson"
)

type explorer struct {
	index      *repoIndex
	extraction chan struct{}
}

// gstyle-allow(side_effect_density): This explicit localhost server boundary indexes the selected repo and serves read-only navigation.
func serveRepository(root string, port int) error {
	index, err := indexRepository(root)
	if err != nil {
		return err
	}
	app := &explorer{index: index, extraction: make(chan struct{}, 1)}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", app.home)
	mux.HandleFunc("GET /api/index", app.summary)
	mux.HandleFunc("GET /api/source", app.source)
	mux.HandleFunc("GET /api/pdg", app.detail)
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return err
	}
	slog.Info("code explorer ready", "url", "http://"+listener.Addr().String(), "functions", len(index.Symbols), "lines", index.Lines)
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	return server.Serve(listener)
}

// gstyle-allow(side_effect_density): HTTP HTML response boundary.
func (e *explorer) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprint(w, page([]byte("null")))
}

// gstyle-allow(side_effect_density): HTTP JSON response boundary.
func (e *explorer) summary(w http.ResponseWriter, r *http.Request) { writeJSON(w, e.index) }

// gstyle-allow(side_effect_density): Encodes read-only API responses; network output is its purpose.
func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("encode response", "error", err)
	}
}
func (e *explorer) symbol(id string) (repoSymbol, error) {
	for _, s := range e.index.Symbols {
		if s.ID == id {
			return s, nil
		}
	}
	return repoSymbol{}, fmt.Errorf("unknown symbol %q", id)
}

// gstyle-allow(side_effect_density): This source endpoint returns only an indexed symbol and rejects changed files.
func (e *explorer) source(w http.ResponseWriter, r *http.Request) {
	symbol, err := e.symbol(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	content, err := e.symbolSource(symbol)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, map[string]any{"symbol": symbol, "content": content})
}

// gstyle-allow(side_effect_density): Confined source file reads and snapshot hash validation.
func (e *explorer) symbolSource(s repoSymbol) (string, error) {
	content, err := readSource(e.index.Root, s.File)
	if err != nil {
		return "", err
	}
	identity := source{Path: s.File, ContentIdentity: json.RawMessage(fmt.Sprintf(`{"status":"known","value":%q}`, s.Digest))}
	if err := verifyIdentity(identity, content); err != nil {
		return "", err
	}
	return string(content[s.Start:s.End]), nil
}

// gstyle-allow(side_effect_density): This endpoint serializes typed extraction to bound memory; detailed PDGs are loaded only on demand.
func (e *explorer) detail(w http.ResponseWriter, r *http.Request) {
	symbol, err := e.symbol(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	select {
	case e.extraction <- struct{}{}:
		defer func() { <-e.extraction }()
	default:
		http.Error(w, "A PDG is loading. Retry after it completes.", http.StatusServiceUnavailable)
		return
	}
	data, err := e.symbolPDG(r.Context(), symbol)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// gstyle-allow(side_effect_density): Typed extraction loads the selected package and its dependencies; source hash checks prevent stale navigation.
func (e *explorer) symbolPDG(ctx context.Context, s repoSymbol) ([]byte, error) {
	if _, err := e.symbolSource(s); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	extracted, err := gopatterns.Extract(ctx, e.index.Root, []string{"./" + filepath.ToSlash(filepath.Dir(s.File))})
	if err != nil {
		return nil, err
	}
	for _, f := range extracted.Funcs {
		if filepath.ToSlash(f.Path) == filepath.ToSlash(s.File) && f.Line == s.Line {
			return functionDocument(&f, e.index.Root)
		}
	}
	return nil, fmt.Errorf("function not in active Go build; it may be excluded by build tags")
}

// gstyle-allow(side_effect_density): Encodes a single validated graph with its source text.
func functionDocument(f *gopatterns.FuncPdg, root string) ([]byte, error) {
	var data bytes.Buffer
	data.WriteString("[")
	if err := pdgjson.Write(&data, &f.Pdg); err != nil {
		return nil, err
	}
	data.WriteString("]")
	return prepare(data.Bytes(), root)
}
