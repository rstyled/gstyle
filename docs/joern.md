# Joern CPG input

Use `--cpg` to inspect a Joern full GraphSON export. Gstyle reads the export
directly; Joern and Java are not runtime dependencies of the importer.

Create the graph with Joern's C/C++ frontend, then export all layers:

```sh
c2cpg --enable-file-content -o cpg.bin src
joern-export --repr=all --format=graphson --out cpg-export cpg.bin
gstyle --cpg cpg-export
gstyle search --cpg --format text cpg-export
gstyle patterns --cpg --format json cpg-export
gstyle patterns --cpg --format pdg-json cpg-export > pdg.json
```

The C/C++ frontend may be installed as `c2cpg.sh` or under
`frontends/c2cpg/bin/c2cpg`, depending on the Joern distribution. Use
`--compilation-database build/compile_commands.json` on that frontend when
compiler options are needed. `joern-export` creates its output directory and
adds missing default/dataflow overlays. Its output directory must not exist.
These commands are verified with Joern 4.0.667.

Pass either the export directory or its `export.json` file. One graph is
accepted per invocation. The importer accepts Joern's `tinker:graph` GraphSON
envelope, typed integer IDs, and scalar or singleton-list properties. Metadata
`VERSION` may identify schema 1.1 or a Joern 4.x producer. Other export dialects,
such as per-method DOT files and the native `cpg.bin`, are not input formats.

## Complexity policy

The v1 policy scores each non-external method separately, including lambda
methods. Synthetic `<global>` methods are excluded. Scores describe the source
constructs present in the CPG, including unreachable source constructs.

Cyclomatic complexity starts at one. Each `if`, loop, catch, non-default case,
ternary, and built-in logical operator adds one. Switches have no separate
cyclomatic increment.

Cognitive complexity uses source AST nesting. Conditions, loops, switches,
catches, and ternaries add one plus nesting. `else if` and `else` add one without
a nesting penalty. A switch adds one structural increment for all its cases.
Each sequence of like logical operators adds one; changing the operator starts
a new sequence. Each member of a resolved recursive call cycle adds one.
`goto` adds one. Ordinary returns, breaks, and continues add none.

This follows the common rules in the
[Cognitive Complexity white paper](https://www.sonarsource.com/docs/CognitiveComplexity.pdf),
with an explicit method-local policy for lambda bodies. It does not claim exact
Sonar C++ parity. Preprocessor branches, source constructs omitted by the
frontend, unresolved calls, and alternate language frontend conventions can
change the score. The checked fixture covers the C/C++ frontend; other language
frontends need their own source fixtures before claiming comparable scores.

Embedded FILE content supplies source snapshots. Method offsets select the
method body, including lambda source ranges. Without embedded content, method
CODE remains available for display, byte spans remain unavailable, and source
locations stay in the `gstyle.joern:properties` extension. The importer never
reads source paths from the graph on the local filesystem.

## Dependency conversion

Each source method becomes a canonical `domain/pdg.Graph` with the
`gstyle.joern.import/1` profile. Local declarations and parameters retain symbol
identity through CPG `REF` edges. Direct scalar assignments retain write and
definition facts. Operator calls become computation or assignment nodes;
ordinary calls stay call nodes.

`REACHING_DEF` becomes a data edge and `CDG` becomes a control edge. Parallel
edges and original edge IDs survive conversion. Reaching-definition variable
names remain in `gstyle.joern:variable` annotations, including the current
exporter's generic `EdgeProperty` field. Expression operands feed their
consumers through separate `value-operand` data edges. A simple assignment's
left identifier is a destination, not an input value.

CFG successors remain in `gstyle.joern:cfg`; they do not become gstyle's
must-precede execution dependencies. Cross-method dependencies remain in
`gstyle.joern:cross-method-dependencies`. AST containment does not become
control dependence.

Imported graphs have `not_evaluated` source conformance and explicit partial or
unknown capability evidence. Thin slicing runs on the imported dependencies;
pattern reports expose that partial coverage. Native Go matching certification,
Rust/Go-specific normalization, automatic source fixes, typed mutation/effect
checks, and DDD checks are not enabled. TUI and search show typed quality as
unavailable. `--config`, Go build tags, and Go mining cache/threshold overrides
do not apply to CPG input.

## Fixture regeneration

`adapters/joern/testdata/sample.cpp` is the source for the checked full export.
Generate it with the commands above, using that file as the frontend input.
The checked JSON is minified, with META_DATA.ROOT changed to `.`; all nodes,
edges, IDs, properties, source content, and producer version are preserved.
Normal tests read the checked export and require no Joern installation.
