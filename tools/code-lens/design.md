# Investigation contract

One finding, one full function export, one linked source pane. The graph is a
navigation surface; the detector witness is a distinct evidence overlay.

All graph nodes and edges remain loaded when the camera, source selection,
filters, proof steps, or backward slice change. Unrelated context dims but does
not disappear. A minimap locates a focused region in the full function. Source
mapping uses original UTF-8 byte offsets; synthetic joins retain their recorded
read span. No AST hierarchy or execution path is inferred from layout. The fixture
uses a witness-derived directed layout: two branch lanes feed separate top ports
on the join, followed by the read and comparison from left to right. Syntax
operands remain visible in a separate support lane. Orthogonal routes avoid
all card interiors; primary edge labels explain their role.

The detector witness comes from `interfacenil.Explain`, which uses the existing
solver, not a parallel implementation. It records concrete seed nodes and the
actual dependencies of successful transfers, then records the matching operand
of each reported comparison. Seed operands do not become proof inputs merely
because they are adjacent in the PDG. Control and order edges remain available
as context and retain their approximate evidence labels.

Independent checks cover AST identity, CFG branch convergence, Go executions,
Find/Explain parity across 44 fixtures, valid proof endpoints, full graph identity
preservation, camera anchor math, and linked browser interactions. Human usability
has not been independently tested; browser checks establish behavior, not ease of
understanding.

The repair experiment declares its contract instead of guessing intent. It uses
a temporary source copy, runs both inputs, and reruns gstyle. The original fixture
is not edited. This keeps observed execution separate from static evidence and
from a proposed behavioral change.
