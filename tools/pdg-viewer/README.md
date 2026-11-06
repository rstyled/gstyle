# gstyle atlas

Explore a Go module as packages, function calls, and per-function CPG evidence.
Repository graphs and findings come from rstyle; standalone PDG exports still work.
The viewer uses embedded HTML, CSS, and JavaScript. No frontend packages or CDN are required.

The Go route needs current `rstyle` and `rstyle-go`. Tested revisions are recorded
in `rstyle-versions.json`; the release version alone does not identify CPG support.
Install the tested tools (the Go frontend needs GitHub access):

```sh
cargo +nightly-2026-09-30 install --locked \
  --git https://github.com/rstyled/rstyle.git \
  --rev 01dc7915feb99da684f927e9a8fa24882667c933 cargo-rstyle
GOPRIVATE=github.com/rstyled go install \
  github.com/rstyled/rstyle-go/cmd/rstyle-go@39a1f8fca8790541facd077bfcdbfa565c266076
```

The Go route does not need `rstyle-driver`. Use `-rstyle /path/to/rstyle` to choose
the runner; `RSTYLE_GO=/path/to/rstyle-go` chooses its frontend.

```sh
go run ./tools/pdg-viewer -repo /path/to/gstyle -port 8090
# Open http://127.0.0.1:8090
```

The package map groups packages by path. Select a package to trace its imports;
double-click it or use the sidebar to open its function calls. Search accepts
function names and file paths. Large packages start with a public function and its
nearby calls. **All functions** shows every indexed function in that package,
including those with no recorded calls. Cross-package calls aggregate by package;
counts retain repeated call sites.

**All functions** beside the package map opens the entire repository on one map.
Every indexed function has a tile, grouped by package and architectural layer,
including functions with no recorded callers. All resolved internal call links
retain their exact function endpoints and repeated-call counts. **All calls: on/off**
shows or hides all resolved background links. The legend shows the active mode;
selection always highlights direct calls. Search and
package focus move the camera without filtering out functions. Zoom reveals names;
Fit restores the whole map. This view shares the index's production-source scope.
The overview caches function tiles and background wires in bounded raster layers.
A viewport-sized canvas draws sharp close-up tiles and text at display resolution.
A separate canvas draws direct call highlights, so hover and selection reuse the
base map without repainting its labels. Spatial buckets handle hit testing and visible detail;
camera and hover work runs once per animation frame. Hover leaves the graph DOM
intact. Large package sidebars render only visible rows while retaining the full
scroll range and search index. Source highlighting is cached for each function.
A padded canvas surface moves
through the compositor while dragging; it repaints when the pan reaches new
content, then returns to viewport size after release. Rendering does not filter
the underlying graph. Use arrow
keys to select functions when the map has keyboard focus, then Enter to open a PDG.

At startup the server runs `rstyle go --filter check,patterns,flow` once and reads
its completed, sealed plain-JSON capture. Exit 1 means findings, not capture failure.
Tool errors fail startup. The temporary run is removed after the snapshot is loaded.
Open **Dependence graph** to load one function from that snapshot. **Statements** folds nested
expressions into their enclosing statements. **Expressions** retains the CPG operations while folding block and expression-statement
wrappers. Both views preserve the original operation IDs and raw endpoints.
AST, CFG, data dependence and control dependence start enabled; execution dependence
is optional. The layout puts source statements on a spine, syntax below it and
dependencies above it. The spine follows source order, not a claimed runtime order. Select code or a graph node to
inspect incoming and outgoing links; choose Neighbors, Upstream, or Downstream to
highlight a trace.
Derived analysis nodes use short kind labels, not source-code claims.

Drag to pan; use the wheel or +/− to zoom. **F** fits the graph. **/** focuses search.
**Expand canvas** hides the side panels. Graph nodes accept keyboard focus: Enter
selects a node, then Enter opens it. Escape clears selection or goes up one level.
Links retain the current package or function in the URL.

The inspector includes real checks and pattern candidates. Selecting a check highlights
its recorded detail lines and matching operations. A function-wide diagnostic or pattern
candidate highlights the whole function; it is not presented as an exact node match.
Counts on operations identify matching findings, not extra findings. Selecting code or
an operation returns to dependency tracing. Pattern proposals remain separate from
failed checks, and empty results are not a clean-coverage claim.

Flow standing (`known`, `approximated`, `unknown`), reasons and assumptions remain
visible. Approximate data edges are dashed. Function coverage overrides the unit's
coverage. Go library-effect classification is incomplete in this CPG route, so
side-effect density is explicitly not presented as a reliable health result.

## Scope and accuracy

The inspector shows the exact indexed checkout and its package, function, file,
and line counts. The index includes production `.go` declarations in that module.
It excludes tests, testdata, vendor, hidden directories, node_modules, dist, and
nested Go modules. Syntax indexing retains platform-specific declarations even
when the active Go build excludes them. Counts describe indexed source, not runtime
reachability; line counts include comments and blank lines.

Static calls use Go type identities, including concrete methods and generic
function origins. Builtins and type conversions do not create call edges. Interface
and function-value calls remain dynamic. Packages with type errors and declarations
outside the active build have unavailable calls; the viewer never substitutes name
guesses. Static calls written inside closures are attributed to the containing
function. The graph does not prove that a call runs. Tests, external callers, runtime
initialization, and dynamic dispatch can add links. No missing edge or caller count
is an unused-code finding.

Per-function graphs preserve rstyle's analysis limits and source snapshot identity. Statement folding changes
presentation; it does not add AST edges or claim that missing dependencies imply
independence. Extraction evidence remains available in the inspector. Source spans
use UTF-8 byte offsets. Source identity checks reject files changed since indexing.
Restart the server after code changes. The server binds only localhost, confines
source reads to the selected root, and serves only indexed function IDs. The Go
toolchain and rstyle frontend must be able to load the selected module and its dependencies.

## Imported frontend indexes

`-repo` builds a Go index. A C++/Joern backend can serve the same atlas through
`/api/index`, `/api/source?id=...`, and `/api/pdg?id=...`. The frontend retains
every indexed method, including declarations and methods with no resolved calls.

An imported index can supply `language: "cpp"`, `group_label`, `layers`,
`layer_columns`, and hex `layer_colors` to group functions by the source tree's
components. Layer names use lowercase CSS-safe identifiers and include `root`
for other paths. Columns place each layer exactly once; omitted custom columns
use one column per layer. `relationships: "calls"` labels component links as calls. The optional
`scope_note`, `call_resolution_note`, and `analysis_limits` describe the import's
coverage without Go build claims. Symbol `qualified_name` participates in search;
`source_note` records method source limits. Call targets must name exact indexed
IDs. Preserve repeated sites; retain unresolved calls without guessed targets.

The PDG endpoint returns a one-document array with its recorded extraction
limits and embedded source snapshot. If the method's byte range is unverified,
a viewer response may include `display_source` with `content`, `line`, and
`reason`; the source panel displays that evidence without inferred node mapping.
Large imports load these per-function documents on demand, while all function
tiles remain on the repository map.

## Standalone exports

```sh
go run . patterns --format pdg-json ./domain/pdg > /tmp/pdg.json
go run ./tools/pdg-viewer -source-root "$PWD" < /tmp/pdg.json > /tmp/pdg.html
```

Without `-source-root`, the page still shows exported operations and dependencies.
With it, source is copied into the standalone HTML; share it where that code belongs.
The function chooser handles multi-function exports. JSON is HTML-escaped before embedding.

## Verification

```sh
go test ./tools/pdg-viewer
RSTYLE_VIEWER_TEST_BIN="$(command -v rstyle)" go test ./tools/pdg-viewer -run RstyleLive
bun test tools/pdg-viewer/*.test.ts
# The browser regression test runs when Chromium is installed.
make check
make quality-gate
```

Tests cover typed method dispatch, shadowing, generics, dynamic calls, build coverage,
source confinement, stale files, statement folding, UTF-8 positions, graph cycles,
disconnected functions, cross-package counts, and a 12,000-node chain.
The localhost and file/stream boundaries explain their deliberate IO through
`gstyle-allow(side_effect_density)` comments. Complexity rules stay enabled.
