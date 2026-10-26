module
public import Gstyle.Proof.Worklist.ConstructorOperands
public import Gstyle.Proof.Worklist.ConstructorRows
public import Gstyle.Proof.Worklist.ConstructorSyntax
@[expose] public section
noncomputable section
set_option maxHeartbeats 0
set_option maxRecDepth 100000
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

/-- A finite scalar graph may use both assignment and branch operand labels. -/
def constructorAcceptedEdge [FfiSyntax] (strings : List GoString) (count : Nat) (edge : pdg.Edge) : Prop :=
  edge.Kind' = W8 0 ∧ (0 < uint.Z edge.Position' ∧ uint.Z edge.Position' ≤ 2) ∧
    constructorOperandLabel (pdg.tableText strings edge.Subkind') = true ∧
    (0 < uint.Z edge.Source' ∧ uint.Z edge.Source' ≤ count) ∧
    (0 < uint.Z edge.Target' ∧ uint.Z edge.Target' ≤ count)

section execution
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [pdg.query.Assumptions] [graphView.TypeAssumptions]
variable [pdg.Edge.TypeAssumptions] [pdg.EdgeKind.TypeAssumptions] [pdg.Position.TypeAssumptions]
variable [pdg.Graph.TypeAssumptions] [pdg.Node.TypeAssumptions]
variable [pdg.AttributeSet.TypeAssumptions] [pdg.AttributeTables.TypeAssumptions]
variable [pdg.Producer.TypeAssumptions] [pdg.Function.TypeAssumptions] [pdg.Span.TypeAssumptions]
variable [pdg.NormalFlow.TypeAssumptions] [pdg.CountFact.TypeAssumptions]
variable [pdg.Conformance.TypeAssumptions] [pdg.Fact.TypeAssumptions]
variable [pdg.CallForm.TypeAssumptions] [pdg.ExitKind.TypeAssumptions]
variable [pdg.ConformanceStatus.TypeAssumptions] [pdg.Category.TypeAssumptions]
variable [pdg.Tables.TypeAssumptions]
variable [FuncUnfold sourceOperandEdge [] sourceOperandEdge.impl]
variable [FuncUnfold operandReferences [] operandReferences.impl]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

/-- Execute every source-selected edge iteration, including repeated target slots. -/
theorem wp_constructor_edges_raw (graphLocal viewLocal receiver : Loc) (view : graphView)
    (graph : pdg.Graph) (table : pdg.Tables) (strings : List GoString)
    (nodes : List pdg.Node) (edges : List pdg.Edge) (initial : List (GoArray w32 2))
    (hrows : constructorRowsValid nodes.length initial)
    (hedges : ∀ edge ∈ edges, constructorAcceptedEdge strings nodes.length edge) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ graphLocal ↦ view.graph' ∗ viewLocal ↦ receiver ∗
       receiver ↦ view ∗ view.graph' ↦□ graph ∗ graph.Tables' ↦□ table ∗ table.Strings' ↦*□ strings ∗
       graph.Nodes' ↦*□ nodes ∗ graph.Edges' ↦*□ edges ∗ view.operands' ↦* initial }}
      (constructorEdgesRawRun graphLocal viewLocal)
    {{ RET executeVal; graphLocal ↦ view.graph' ∗ viewLocal ↦ receiver ∗ receiver ↦ view ∗
       view.operands' ↦* edges.foldl constructorOperandSet initial }} := by
  wp_start_folded as ⟨Hg, Hv, Hview, #Hgraph, #Htable, #Hstrings, #Hnodes, #Hedges, Hrows⟩
  ihave %Hlen := ownSlice_len $$ Hedges
  unfold constructorEdgesRawRun constructorEdgeRange constructorBody newGraphView.impl
  dsimp only [constructorRangeSite, Option.or, Option.getD]
  simp only [Let, Lam]
  try wp_auto
  ihave HI : (∃ (iv : w64) (previous : pdg.Edge),
      "i" ∷ i_ptr ↦ iv ∗ "edge" ∷ edge_ptr ↦ previous ∗
      "Hg" ∷ graphLocal ↦ view.graph' ∗ "Hv" ∷ viewLocal ↦ receiver ∗ "Hview" ∷ receiver ↦ view ∗
      "Hrows" ∷ view.operands' ↦* (edges.take (sint.nat iv)).foldl constructorOperandSet initial ∗
      "%Hbounds" ∷ ⌜0 ≤ sint.Z iv ∧ sint.Z iv ≤ sint.Z graph.Edges'.len⌝ : IProp GF)
      $$ [i edge Hg Hv Hview Hrows]
  · iexists (W64 0), (zero_val pdg.Edge)
    simp only [show sint.nat (W64 0) = 0 from rfl, List.take_zero, List.foldl_nil]
    iframe
    ipureintro; constructor <;> word
  wp_for HI
  wp_if_destruct
  · have hi : sint.nat iv < edges.length := by word
    list_elem edges (sint.nat iv) as entry
    rw [ite_eq_left (by constructor <;> word : 0 ≤ sint.Z iv ∧ sint.Z iv < sint.Z graph.Edges'.len)]
    wp_apply wp_load_slice_index graph.Edges' (sint.Z iv) edges DFrac.discard entry (by word)
      $$ [Hedges] with Hedges
    · iframe Hedges; ipureintro; exact Hentry_lookup
    have he := hedges entry (List.mem_of_getElem? Hentry_lookup)
    rcases he with ⟨hkind, hposition, hlabel, hsource, htarget⟩
    let current := (edges.take (sint.nat iv)).foldl constructorOperandSet initial
    have hcurrent : constructorRowsValid nodes.length current := by
      apply constructorRowsValid_fold nodes.length initial _ hrows
      intro edge hm
      have he := hedges edge (List.mem_of_mem_take hm)
      exact ⟨he.2.2.2.2, Or.inr he.2.2.2.1⟩
    have ht : (uint.Z entry.Target' - 1).toNat < current.length := by
      obtain ⟨hlen, _⟩ := hcurrent
      omega
    list_elem current (uint.Z entry.Target' - 1).toNat as row
    have harity := (hcurrent.2 row (List.mem_of_getElem? Hrow_lookup)).1
    try wp_auto
    wp_apply wp_operandEdge_write receiver view graph table strings nodes current row entry
      (DFrac.own 1) DFrac.discard hkind hposition hlabel hsource htarget hcurrent.1 Hrow_lookup harity
      $$ [Hview Hgraph Hnodes Htable Hstrings Hrows]
      with ⟨Hview, Hgraph, Hnodes, Htable, Hstrings, Hrows⟩
    · iframe Hview Hgraph Hnodes Htable Hstrings Hrows
    wp_for_post
    iframe
    iexists (iv + W64 1), entry
    have hn : sint.nat (iv + W64 1) = sint.nat iv + 1 := by word
    have hstep : (edges.take (sint.nat iv + 1)).foldl constructorOperandSet initial =
        constructorOperandSet current entry := by
      rw [List.take_add_one, Hentry_lookup]
      simp only [Option.toList_some, List.foldl_append, List.foldl_cons, List.foldl_nil]
      rfl
    have hset : constructorOperandSet current entry = current.set (uint.Z entry.Target' - 1).toNat
        (array.mk 2 (row.arr.set (uint.Z entry.Position' - 1).toNat entry.Source')) := by
      simp only [constructorOperandSet, Hrow_lookup, Option.getD_some]
    rw [hn, hstep, hset]
    iframe
    ipureintro; constructor <;> word
  · have hi : sint.nat iv = edges.length := by word
    rw [hi, List.take_length]
    wp_end

end execution
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
