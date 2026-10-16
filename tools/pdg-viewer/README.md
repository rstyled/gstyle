# PDG dependency instrument

A Go command builds a standalone HTML page from gstyle's `pdg-json` export,
or serves a repository explorer. The function view is a large layered SVG graph:
rectangular statement/control nodes, oval expression/input nodes, source snippets,
line numbers, and directed data/control/execution edges. Pan, zoom, fit, full screen,
and linked source selection support inspection. No JavaScript packages or CDN are
needed. The export has no AST parent/child edges; the renderer does not invent them.

From the repository root:

```sh
go run ./cmd/gstyle patterns --format pdg-json ./domain/pdg > /tmp/pdg.json
go run ./tools/pdg-viewer -source-root "$PWD" < /tmp/pdg.json > /tmp/pdg.html
xdg-open /tmp/pdg.html
```

Replace `./domain/pdg` with the packages to inspect, or `.` for the codebase.
Without `-source-root`, the page still shows graphs and facts. With it, the command
includes source text in the HTML. Share that file only where you intend to share
that code. Paths and symlinks cannot escape the source root.

- Search by function name or source file.
- Select a socket or a node in the list. Click source code to select its node.
- Inspect incoming and outgoing edges. Click an edge entry to visit its other node.
- Switch between direct neighbors, upstream dependencies, and downstream impact.
  Traces use enabled edge types, follow cycles safely, and stay within a function.
- Enable execution edges to inspect order constraints. Data and control start enabled.
- Drag to pan, scroll to zoom, or use the zoom and Fit graph buttons. Full screen expands the graph.
- Selecting code zooms to that operation and highlights its direct dependencies.
- Switch between paper and dark themes.
- Read exact node facts and extraction coverage in the expandable panels.

The whole function is visible by default. Focus trace restricts the display to
selected neighbors or a transitive slice. Cycles are collapsed only for layout;
feedback edges and all original nodes remain visible. Layers are not execution
steps. Source positions use UTF-8 byte offsets, including Unicode. Known source
hashes must match; changed source files require a fresh export.

## Checks

```sh
go test ./tools/pdg-viewer
go run ./cmd/gstyle check --format json ./tools/pdg-viewer
go run github.com/fzipp/gocyclo/cmd/gocyclo -over 6 -ignore '_test.go' tools/pdg-viewer
go run github.com/uudashr/gocognit/cmd/gocognit -over 15 -ignore '_test.go' tools/pdg-viewer
```

File/stream boundaries have explicit `gstyle-allow(side_effect_density)` reasons.
The default policy counts required file IO and propagated helper effects; splitting
these functions just to reduce that count would obscure the boundary. The renderer
also records why embedded immutable assets and JSON escaping are deliberate.
No structural, mutation, or complexity rule is disabled.

The shared `graph-layout.js` uses an iterative strongly-connected-component pass
and a layered layout. `bun test tools/pdg-viewer/graph-layout.test.ts` checks cycles
and a 12,000-node chain. Graph JSON uses Go's HTML-safe JSON encoder.

## Explore a repository

```sh
go run ./tools/pdg-viewer -repo /path/to/go/repo -port 8090
# Open http://127.0.0.1:8090 in Chrome.
```

The atlas scans production Go syntax once and provides package imports, global
search, file locations, source, and possible callers/callees. It excludes tests,
vendor, hidden directories, and node_modules. Function and package lists cap
visible matches at 100; refine the search to narrow them. The package map defaults to the selected package and its import neighbors.
Whole repo shows every indexed package; both modes support pan and zoom. Source is read on selection, not copied into the full index.
The server binds only localhost and accepts source requests only for indexed IDs.

PDGs load on request, one package at a time. There is no whole-repo PDG export on
startup. The Go toolchain must be able to load the selected package and its module
dependencies. Build tags may exclude indexed declarations; such failures appear
in the detail pane. Restart the explorer after editing files: source hash checks
reject stale locations. Syntax call candidates do not prove receiver dispatch or
account for shadowed names, build variants, or runtime targets. These limitations
are stated in the inspector. This is not yet a whole-program dependence graph.

Scale check: `go test ./tools/pdg-viewer -run '^$' -bench BenchmarkLargeRepositoryIndex -benchtime=1x -benchmem` creates a synthetic 401,468-line, 20,040-function repository. One measured scan on this machine takes about 0.22 seconds and allocates about 113 MB in total. This measures syntax indexing, not peak memory, Prometheus, or typed PDG extraction.

## Whole-program overview

The default canvas contains every indexed production function, including functions
with no recorded callers or callees. Every recorded call has a visible endpoint.
Unresolved calls use beige nodes scoped to their calling package; repeated call
pairs carry a count instead of duplicate edges. No package selection or search
removes nodes from this overview. Search highlights declarations; selecting one
opens its source and moves the camera. Fit all restores the entire overview.

Canvas rendering keeps large maps out of the DOM. Labels appear when zoomed in;
viewport clipping reduces drawing work without removing graph data. SCC layout
retains cycles and self calls. Run `bun test tools/pdg-viewer/program-graph.test.ts`
to check disconnected functions, unresolved endpoints, repeated calls, and loops.

This overview is a syntax call graph of repository-owned production declarations,
not a whole-program PDG. Tests, dependencies, anonymous function identities,
receiver dispatch, and dynamic call resolution are not fully indexed. Per-function
typed PDGs remain available from the inspector.

## Executable entrypoints

The index marks receiver-free, non-generic `func main()` declarations with no
parameters or results in `package main`. It separately marks valid `init()`
declarations. A `main` method or a library function with that name is not an
entrypoint. The chooser defaults to a root `main.go`, then `cmd/<module-name>/main.go`,
then the first indexed executable declaration. Fixture paths are labeled.

The selected `main.main` starts the layout. Calls expand outward by their recorded
distance from it; cycles remain present. `init()` functions in the executable's
transitive internal imports are included as startup roots in a separate lane.
Variable initialization and active build-tag selection are not modeled. The right
area retains every declaration not reached by those recorded links; it is not a
dead-code claim. Unknown callees have separate boundary areas. Repositories without
a `main.main` declaration stay in library mode without inventing an entrypoint.

## Edge visibility and clutter

The default view draws individual calls only to and from the selected function.
Selecting `main.main` does not highlight every reachable edge. At overview zoom,
cross-package calls are bundled into one connection per package pair. Package
bands group declarations within call-depth rows. The full function and call data
remain loaded. **All call edges** restores every individual link with a faint
background stroke; selected calls remain distinct. The tests verify that this
presentation change preserves node counts, call counts, and the underlying edges.
