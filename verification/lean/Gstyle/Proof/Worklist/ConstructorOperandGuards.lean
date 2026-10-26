module
public import Gstyle.Proof.Worklist.ConstructorEdges
@[expose] public section
noncomputable section
set_option maxHeartbeats 0
set_option maxRecDepth 100000
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

/-- The production kind, position, label, and node-reference guards decide each write. -/
def constructorEdgeApplies [FfiSyntax] (strings : List GoString) (count : Nat) (edge : pdg.Edge) : Bool := by
  classical
  exact decide (constructorAcceptedEdge strings count edge)

/-- The constructor writes only edges accepted by every production guard. -/
def constructorFilteredOperandSet [FfiSyntax] (strings : List GoString) (count : Nat)
    (rows : List (GoArray w32 2)) (edge : pdg.Edge) : List (GoArray w32 2) :=
  if constructorEdgeApplies strings count edge then constructorOperandSet rows edge else rows

section execution
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [pdg.query.Assumptions]
variable [pdg.Edge.TypeAssumptions] [pdg.EdgeKind.TypeAssumptions] [pdg.Position.TypeAssumptions]
variable [FuncUnfold operandReferences [] operandReferences.impl]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

/-- The production reference guard rejects every zero or out-of-range endpoint. -/
theorem wp_operandReferences_invalid (edge : pdg.Edge) (count : w64)
    (hinvalid : ¬ ((0 < uint.Z edge.Source' ∧ uint.Z edge.Source' ≤ sint.Z count) ∧
      (0 < uint.Z edge.Target' ∧ uint.Z edge.Target' ≤ sint.Z count))) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! operandReferences)) (Val #edge)) (Val #count))
    {{ RET #false; True }} := by
  wp_start
  wp_auto
  by_cases hs : edge.Source' = W32 0
  · rw [decide_eq_true hs]
    wp_end
  rw [decide_eq_false hs]
  wp_auto
  by_cases hsource : sint.Z (W64 (uint.Z edge.Source')) ≤ sint.Z count
  · rw [decide_eq_true hsource]
    wp_auto
    by_cases ht : edge.Target' = W32 0
    · rw [decide_eq_true ht]
      wp_end
    rw [decide_eq_false ht]
    wp_auto
    have htarget : ¬ (sint.Z (W64 (uint.Z edge.Target')) ≤ sint.Z count) := by
      intro ht
      apply hinvalid
      constructor <;> constructor <;> word
    rw [decide_eq_false htarget]
    wp_end
  · rw [decide_eq_false hsource]
    wp_end

/-- The production reference guard computes the exact bounds predicate for any count. -/
theorem wp_operandReferences_result (edge : pdg.Edge) (count : w64) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! operandReferences)) (Val #edge)) (Val #count))
    {{ RET #(decide ((0 < uint.Z edge.Source' ∧ uint.Z edge.Source' ≤ sint.Z count) ∧
      (0 < uint.Z edge.Target' ∧ uint.Z edge.Target' ≤ sint.Z count))); True }} := by
  by_cases hvalid : (0 < uint.Z edge.Source' ∧ uint.Z edge.Source' ≤ sint.Z count) ∧
      (0 < uint.Z edge.Target' ∧ uint.Z edge.Target' ≤ sint.Z count)
  · simpa only [decide_eq_true hvalid] using
      wp_operandReferences_valid edge count hvalid.1 hvalid.2
  · simpa only [decide_eq_false hvalid] using
      wp_operandReferences_invalid edge count hvalid

variable [pdg.Graph.TypeAssumptions] [pdg.Node.TypeAssumptions]
variable [pdg.AttributeSet.TypeAssumptions] [pdg.AttributeTables.TypeAssumptions]
variable [pdg.Producer.TypeAssumptions] [pdg.Function.TypeAssumptions] [pdg.Span.TypeAssumptions]
variable [pdg.NormalFlow.TypeAssumptions] [pdg.CountFact.TypeAssumptions]
variable [pdg.Conformance.TypeAssumptions] [pdg.Fact.TypeAssumptions]
variable [pdg.CallForm.TypeAssumptions] [pdg.ExitKind.TypeAssumptions]
variable [pdg.ConformanceStatus.TypeAssumptions] [pdg.Category.TypeAssumptions]
variable [pdg.Tables.TypeAssumptions] [graphView.TypeAssumptions]
variable [FuncUnfold sourceOperandEdge [] sourceOperandEdge.impl]

omit [FuncUnfold operandReferences [] operandReferences.impl]
  [pdg.Node.TypeAssumptions] [pdg.AttributeSet.TypeAssumptions]
  [pdg.NormalFlow.TypeAssumptions] [pdg.CallForm.TypeAssumptions]
  [pdg.ExitKind.TypeAssumptions] [pdg.Category.TypeAssumptions]
  [graphView.TypeAssumptions] in
/-- The actual source-edge function computes its kind, position, and label guards. -/
theorem wp_sourceOperandEdge_result (receiver : Loc) (graph : pdg.Graph)
    (table : pdg.Tables) (strings : List GoString) (edge : pdg.Edge) (dq : DFrac) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ receiver ↦{dq} graph ∗
      graph.Tables' ↦{dq} table ∗ table.Strings' ↦*{dq} strings }}
      (App (App (Val (@! sourceOperandEdge)) (Val #receiver)) (Val #edge))
    {{ RET #(decide (edge.Kind' = W8 0 ∧
        (0 < uint.Z edge.Position' ∧ uint.Z edge.Position' ≤ 2) ∧
        constructorOperandLabel (pdg.tableText strings edge.Subkind') = true));
      receiver ↦{dq} graph ∗ graph.Tables' ↦{dq} table ∗ table.Strings' ↦*{dq} strings }} := by
  classical
  by_cases hkind : edge.Kind' = W8 0
  · by_cases hposition : 0 < uint.Z edge.Position' ∧ uint.Z edge.Position' ≤ 2
    · have hresult : decide (edge.Kind' = W8 0 ∧
          (0 < uint.Z edge.Position' ∧ uint.Z edge.Position' ≤ 2) ∧
          constructorOperandLabel (pdg.tableText strings edge.Subkind') = true) =
          constructorOperandLabel (pdg.tableText strings edge.Subkind') := by
        cases hlabel : constructorOperandLabel (pdg.tableText strings edge.Subkind') <;>
          simp only [hkind, hposition, Bool.false_eq_true, _root_.and_false,
            _root_.and_self, decide_false, decide_true]
      rw [hresult]
      exact wp_sourceOperandEdge_position receiver graph table strings edge dq hkind hposition
    · have hrejected : ¬ (edge.Kind' = W8 0 ∧
          (0 < uint.Z edge.Position' ∧ uint.Z edge.Position' ≤ 2) ∧
          constructorOperandLabel (pdg.tableText strings edge.Subkind') = true) :=
        fun h => hposition h.2.1
      rw [decide_eq_false hrejected]
      wp_start as ⟨Hgraph, Htable, Hstrings⟩
      wp_auto
      rw [decide_eq_true hkind]
      wp_auto
      by_cases hzero : edge.Position' = W32 0
      · rw [decide_eq_true hzero]
        wp_end
      rw [decide_eq_false hzero]
      wp_auto
      have hout : uint.Z edge.Position' > uint.Z (W32 2) := by word
      rw [decide_eq_true hout]
      wp_end
  · have hrejected : ¬ (edge.Kind' = W8 0 ∧
        (0 < uint.Z edge.Position' ∧ uint.Z edge.Position' ≤ 2) ∧
        constructorOperandLabel (pdg.tableText strings edge.Subkind') = true) :=
      fun h => hkind h.1
    rw [decide_eq_false hrejected]
    wp_start as ⟨Hgraph, Htable, Hstrings⟩
    wp_auto
    rw [decide_eq_false hkind]
    wp_end

/-- Row validity supplies the actual slot needed by every accepted production write. -/
theorem wp_operandEdge_accepted (receiver : Loc) (view : graphView) (graph : pdg.Graph)
    (table : pdg.Tables) (strings : List GoString) (nodes : List pdg.Node)
    (rows : List (GoArray w32 2)) (edge : pdg.Edge) (vd gd : DFrac)
    (hrows : constructorRowsValid nodes.length rows)
    (haccepted : constructorAcceptedEdge strings nodes.length edge) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ receiver ↦{vd} view ∗ view.graph' ↦{gd} graph ∗
      graph.Nodes' ↦*{gd} nodes ∗ graph.Tables' ↦{gd} table ∗ table.Strings' ↦*{gd} strings ∗
      view.operands' ↦* rows }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"operandEdge" #receiver)) (Val #edge))
    {{ RET #(); receiver ↦{vd} view ∗ view.graph' ↦{gd} graph ∗
      graph.Nodes' ↦*{gd} nodes ∗ graph.Tables' ↦{gd} table ∗ table.Strings' ↦*{gd} strings ∗
      view.operands' ↦* constructorOperandSet rows edge }} := by
  rcases haccepted with ⟨hkind, hposition, hlabel, hsource, htarget⟩
  have hi : (uint.Z edge.Target' - 1).toNat < rows.length := by
    obtain ⟨hlen, _⟩ := hrows
    omega
  obtain ⟨row, hrow⟩ : ∃ row, rows[(uint.Z edge.Target' - 1).toNat]? = some row :=
    ⟨_, List.getElem?_eq_getElem hi⟩
  have harity := (hrows.2 row (List.mem_of_getElem? hrow)).1
  simpa only [constructorOperandSet, hrow, Option.getD_some] using
    wp_operandEdge_write receiver view graph table strings nodes rows row edge vd gd
      hkind hposition hlabel hsource htarget hrows.1 hrow harity

omit [pdg.Node.TypeAssumptions] [pdg.AttributeSet.TypeAssumptions]
  [pdg.NormalFlow.TypeAssumptions] [pdg.CallForm.TypeAssumptions]
  [pdg.ExitKind.TypeAssumptions] [pdg.Category.TypeAssumptions] in
/-- Invalid node references leave every operand row unchanged in the actual method. -/
theorem wp_operandEdge_invalid_references (receiver : Loc) (view : graphView) (graph : pdg.Graph)
    (table : pdg.Tables) (strings : List GoString) (nodes : List pdg.Node)
    (rows : List (GoArray w32 2)) (edge : pdg.Edge) (vd gd : DFrac)
    (hkind : edge.Kind' = W8 0)
    (hposition : 0 < uint.Z edge.Position' ∧ uint.Z edge.Position' ≤ 2)
    (hlabel : constructorOperandLabel (pdg.tableText strings edge.Subkind') = true)
    (hinvalid : ¬ ((0 < uint.Z edge.Source' ∧ uint.Z edge.Source' ≤ nodes.length) ∧
      (0 < uint.Z edge.Target' ∧ uint.Z edge.Target' ≤ nodes.length))) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ receiver ↦{vd} view ∗ view.graph' ↦{gd} graph ∗
      graph.Nodes' ↦*{gd} nodes ∗ graph.Tables' ↦{gd} table ∗ table.Strings' ↦*{gd} strings ∗
      view.operands' ↦* rows }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"operandEdge" #receiver)) (Val #edge))
    {{ RET #(); receiver ↦{vd} view ∗ view.graph' ↦{gd} graph ∗
      graph.Nodes' ↦*{gd} nodes ∗ graph.Tables' ↦{gd} table ∗ table.Strings' ↦*{gd} strings ∗
      view.operands' ↦* rows }} := by
  wp_start as ⟨Hview, Hgraph, Hnodes, Htable, Hstrings, Hrows⟩
  ihave %Hnlen := ownSlice_len $$ Hnodes
  wp_auto
  wp_apply wp_sourceOperandEdge_position view.graph' graph table strings edge gd hkind hposition
    $$ [Hgraph Htable Hstrings] with ⟨Hgraph, Htable, Hstrings⟩
  · iframe
  rw [hlabel]
  wp_auto
  wp_apply wp_operandReferences_invalid edge graph.Nodes'.len (by
    intro h
    apply hinvalid
    obtain ⟨⟨hs, hsb⟩, ⟨ht, htb⟩⟩ := h
    constructor <;> constructor <;> word)
  wp_end

omit [pdg.Node.TypeAssumptions] [pdg.AttributeSet.TypeAssumptions]
  [pdg.NormalFlow.TypeAssumptions] [pdg.CallForm.TypeAssumptions]
  [pdg.ExitKind.TypeAssumptions] [pdg.Category.TypeAssumptions] in
/-- Every failed production guard is a no-op, including short-circuit guard failures. -/
theorem wp_operandEdge_rejected (receiver : Loc) (view : graphView) (graph : pdg.Graph)
    (table : pdg.Tables) (strings : List GoString) (nodes : List pdg.Node)
    (rows : List (GoArray w32 2)) (edge : pdg.Edge) (vd gd : DFrac)
    (hrejected : ¬ constructorAcceptedEdge strings nodes.length edge) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ receiver ↦{vd} view ∗ view.graph' ↦{gd} graph ∗
      graph.Nodes' ↦*{gd} nodes ∗ graph.Tables' ↦{gd} table ∗ table.Strings' ↦*{gd} strings ∗
      view.operands' ↦* rows }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"operandEdge" #receiver)) (Val #edge))
    {{ RET #(); receiver ↦{vd} view ∗ view.graph' ↦{gd} graph ∗
      graph.Nodes' ↦*{gd} nodes ∗ graph.Tables' ↦{gd} table ∗ table.Strings' ↦*{gd} strings ∗
      view.operands' ↦* rows }} := by
  classical
  by_cases hsource : edge.Kind' = W8 0 ∧
      (0 < uint.Z edge.Position' ∧ uint.Z edge.Position' ≤ 2) ∧
      constructorOperandLabel (pdg.tableText strings edge.Subkind') = true
  · apply wp_operandEdge_invalid_references receiver view graph table strings nodes rows edge vd gd
      hsource.1 hsource.2.1 hsource.2.2
    intro hrefs
    exact hrejected ⟨hsource.1, hsource.2.1, hsource.2.2, hrefs.1, hrefs.2⟩
  · wp_start as ⟨Hview, Hgraph, Hnodes, Htable, Hstrings, Hrows⟩
    wp_auto
    wp_apply wp_sourceOperandEdge_result view.graph' graph table strings edge gd
      $$ [Hgraph Htable Hstrings] with ⟨Hgraph, Htable, Hstrings⟩
    · iframe
    rw [decide_eq_false hsource]
    wp_end

/-- The actual method implements the filtered pure update for every arbitrary edge. -/
theorem wp_operandEdge_filtered (receiver : Loc) (view : graphView) (graph : pdg.Graph)
    (table : pdg.Tables) (strings : List GoString) (nodes : List pdg.Node)
    (rows : List (GoArray w32 2)) (edge : pdg.Edge) (vd gd : DFrac)
    (hrows : constructorRowsValid nodes.length rows) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ receiver ↦{vd} view ∗ view.graph' ↦{gd} graph ∗
      graph.Nodes' ↦*{gd} nodes ∗ graph.Tables' ↦{gd} table ∗ table.Strings' ↦*{gd} strings ∗
      view.operands' ↦* rows }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"operandEdge" #receiver)) (Val #edge))
    {{ RET #(); receiver ↦{vd} view ∗ view.graph' ↦{gd} graph ∗
      graph.Nodes' ↦*{gd} nodes ∗ graph.Tables' ↦{gd} table ∗ table.Strings' ↦*{gd} strings ∗
      view.operands' ↦* constructorFilteredOperandSet strings nodes.length rows edge }} := by
  classical
  by_cases haccepted : constructorAcceptedEdge strings nodes.length edge
  · have hfiltered : constructorFilteredOperandSet strings nodes.length rows edge =
        constructorOperandSet rows edge := by
      simp [constructorFilteredOperandSet, constructorEdgeApplies, haccepted]
    rw [hfiltered]
    exact wp_operandEdge_accepted receiver view graph table strings nodes rows edge vd gd hrows haccepted
  · have hfiltered : constructorFilteredOperandSet strings nodes.length rows edge = rows := by
      simp [constructorFilteredOperandSet, constructorEdgeApplies, haccepted]
    rw [hfiltered]
    exact wp_operandEdge_rejected receiver view graph table strings nodes rows edge vd gd haccepted

end execution
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
