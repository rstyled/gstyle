module
public import Gstyle.Proof.Worklist.ConstructorLayout
public import Gstyle.Proof.RuntimePackage
public import Gstyle.Proof.Transfer.FactSnapshot
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

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
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

/-- The absent binding returns before reading the tracked slice or queue. -/
theorem wp_track_zero (receiver queue : Loc) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"track" #receiver))
        (Val #(W32 0))) (Val #queue))
    {{ RET #(); True }} := by
  wp_start
  wp_auto
  wp_end

/-- The absent node returns before reading the visited slice or receiver. -/
theorem wp_trackValue_zero (receiver queue : Loc) (visited : GoSlice) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"trackValue" #receiver))
        (Val #(W32 0))) (Val #visited)) (Val #queue))
    {{ RET #(); True }} := by
  wp_start
  wp_auto
  wp_end

/-- A non-call executes the production category check without reading or changing
the visited slice, queue, or tracked bindings. -/
theorem wp_trackCall_scalar (receiver : Loc) (view : graphView) (graph : pdg.Graph)
    (nodes : List pdg.Node) (node : pdg.Node) (ref : w32)
    (visited : GoSlice) (queue : Loc) (vd gd : DFrac)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ nodes.length)
    (hnode : nodes[(uint.Z ref - 1).toNat]? = some node)
    (hcategory : node.Category' ≠ W8 5) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ receiver ↦{vd} view ∗
       view.graph' ↦{gd} graph ∗ graph.Nodes' ↦*{gd} nodes }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"trackCall" #receiver))
        (Val #ref)) (Val #visited)) (Val #queue))
    {{ RET #(); receiver ↦{vd} view ∗ view.graph' ↦{gd} graph ∗ graph.Nodes' ↦*{gd} nodes }} := by
  wp_start as ⟨Hview, Hgraph, Hnodes⟩
  ihave %Hlen := ownSlice_len $$ Hnodes
  wp_auto
  have Hi : sint.Z (W64 (uint.Z (ref - W32 1))) = uint.Z ref - 1 := by
    obtain ⟨_, _⟩ := href; word
  rw [Hi]
  rw [ite_eq_left (by obtain ⟨_, _⟩ := href; constructor <;> word :
    0 ≤ uint.Z ref - 1 ∧ uint.Z ref - 1 < sint.Z graph.Nodes'.len)]
  icases ownSlice_elem_acc (uint.Z ref - 1) node graph.Nodes' gd nodes
    (by omega) hnode $$ Hnodes with ⟨Hnode, Hnodes⟩
  wp_auto
  ihave Hnodes := Hnodes $$ %node Hnode
  rw [list_set_lookup_self nodes (uint.Z ref - 1).toNat node hnode]
  rw [decide_eq_false hcategory]
  wp_auto
  wp_end

end execution
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
