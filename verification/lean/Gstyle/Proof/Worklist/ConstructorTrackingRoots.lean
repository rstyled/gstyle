module
public import Gstyle.Proof.Worklist.ConstructorTracking
public import Gstyle.Proof.Worklist.ConstructorFact
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

/-- These data restrictions leave reads, writes, scalar comparisons, and branch
operators available. They exclude only the sources of tracking roots. -/
structure ScalarTrackingRoots [FfiSyntax] (s : QuerySnapshot) : Prop where
  operandsLength : s.operands.length = s.nodes.length
  operandShape : ∀ row ∈ s.operands, row.arr.length = 2
  operandRefs : ∀ row ∈ s.operands, ∀ child ∈ row.arr, validSymbolRef s child
  noIndex : ∀ ref, validSymbolRef s ref → snapshotFact s ref (go!"index") ≠ go!"true"
  noDereference : ∀ ref, validSymbolRef s ref → snapshotFact s ref (go!"dereference") ≠ go!"true"
  noNil : ∀ ref, validSymbolRef s ref → snapshotFact s ref (go!"nil") ≠ go!"true"
  noCalls : ∀ node ∈ s.nodes, node.Category' ≠ W8 5
  outcomeBound : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46

section execution
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [graphView.TypeAssumptions] [pdg.query.Assumptions]
variable [pdg.Graph.TypeAssumptions] [pdg.Node.TypeAssumptions] [pdg.Attributes.TypeAssumptions]
variable [pdg.AttributeSet.TypeAssumptions] [pdg.AttributeTables.TypeAssumptions]
variable [pdg.Producer.TypeAssumptions] [pdg.Function.TypeAssumptions] [pdg.Span.TypeAssumptions]
variable [pdg.CountFact.TypeAssumptions] [pdg.Conformance.TypeAssumptions] [pdg.Fact.TypeAssumptions]
variable [pdg.Position.TypeAssumptions] [pdg.CallForm.TypeAssumptions] [pdg.ExitKind.TypeAssumptions]
variable [pdg.ConformanceStatus.TypeAssumptions] [pdg.Category.TypeAssumptions]
variable [pdg.Access.TypeAssumptions] [pdg.Tables.TypeAssumptions] [pdg.Extension.TypeAssumptions]
variable [FuncUnfold pdg.listAt [pdg.Ref.ty] (pdg.listAt.impl pdg.Ref.ty)]
variable [FuncUnfold pdg.listAt [pdg.Access.ty] (pdg.listAt.impl pdg.Access.ty)]
variable [FuncUnfold pdg.listAt [pdg.Extension.ty] (pdg.listAt.impl pdg.Extension.ty)]
variable [FuncUnfold extension [] extension.impl]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

/-- Scalar roots execute every production fact and operand lookup. The queue,
visited flags, and tracked bindings remain unchanged. Full receiver ownership is
allowed; no persistent receiver permission is needed. -/
theorem wp_trackRoots_scalar (receiver : Loc) (view : graphView) (s : QuerySnapshot)
    (rows : List (GoArray w32 2)) (row : GoArray w32 2) (ref left right : w32)
    (visited : GoSlice) (queue : Loc) (vd rd : DFrac)
    (hgraph : view.graph' = s.view.graph')
    (hvalid : validSymbolRef s ref)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ rows.length)
    (hrow : rows[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some left) (hright : row.arr[1]? = some right)
    (hlvalid : validSymbolRef s left) (hrvalid : validSymbolRef s right)
    (hindex : snapshotFact s ref (go!"index") ≠ go!"true")
    (hderef : snapshotFact s ref (go!"dereference") ≠ go!"true")
    (hlnil : snapshotFact s left (go!"nil") ≠ go!"true")
    (hrnil : snapshotFact s right (go!"nil") ≠ go!"true")
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ receiver ↦{vd} view ∗
      constructorGraphOwn s ∗ view.operands' ↦*{rd} rows }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"trackRoots" #receiver))
        (Val #ref)) (Val #visited)) (Val #queue))
    {{ RET #(); receiver ↦{vd} view ∗ constructorGraphOwn s ∗ view.operands' ↦*{rd} rows }} := by
  wp_start as ⟨Hview, #Hgraph, Hrows⟩
  try wp_auto
  wp_apply wp_graphView_fact_raw receiver view s ref (go!"index") vd hgraph hvalid halloc
    $$ [Hview Hgraph] with ⟨Hview, Hgraph⟩
  · iframe Hview Hgraph
  try wp_auto
  rw [decide_eq_false hindex]
  try wp_auto
  wp_apply wp_graphView_fact_raw receiver view s ref (go!"dereference") vd hgraph hvalid halloc
    $$ [Hview Hgraph] with ⟨Hview, Hgraph⟩
  · iframe Hview Hgraph
  try wp_auto
  rw [decide_eq_false hderef]
  try wp_auto
  wp_apply wp_operand_readonly receiver view rows ref (W64 0) row left href (by decide) hrow hleft vd rd
    $$ [Hview Hrows] with ⟨Hview, Hrows⟩
  · iframe
  try wp_auto
  wp_apply wp_graphView_fact_raw receiver view s left (go!"nil") vd hgraph hlvalid halloc
    $$ [Hview Hgraph] with ⟨Hview, Hgraph⟩
  · iframe Hview Hgraph
  try wp_auto
  rw [decide_eq_false hlnil]
  try wp_auto
  wp_apply wp_operand_readonly receiver view rows ref (W64 1) row right href (by decide) hrow hright vd rd
    $$ [Hview Hrows] with ⟨Hview, Hrows⟩
  · iframe
  try wp_auto
  wp_apply wp_graphView_fact_raw receiver view s right (go!"nil") vd hgraph hrvalid halloc
    $$ [Hview Hgraph] with ⟨Hview, Hgraph⟩
  · iframe Hview Hgraph
  try wp_auto
  rw [decide_eq_false hrnil]
  try wp_auto
  wp_end

/-- The finite scalar data invariant discharges every operand and fact boundary
needed by the actual root execution. -/
theorem wp_trackRoots_scalar_data (receiver : Loc) (view : graphView) (s : QuerySnapshot)
    (ref : w32) (visited : GoSlice) (queue : Loc) (vd rd : DFrac)
    (hgraph : view.graph' = s.view.graph') (hdata : ScalarTrackingRoots s)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.nodes.length) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ receiver ↦{vd} view ∗
      constructorGraphOwn s ∗ view.operands' ↦*{rd} s.operands }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"trackRoots" #receiver))
        (Val #ref)) (Val #visited)) (Val #queue))
    {{ RET #(); receiver ↦{vd} view ∗ constructorGraphOwn s ∗ view.operands' ↦*{rd} s.operands }} := by
  have hi : (uint.Z ref - 1).toNat < s.operands.length := by rw [hdata.operandsLength]; omega
  let row := s.operands[(uint.Z ref - 1).toNat]'hi
  have hrow : s.operands[(uint.Z ref - 1).toNat]? = some row := List.getElem?_eq_getElem hi
  have hmember := List.mem_of_getElem? hrow
  have hshape := hdata.operandShape row hmember
  have h0 : 0 < row.arr.length := by omega
  have h1 : 1 < row.arr.length := by omega
  let left := row.arr[0]'h0
  let right := row.arr[1]'h1
  have hl : row.arr[0]? = some left := List.getElem?_eq_getElem h0
  have hr : row.arr[1]? = some right := List.getElem?_eq_getElem h1
  have hv : validSymbolRef s ref := Or.inr href
  have hvl := hdata.operandRefs row hmember left (List.mem_of_getElem? hl)
  have hvr := hdata.operandRefs row hmember right (List.mem_of_getElem? hr)
  wp_start_folded as ⟨Hview, Hgraph, Hrows⟩
  wp_apply wp_trackRoots_scalar receiver view s s.operands row ref left right visited queue vd rd
    hgraph hv (by rw [hdata.operandsLength]; exact href) hrow hl hr hvl hvr
    (hdata.noIndex ref hv) (hdata.noDereference ref hv)
    (hdata.noNil left hvl) (hdata.noNil right hvr) hdata.outcomeBound
    $$ [Hview Hgraph Hrows] with ⟨Hview, Hgraph, Hrows⟩
  · iframe
  wp_end

end execution
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
