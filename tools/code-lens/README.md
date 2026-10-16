# One finding: graph ↔ source ↔ detector evidence

Open `index.html` offline, or run `bun tools/code-lens/serve.ts` and open
<http://127.0.0.1:8092/>. No package install or CDN is needed.

```sh
bun tools/code-lens/build.ts
bun test tools/code-lens/*.test.ts
go test ./domain/interfacenil ./adapters/gopatterns ./tools/code-lens/...
```

The page opens with a four-step explanation: choose one branch, assign boxed,
follow the possible values through the join, and inspect the nil check. Each
step highlights matching code and focuses the relevant graph region. Manual
node/source selection leaves the walkthrough and opens the full inspector.
**Whole function** restores all context. No exported data is removed.

The main canvas contains all 14 exported operations and all 20 exported edges of
`branchAssignments`. This is a full export of one function, not a whole-program
PDG or a claim that every semantic dependency is known. The directed layout follows the actual two-branch witness: input, branch values,
assignments, join, read, and comparison. Supporting syntax stays in a separate
lane. These positions do not claim execution order or AST parenthood.

The obstacle-aware Manhattan router keeps every edge outside the node cards.
The two definitions enter distinct top ports on the join; its result leaves
from the right. Main edges use descriptive labels; export IDs remain in tooltips
and exact evidence.

- Drag or wheel to move the camera. The minimap locates the viewport in the whole
  graph. Zoom buttons, `+`/`−`, and Home work without a pointer.
- Select a node for its exact source range. Select source to focus the graph;
  repeated selection cycles through operations at that line.
- Select an edge or connection once for evidence, twice to follow its other end.
- **Whole function** fits the entire graph and preserves selection.
- **Trace inputs** highlights a backward data slice without removing context.
- **Walk through this finding** restarts the explanation. The **Detector proof**
  tab highlights the actual detector witness. Select a proof
  step to see its graph operation, source, and the corresponding detector code.
- Edge filters affect visibility only. Every graph identity remains loaded.

## Evidence

`build.ts` checks the fixture against HEAD and verifies its hash against the raw
PDG export. It runs the actual gstyle detector and the `interfacenil.Explain` API.
That API uses the same eligibility, supported dependencies, worklist, and operand
checks as `Find`; the view does not reconstruct a proof from nearby edges. Tests
compare both APIs across all 44 interface-nil fixtures, including interprocedural
cases. This view intentionally rejects witnesses outside the selected function.

The Go helper records exact AST spans and an independently built CFG. It runs an
instrumented copy of the original statements for both boolean inputs. Raw source,
PDG, candidate, solver witness, detector source hash, and Go observations are in
`data/evidence.json` and embedded in the standalone page.

## Repair experiment

The original fixture demonstrates an always-false comparison; it does not define
product intent. The experiment explicitly assumes **disabled means no value**,
replaces only `boxed = "queued"` with `boxed = nil` in a temporary copy, runs both
inputs in Go, and reruns the actual detector. Disabled now compares equal to nil,
enabled still does not, and the typednil finding disappears. The original source
stays intact. This is not presented as a safe automatic fix.

`browser-actions.json` checks the linked workflow in Chromium via Anatomy's
`drive.mjs`. The visual uses the existing paper/green palette. There are no new
runtime or build dependencies.
