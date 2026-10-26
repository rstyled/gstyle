module
public import Gstyle.Proof.RuntimePackage
public import Gstyle.Proof.Worklist.FlowLayout
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

/-- The production reference check uses one-based references and a signed count. -/
def constructorRefsValid (refs : List w32) (count : Int) : Prop :=
  ∀ ref ∈ refs, 0 < uint.Z ref ∧ uint.Z ref ≤ count

/-- A block and the contents of its two reference slices. -/
structure ConstructorFlowRow [FfiSyntax] where
  block : pdg.FlowBlock
  operations : List w32
  successors : List w32

/-- These are precisely the successful branches of the production flow validator. -/
def constructorFlowRowsValid [FfiSyntax] (nodeCount : Nat) (rows : List ConstructorFlowRow) : Prop :=
  ∀ row ∈ rows,
    uint.Z row.block.Condition' ≤ nodeCount ∧
    constructorRefsValid row.operations nodeCount ∧
    constructorRefsValid row.successors rows.length

section ownership
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics]

/-- Own each represented row; empty rows are allowed but are not required. -/
def constructorFlowRowsOwn (dq : DFrac) : List ConstructorFlowRow → IProp GF
  | [] => iprop(True)
  | row :: rows => iprop(
      row.block.Operations' ↦*{dq} row.operations ∗
      row.block.Successors' ↦*{dq} row.successors ∗ constructorFlowRowsOwn dq rows)

theorem constructorFlowRowsOwn_acc (dq : DFrac) (rows : List ConstructorFlowRow)
    (index : Nat) (row : ConstructorFlowRow) (hrow : rows[index]? = some row) :
    constructorFlowRowsOwn (GF := GF) dq rows ⊢
      (row.block.Operations' ↦*{dq} row.operations ∗ row.block.Successors' ↦*{dq} row.successors) ∗
      ((row.block.Operations' ↦*{dq} row.operations ∗ row.block.Successors' ↦*{dq} row.successors) -∗
        constructorFlowRowsOwn dq rows) := by
  induction rows generalizing index with
  | nil => simp at hrow
  | cons head rows ih =>
    cases index with
    | zero =>
      simp only [List.getElem?_cons_zero, Option.some.injEq] at hrow
      subst row
      simp only [constructorFlowRowsOwn]
      iintro ⟨Hoperations, Hsuccessors, Htail⟩
      iframe Hoperations Hsuccessors
      iintro ⟨Hoperations, Hsuccessors⟩
      iframe
    | succ index =>
      simp only [List.getElem?_cons_succ] at hrow
      simp only [constructorFlowRowsOwn]
      iintro ⟨Hoperations, Hsuccessors, Htail⟩
      ihave ⟨Hrow, Hrestore⟩ := ih index hrow $$ Htail
      iframe Hrow
      iintro Hrow
      ihave Htail := Hrestore $$ Hrow
      iframe

end ownership

section execution
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [pdg.query.Assumptions]
variable [FuncUnfold validateRefs [] validateRefs.impl]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

/-- Execute every reference check, retaining the input array at its original fraction. -/
theorem wp_validateRefs_valid (refs : GoSlice) (values : List w32) (count : w64)
    (dq : DFrac) (hvalid : constructorRefsValid values (sint.Z count)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ refs ↦*{dq} values }}
      (App (App (Val (@! validateRefs)) (Val #refs)) (Val #count))
    {{ RET #interface.nil; refs ↦*{dq} values }} := by
  wp_start as Hrefs
  ihave %Hlen := ownSlice_len $$ Hrefs
  wp_auto
  ihave HI : (∃ (iv : w64) (rv : w32),
      "i" ∷ i_ptr ↦ iv ∗
      "ref" ∷ ref_ptr ↦ rv ∗
      "Hrefs" ∷ refs ↦*{dq} values ∗
      "%Hbounds" ∷ ⌜0 ≤ sint.Z iv ∧ sint.Z iv ≤ sint.Z refs.len⌝ : IProp GF)
      $$ [i ref Hrefs]
  · iexists (W64 0), (zero_val w32)
    iframe
    ipureintro
    constructor <;> word
  wp_for HI
  wp_if_destruct
  · have hi : sint.nat iv < values.length := by word
    list_elem values (sint.nat iv) as value
    rw [ite_eq_left (by constructor <;> word : 0 ≤ sint.Z iv ∧ sint.Z iv < sint.Z refs.len)]
    wp_apply wp_load_slice_index refs (sint.Z iv) values dq value Hbounds.1
      $$ [Hrefs] with Hrefs
    · iframe
      ipureintro
      exact Hvalue_lookup
    try wp_auto
    have hv := hvalid value (List.mem_of_getElem? Hvalue_lookup)
    rw [decide_eq_false (by obtain ⟨_, _⟩ := hv; word : value ≠ W32 0)]
    try wp_auto
    rw [decide_eq_false (by obtain ⟨_, _⟩ := hv; word : ¬ sint.Z (W64 (uint.Z value)) > sint.Z count)]
    try wp_auto
    wp_for_post
    iframe
    iexists (iv + W64 1), value
    iframe
    ipureintro
    constructor <;> word
  · wp_end

variable [pdg.Graph.TypeAssumptions]
variable [pdg.AttributeTables.TypeAssumptions]
variable [pdg.Producer.TypeAssumptions] [pdg.Function.TypeAssumptions] [pdg.Span.TypeAssumptions]
variable [pdg.NormalFlow.TypeAssumptions] [pdg.FlowBlock.TypeAssumptions]
variable [pdg.CountFact.TypeAssumptions] [pdg.Conformance.TypeAssumptions] [pdg.Fact.TypeAssumptions]
variable [pdg.ConformanceStatus.TypeAssumptions]
variable [FuncUnfold validateBlock [] validateBlock.impl]

/-- Check the condition and both represented reference rows of the actual block. -/
theorem wp_validateBlock_valid (receiver : Loc) (graph : pdg.Graph) (flow : pdg.NormalFlow)
    (nodes : List pdg.Node) (blocks : List pdg.FlowBlock) (block : pdg.FlowBlock)
    (operations successors : List w32) (dq : DFrac)
    (hcondition : uint.Z block.Condition' ≤ nodes.length)
    (hoperations : constructorRefsValid operations nodes.length)
    (hsuccessors : constructorRefsValid successors blocks.length) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ receiver ↦{dq} graph ∗
       graph.ValueFlow' ↦{dq} flow ∗ graph.Nodes' ↦*{dq} nodes ∗
       flow.Blocks' ↦*{dq} blocks ∗ block.Operations' ↦*{dq} operations ∗
       block.Successors' ↦*{dq} successors }}
      (App (App (Val (@! validateBlock)) (Val #receiver)) (Val #block))
    {{ RET #interface.nil; receiver ↦{dq} graph ∗ graph.ValueFlow' ↦{dq} flow ∗
       graph.Nodes' ↦*{dq} nodes ∗ flow.Blocks' ↦*{dq} blocks ∗
       block.Operations' ↦*{dq} operations ∗ block.Successors' ↦*{dq} successors }} := by
  wp_start as ⟨Hgraph, Hflow, Hnodes, Hblocks, Hoperations, Hsuccessors⟩
  ihave %HnodesLen := ownSlice_len $$ Hnodes
  ihave %HblocksLen := ownSlice_len $$ Hblocks
  wp_auto
  rw [decide_eq_false (by word : ¬ sint.Z (W64 (uint.Z block.Condition')) > sint.Z graph.Nodes'.len)]
  wp_auto
  wp_apply wp_validateRefs_valid block.Operations' operations graph.Nodes'.len dq
    (by intro ref href; have h := hoperations ref href; constructor <;> word)
    $$ [Hoperations] with Hoperations
  · iframe
  try wp_auto
  wp_apply wp_validateRefs_valid block.Successors' successors flow.Blocks'.len dq
    (by intro ref href; have h := hsuccessors ref href; constructor <;> word)
    $$ [Hsuccessors] with Hsuccessors
  · iframe
  wp_end

variable [FuncUnfold validateFlow [] validateFlow.impl]

/-- Execute the entry guard and every block check of the production validator.
    The graph, flow, node and block arrays, and all reference rows remain unchanged. -/
theorem wp_validateFlow_valid (receiver : Loc) (graph : pdg.Graph) (flow : pdg.NormalFlow)
    (nodes : List pdg.Node) (rows : List ConstructorFlowRow) (dq : DFrac)
    (htables : graph.Tables' ≠ null)
    (hentry : 0 < uint.Z flow.Entry' ∧ uint.Z flow.Entry' ≤ rows.length)
    (hrows : constructorFlowRowsValid nodes.length rows) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ receiver ↦{dq} graph ∗
       graph.ValueFlow' ↦{dq} flow ∗ graph.Nodes' ↦*{dq} nodes ∗
       flow.Blocks' ↦*{dq} rows.map ConstructorFlowRow.block ∗ constructorFlowRowsOwn dq rows }}
      (App (Val (@! validateFlow)) (Val #receiver))
    {{ RET #interface.nil; receiver ↦{dq} graph ∗ graph.ValueFlow' ↦{dq} flow ∗
       graph.Nodes' ↦*{dq} nodes ∗ flow.Blocks' ↦*{dq} rows.map ConstructorFlowRow.block ∗
       constructorFlowRowsOwn dq rows }} := by
  wp_start as ⟨Hgraph, Hflow, Hnodes, Hblocks, Hrows⟩
  ihave %Hlen := ownSlice_len $$ Hblocks
  simp only [List.length_map] at Hlen
  wp_auto
  rw [decide_eq_false htables]
  wp_auto
  rw [decide_eq_false (by obtain ⟨_, _⟩ := hentry; word : flow.Entry' ≠ W32 0)]
  wp_auto
  rw [decide_eq_false (by obtain ⟨_, _⟩ := hentry; word :
    ¬ sint.Z (W64 (uint.Z flow.Entry')) > sint.Z flow.Blocks'.len)]
  wp_auto
  ihave HI : (∃ (iv : w64) (bv : pdg.FlowBlock),
      "i" ∷ i_ptr ↦ iv ∗ "block" ∷ block_ptr ↦ bv ∗
      "Hgraph" ∷ receiver ↦{dq} graph ∗ "Hflow" ∷ graph.ValueFlow' ↦{dq} flow ∗
      "Hnodes" ∷ graph.Nodes' ↦*{dq} nodes ∗
      "Hblocks" ∷ flow.Blocks' ↦*{dq} rows.map ConstructorFlowRow.block ∗
      "Hrows" ∷ constructorFlowRowsOwn dq rows ∗
      "%Hbounds" ∷ ⌜0 ≤ sint.Z iv ∧ sint.Z iv ≤ sint.Z flow.Blocks'.len⌝ : IProp GF)
      $$ [i block Hgraph Hflow Hnodes Hblocks Hrows]
  · iexists (W64 0), (zero_val pdg.FlowBlock)
    iframe
    ipureintro
    constructor <;> word
  wp_for HI
  wp_if_destruct
  · have hi : sint.nat iv < rows.length := by word
    list_elem rows (sint.nat iv) as row
    have hblock : (rows.map ConstructorFlowRow.block)[sint.nat iv]? = some row.block := by
      simp only [List.getElem?_map, Hrow_lookup, Option.map_some]
    rw [ite_eq_left (by constructor <;> word :
      0 ≤ sint.Z iv ∧ sint.Z iv < sint.Z flow.Blocks'.len)]
    wp_apply wp_load_slice_index flow.Blocks' (sint.Z iv) (rows.map ConstructorFlowRow.block)
      dq row.block Hbounds.1 $$ [Hblocks] with Hblocks
    · iframe
      ipureintro
      exact hblock
    ihave ⟨Hrow, Hrestore⟩ := constructorFlowRowsOwn_acc dq rows (sint.nat iv) row Hrow_lookup $$ Hrows
    icases Hrow with ⟨Hoperations, Hsuccessors⟩
    have hv := hrows row (List.mem_of_getElem? Hrow_lookup)
    try wp_auto
    wp_apply wp_validateBlock_valid receiver graph flow nodes (rows.map ConstructorFlowRow.block)
      row.block row.operations row.successors dq hv.1 hv.2.1
      (by simpa only [List.length_map] using hv.2.2)
      $$ [Hgraph Hflow Hnodes Hblocks Hoperations Hsuccessors]
      with ⟨Hgraph, Hflow, Hnodes, Hblocks, Hoperations, Hsuccessors⟩
    · iframe
    ihave Hrow : (row.block.Operations' ↦*{dq} row.operations ∗
        row.block.Successors' ↦*{dq} row.successors : IProp GF) $$ [Hoperations Hsuccessors]
    · iframe
    ihave Hrows := Hrestore $$ Hrow
    try wp_auto
    wp_for_post
    iframe
    iexists (iv + W64 1), row.block
    iframe
    ipureintro
    constructor <;> word
  · wp_end

/-- An explicit nonempty case: node 1 is the condition and operation, and the
    single block has a self-successor. This does not require an acyclic flow. -/
theorem wp_validateFlow_singleton (receiver : Loc) (graph : pdg.Graph) (flow : pdg.NormalFlow)
    (nodes : List pdg.Node) (block : pdg.FlowBlock) (dq : DFrac)
    (htables : graph.Tables' ≠ null) (hentry : flow.Entry' = W32 1)
    (hcondition : block.Condition' = W32 1) (hnodes : 0 < nodes.length) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ receiver ↦{dq} graph ∗
       graph.ValueFlow' ↦{dq} flow ∗ graph.Nodes' ↦*{dq} nodes ∗ flow.Blocks' ↦*{dq} [block] ∗
       block.Operations' ↦*{dq} [W32 1] ∗ block.Successors' ↦*{dq} [W32 1] }}
      (App (Val (@! validateFlow)) (Val #receiver))
    {{ RET #interface.nil; receiver ↦{dq} graph ∗ graph.ValueFlow' ↦{dq} flow ∗
       graph.Nodes' ↦*{dq} nodes ∗ flow.Blocks' ↦*{dq} [block] ∗
       block.Operations' ↦*{dq} [W32 1] ∗ block.Successors' ↦*{dq} [W32 1] }} := by
  let row : ConstructorFlowRow := ⟨block, [W32 1], [W32 1]⟩
  have hvalid : constructorFlowRowsValid nodes.length [row] := by
    intro current hcurrent
    simp only [List.mem_singleton] at hcurrent
    subst current
    refine ⟨?_, ?_, ?_⟩
    · dsimp only [row]
      rw [hcondition]
      word
    · intro ref href
      have he : ref = W32 1 := by simpa only [row, List.mem_singleton] using href
      subst ref
      constructor <;> word
    · intro ref href
      have he : ref = W32 1 := by simpa only [row, List.mem_singleton] using href
      subst ref
      simp only [List.length_singleton]
      constructor <;> word
  wp_start_folded as ⟨Hgraph, Hflow, Hnodes, Hblocks, Hoperations, Hsuccessors⟩
  wp_apply wp_validateFlow_valid receiver graph flow nodes [row] dq htables
    (by rw [hentry]; simp only [List.length_singleton]; constructor <;> word) hvalid
    $$ [Hgraph Hflow Hnodes Hblocks Hoperations Hsuccessors]
    with ⟨Hgraph, Hflow, Hnodes, Hblocks, Hrows⟩
  · simp only [constructorFlowRowsOwn, List.map_cons, List.map_nil, row]
    iframe
  simp only [constructorFlowRowsOwn, List.map_cons, List.map_nil, row] at *
  icases Hrows with ⟨Hoperations, Hsuccessors, _⟩
  wp_end

end execution
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
