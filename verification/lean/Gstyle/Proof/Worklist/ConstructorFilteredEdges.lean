module
public import Gstyle.Proof.Worklist.ConstructorOperandGuards
@[expose] public section
noncomputable section
set_option maxHeartbeats 0
set_option maxRecDepth 100000
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

section rows
variable [FfiSyntax]

/-- Every applied edge has bounded source and target references. -/
theorem constructorEdgeApplies_accepted (strings : List GoString) (count : Nat) (edge : pdg.Edge)
    (happlies : constructorEdgeApplies strings count edge = true) :
    constructorAcceptedEdge strings count edge := by
  simpa only [constructorEdgeApplies, decide_eq_true_eq] using happlies

/-- Ignored edges preserve rows; applied edges preserve shape and reference safety. -/
theorem constructorRowsValid_filtered_set (count : Nat) (strings : List GoString)
    (rows : List (GoArray w32 2)) (edge : pdg.Edge)
    (hrows : constructorRowsValid count rows) :
    constructorRowsValid count (constructorFilteredOperandSet strings count rows edge) := by
  unfold constructorFilteredOperandSet
  split
  · rename_i happlies
    have hedge := constructorEdgeApplies_accepted strings count edge happlies
    exact constructorRowsValid_set count rows edge hrows hedge.2.2.2.2 (Or.inr hedge.2.2.2.1)
  · exact hrows

/-- Arbitrary source edges produce safe rows; each write passes the actual guards. -/
theorem constructorRowsValid_filtered_fold (count : Nat) (strings : List GoString)
    (rows : List (GoArray w32 2)) (edges : List pdg.Edge)
    (hrows : constructorRowsValid count rows) :
    constructorRowsValid count
      (edges.foldl (constructorFilteredOperandSet strings count) rows) := by
  induction edges generalizing rows with
  | nil => exact hrows
  | cons edge edges ih =>
    exact ih _ (constructorRowsValid_filtered_set count strings rows edge hrows)

/-- Fresh zero arrays followed by the full edge list remain safe without input restrictions. -/
theorem constructorRowsValid_filtered_zero (count : Nat) (strings : List GoString)
    (edges : List pdg.Edge) :
    constructorRowsValid count (edges.foldl (constructorFilteredOperandSet strings count)
      (List.replicate count (zero_val (GoArray w32 2)))) :=
  constructorRowsValid_filtered_fold count strings _ edges (constructorRowsValid_zero count)

/-- The guarded full-list fold equals the ordinary operand fold over applied edges. -/
theorem constructorFilteredOperandSet_fold_filter (strings : List GoString) (count : Nat)
    (rows : List (GoArray w32 2)) (edges : List pdg.Edge) :
    edges.foldl (constructorFilteredOperandSet strings count) rows =
      (edges.filter (constructorEdgeApplies strings count)).foldl constructorOperandSet rows := by
  induction edges generalizing rows with
  | nil => rfl
  | cons edge edges ih =>
    by_cases hedge : constructorEdgeApplies strings count edge = true
    · simp only [List.foldl_cons, constructorFilteredOperandSet, hedge, ite_true,
        List.filter_cons_of_pos hedge]
      exact ih _
    · simp only [List.foldl_cons, constructorFilteredOperandSet, hedge,
        List.filter_cons_of_neg hedge]
      exact ih _

/-- On accepted input the filtered model agrees with the existing constructor model. -/
theorem constructorFilteredOperandSet_accepted (strings : List GoString) (count : Nat)
    (rows : List (GoArray w32 2)) (edge : pdg.Edge)
    (hedge : constructorAcceptedEdge strings count edge) :
    constructorFilteredOperandSet strings count rows edge = constructorOperandSet rows edge := by
  classical
  simp only [constructorFilteredOperandSet, constructorEdgeApplies, decide_eq_true hedge, ite_true]

/-- Existing all-accepted edge proofs retain their original fold result. -/
theorem constructorFilteredOperandSet_fold_accepted (strings : List GoString) (count : Nat)
    (rows : List (GoArray w32 2)) (edges : List pdg.Edge)
    (hedges : ∀ edge ∈ edges, constructorAcceptedEdge strings count edge) :
    edges.foldl (constructorFilteredOperandSet strings count) rows =
      edges.foldl constructorOperandSet rows := by
  induction edges generalizing rows with
  | nil => rfl
  | cons edge edges ih =>
    simp only [List.foldl_cons]
    rw [constructorFilteredOperandSet_accepted strings count rows edge (hedges edge (by simp))]
    exact ih _ (fun e hm => hedges e (by simp [hm]))

end rows

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

/-- Execute the actual source-selected loop over arbitrary edges; rejected guards leave rows intact. -/
theorem wp_constructor_edges_filtered_raw (graphLocal viewLocal receiver : Loc) (view : graphView)
    (graph : pdg.Graph) (table : pdg.Tables) (strings : List GoString)
    (nodes : List pdg.Node) (edges : List pdg.Edge) (initial : List (GoArray w32 2))
    (hrows : constructorRowsValid nodes.length initial) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ graphLocal ↦ view.graph' ∗ viewLocal ↦ receiver ∗
       receiver ↦ view ∗ view.graph' ↦□ graph ∗ graph.Tables' ↦□ table ∗ table.Strings' ↦*□ strings ∗
       graph.Nodes' ↦*□ nodes ∗ graph.Edges' ↦*□ edges ∗ view.operands' ↦* initial }}
      (constructorEdgesRawRun graphLocal viewLocal)
    {{ RET executeVal; graphLocal ↦ view.graph' ∗ viewLocal ↦ receiver ∗ receiver ↦ view ∗
       view.operands' ↦* edges.foldl (constructorFilteredOperandSet strings nodes.length) initial }} := by
  wp_start_folded as ⟨Hg, Hv, Hview, #Hgraph, #Htable, #Hstrings, #Hnodes, #Hedges, Hrows⟩
  ihave %Hlen := ownSlice_len $$ Hedges
  unfold constructorEdgesRawRun constructorEdgeRange constructorBody newGraphView.impl
  dsimp only [constructorRangeSite, Option.or, Option.getD]
  simp only [Let, Lam]
  try wp_auto
  ihave HI : (∃ (iv : w64) (previous : pdg.Edge),
      "i" ∷ i_ptr ↦ iv ∗ "edge" ∷ edge_ptr ↦ previous ∗
      "Hg" ∷ graphLocal ↦ view.graph' ∗ "Hv" ∷ viewLocal ↦ receiver ∗ "Hview" ∷ receiver ↦ view ∗
      "Hrows" ∷ view.operands' ↦* (edges.take (sint.nat iv)).foldl (constructorFilteredOperandSet strings nodes.length) initial ∗
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
    let current := (edges.take (sint.nat iv)).foldl
      (constructorFilteredOperandSet strings nodes.length) initial
    have hcurrent : constructorRowsValid nodes.length current :=
      constructorRowsValid_filtered_fold nodes.length strings initial _ hrows
    try wp_auto
    wp_apply wp_operandEdge_filtered receiver view graph table strings nodes current entry
      (DFrac.own 1) DFrac.discard hcurrent
      $$ [Hview Hgraph Hnodes Htable Hstrings Hrows]
      with ⟨Hview, Hgraph, Hnodes, Htable, Hstrings, Hrows⟩
    · iframe Hview Hgraph Hnodes Htable Hstrings Hrows
    wp_for_post
    iframe
    iexists (iv + W64 1), entry
    have hn : sint.nat (iv + W64 1) = sint.nat iv + 1 := by word
    have hstep : (edges.take (sint.nat iv + 1)).foldl (constructorFilteredOperandSet strings nodes.length) initial =
        constructorFilteredOperandSet strings nodes.length current entry := by
      rw [List.take_add_one, Hentry_lookup]
      simp only [Option.toList_some, List.foldl_append, List.foldl_cons, List.foldl_nil]
      rfl
    rw [hn, hstep]
    iframe
    ipureintro; constructor <;> word
  · have hi : sint.nat iv = edges.length := by word
    rw [hi, List.take_length]
    wp_end

/-- The complete source loop builds the filtered operand model from fresh zero arrays. -/
theorem wp_constructor_edges_filtered_zero_raw (graphLocal viewLocal receiver : Loc) (view : graphView)
    (graph : pdg.Graph) (table : pdg.Tables) (strings : List GoString)
    (nodes : List pdg.Node) (edges : List pdg.Edge) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ graphLocal ↦ view.graph' ∗ viewLocal ↦ receiver ∗
       receiver ↦ view ∗ view.graph' ↦□ graph ∗ graph.Tables' ↦□ table ∗ table.Strings' ↦*□ strings ∗
       graph.Nodes' ↦*□ nodes ∗ graph.Edges' ↦*□ edges ∗
       view.operands' ↦* List.replicate nodes.length (zero_val (GoArray w32 2)) }}
      (constructorEdgesRawRun graphLocal viewLocal)
    {{ RET executeVal; graphLocal ↦ view.graph' ∗ viewLocal ↦ receiver ∗ receiver ↦ view ∗
       view.operands' ↦* edges.foldl (constructorFilteredOperandSet strings nodes.length)
         (List.replicate nodes.length (zero_val (GoArray w32 2))) ∗
       ⌜constructorRowsValid nodes.length
         (edges.foldl (constructorFilteredOperandSet strings nodes.length)
           (List.replicate nodes.length (zero_val (GoArray w32 2))))⌝ }} := by
  wp_start_folded as ⟨Hg, Hv, Hview, #Hgraph, #Htable, #Hstrings, #Hnodes, #Hedges, Hrows⟩
  wp_apply wp_constructor_edges_filtered_raw graphLocal viewLocal receiver view graph table strings nodes edges
    (List.replicate nodes.length (zero_val (GoArray w32 2))) (constructorRowsValid_zero nodes.length)
    $$ [Hg Hv Hview Hgraph Htable Hstrings Hnodes Hedges Hrows] with ⟨Hg, Hv, Hview, Hrows⟩
  · iframe Hg Hv Hview Hgraph Htable Hstrings Hnodes Hedges Hrows
  have hfinal := constructorRowsValid_filtered_zero nodes.length strings edges
  wp_end

end execution
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
