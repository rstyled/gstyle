module
public import Gstyle.Proof.Worklist.ConstructorLayout
public import Gstyle.Proof.RuntimePackage
public import Gstyle.Proof.Pdg.Text
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

/-- The constructor accepts exactly the three production operand edge labels. -/
def constructorOperandLabel (kind : GoString) : Bool :=
  decide (kind = go!"source-value" ∨ kind = go!"scalar-value" ∨ kind = go!"control-value")

section execution
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [pdg.query.Assumptions]
variable [pdg.Edge.TypeAssumptions] [pdg.EdgeKind.TypeAssumptions] [pdg.Position.TypeAssumptions]
variable [FuncUnfold operandReferences [] operandReferences.impl]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

/-- Execute the production bounds check with a positive signed node count. -/
theorem wp_operandReferences_valid (edge : pdg.Edge) (count : w64)
    (hsource : 0 < uint.Z edge.Source' ∧ uint.Z edge.Source' ≤ sint.Z count)
    (htarget : 0 < uint.Z edge.Target' ∧ uint.Z edge.Target' ≤ sint.Z count) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! operandReferences)) (Val #edge)) (Val #count))
    {{ RET #true; True }} := by
  wp_start
  wp_auto
  have hs : edge.Source' ≠ W32 0 := by obtain ⟨_, _⟩ := hsource; word
  have ht : edge.Target' ≠ W32 0 := by obtain ⟨_, _⟩ := htarget; word
  rw [decide_eq_false hs]
  wp_auto
  rw [decide_eq_true (by obtain ⟨_, _⟩ := hsource; word : sint.Z (W64 (uint.Z edge.Source')) ≤ sint.Z count)]
  wp_auto
  rw [decide_eq_false ht]
  wp_auto
  rw [decide_eq_true (by obtain ⟨_, _⟩ := htarget; word : sint.Z (W64 (uint.Z edge.Target')) ≤ sint.Z count)]
  wp_end

variable [pdg.Graph.TypeAssumptions] [pdg.Node.TypeAssumptions]
variable [pdg.AttributeSet.TypeAssumptions] [pdg.AttributeTables.TypeAssumptions]
variable [pdg.Producer.TypeAssumptions] [pdg.Function.TypeAssumptions] [pdg.Span.TypeAssumptions]
variable [pdg.NormalFlow.TypeAssumptions] [pdg.CountFact.TypeAssumptions]
variable [pdg.Conformance.TypeAssumptions] [pdg.Fact.TypeAssumptions]
variable [pdg.CallForm.TypeAssumptions] [pdg.ExitKind.TypeAssumptions]
variable [pdg.ConformanceStatus.TypeAssumptions] [pdg.Category.TypeAssumptions]
variable [pdg.Tables.TypeAssumptions]
variable [FuncUnfold sourceOperandEdge [] sourceOperandEdge.impl]

omit [FuncUnfold operandReferences [] operandReferences.impl]
  [pdg.Node.TypeAssumptions] [pdg.AttributeSet.TypeAssumptions] [pdg.NormalFlow.TypeAssumptions]
  [pdg.CallForm.TypeAssumptions] [pdg.ExitKind.TypeAssumptions] [pdg.Category.TypeAssumptions] in
/-- Execute the label lookup after the production kind and position checks. -/
theorem wp_sourceOperandEdge_position (receiver : Loc) (graph : pdg.Graph)
    (table : pdg.Tables) (strings : List GoString) (edge : pdg.Edge) (dq : DFrac)
    (hkind : edge.Kind' = W8 0) (hposition : 0 < uint.Z edge.Position' ∧ uint.Z edge.Position' ≤ 2) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ receiver ↦{dq} graph ∗
      graph.Tables' ↦{dq} table ∗ table.Strings' ↦*{dq} strings }}
      (App (App (Val (@! sourceOperandEdge)) (Val #receiver)) (Val #edge))
    {{ RET #(constructorOperandLabel (pdg.tableText strings edge.Subkind'));
      receiver ↦{dq} graph ∗ graph.Tables' ↦{dq} table ∗ table.Strings' ↦*{dq} strings }} := by
  wp_start as ⟨Hgraph, Htable, Hstrings⟩
  wp_auto
  rw [decide_eq_true hkind]
  wp_auto
  have hz : edge.Position' ≠ W32 0 := by obtain ⟨_, _⟩ := hposition; word
  rw [decide_eq_false hz]
  wp_auto
  rw [decide_eq_false (by obtain ⟨_, _⟩ := hposition; word : ¬ (uint.Z edge.Position' > uint.Z (W32 2)))]
  wp_auto
  wp_apply pdg.wp_Text graph.Tables' table strings dq edge.Subkind' $$ [Htable Hstrings]
    with ⟨Htable, Hstrings⟩
  · iframe
  by_cases hs : pdg.tableText strings edge.Subkind' = go!"source-value" <;>
    by_cases hv : pdg.tableText strings edge.Subkind' = go!"scalar-value" <;>
    by_cases hc : pdg.tableText strings edge.Subkind' = go!"control-value"
  all_goals simp only [constructorOperandLabel, hs, hv, hc, _root_.true_or, _root_.false_or, _root_.or_true, decide_true, decide_false]
  all_goals iterate 5 (try wp_auto; try simp only [hs, hv, hc, decide_true, decide_false])
  all_goals wp_end

variable [graphView.TypeAssumptions]

/-- The accepted production edge writes one operand slot and preserves every other slot. -/
theorem wp_operandEdge_write (receiver : Loc) (view : graphView) (graph : pdg.Graph)
    (table : pdg.Tables) (strings : List GoString) (nodes : List pdg.Node)
    (rows : List (GoArray w32 2)) (row : GoArray w32 2) (edge : pdg.Edge)
    (vd gd : DFrac) (hkind : edge.Kind' = W8 0)
    (hposition : 0 < uint.Z edge.Position' ∧ uint.Z edge.Position' ≤ 2)
    (hlabel : constructorOperandLabel (pdg.tableText strings edge.Subkind') = true)
    (hsource : 0 < uint.Z edge.Source' ∧ uint.Z edge.Source' ≤ nodes.length)
    (htarget : 0 < uint.Z edge.Target' ∧ uint.Z edge.Target' ≤ nodes.length)
    (hlen : rows.length = nodes.length)
    (hrow : rows[(uint.Z edge.Target' - 1).toNat]? = some row)
    (harity : row.arr.length = 2) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ receiver ↦{vd} view ∗ view.graph' ↦{gd} graph ∗
      graph.Nodes' ↦*{gd} nodes ∗ graph.Tables' ↦{gd} table ∗ table.Strings' ↦*{gd} strings ∗
      view.operands' ↦* rows }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"operandEdge" #receiver)) (Val #edge))
    {{ RET #(); receiver ↦{vd} view ∗ view.graph' ↦{gd} graph ∗
      graph.Nodes' ↦*{gd} nodes ∗ graph.Tables' ↦{gd} table ∗ table.Strings' ↦*{gd} strings ∗
      view.operands' ↦* rows.set (uint.Z edge.Target' - 1).toNat
        (array.mk 2 (row.arr.set (uint.Z edge.Position' - 1).toNat edge.Source')) }} := by
  wp_start as ⟨Hview, Hgraph, Hnodes, Htable, Hstrings, Hrows⟩
  ihave %Hnlen := ownSlice_len $$ Hnodes
  ihave %Hrlen := ownSlice_len $$ Hrows
  wp_auto
  wp_apply wp_sourceOperandEdge_position view.graph' graph table strings edge gd hkind hposition
    $$ [Hgraph Htable Hstrings] with ⟨Hgraph, Htable, Hstrings⟩
  · iframe
  rw [hlabel]
  wp_auto
  wp_apply wp_operandReferences_valid edge graph.Nodes'.len
    (by obtain ⟨_, _⟩ := hsource; constructor <;> word)
    (by obtain ⟨_, _⟩ := htarget; constructor <;> word)
  try wp_auto
  have Htarget : sint.Z (W64 (uint.Z (edge.Target' - W32 1))) = uint.Z edge.Target' - 1 := by
    obtain ⟨_, _⟩ := htarget; word
  have Hposition : sint.Z (W64 (uint.Z (edge.Position' - W32 1))) = uint.Z edge.Position' - 1 := by
    obtain ⟨_, _⟩ := hposition; word
  rw [Htarget]
  rw [ite_eq_left (by obtain ⟨_, _⟩ := htarget; constructor <;> word : 0 ≤ uint.Z edge.Target' - 1 ∧ uint.Z edge.Target' - 1 < sint.Z view.operands'.len)]
  wp_auto
  rw [Hposition]
  rw [ite_eq_left (by omega : 0 ≤ uint.Z edge.Position' - 1 ∧ uint.Z edge.Position' - 1 < 2)]
  obtain ⟨value, hvalue⟩ : ∃ value, row.arr[(uint.Z edge.Position' - 1).toNat]? = some value :=
    ⟨_, List.getElem?_eq_getElem (by omega)⟩
  icases ownSlice_elem_acc (uint.Z edge.Target' - 1) row view.operands' (DFrac.own 1) rows
    (by omega) hrow $$ Hrows with ⟨Hrow, Hrows⟩
  icases array_acc (sliceIndexRef (GoArray w32 2) (uint.Z edge.Target' - 1) view.operands')
    (uint.Z edge.Position' - 1) (DFrac.own 1) 2 row value (by omega) hvalue $$ Hrow
    with ⟨Hvalue, Hrow⟩
  wp_auto
  ihave Hupdated : (sliceIndexRef (GoArray w32 2) (uint.Z edge.Target' - 1) view.operands' ↦
      array.mk 2 (row.arr.set (uint.Z edge.Position' - 1).toNat edge.Source') : IProp GF) $$ [Hrow Hvalue]
  · iapply Hrow $$ %edge.Source'
    iframe Hvalue
  ihave Hrows := Hrows $$ %(array.mk 2 (row.arr.set (uint.Z edge.Position' - 1).toNat edge.Source')) Hupdated
  wp_end

end execution
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
