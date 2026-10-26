module
public import Gstyle.Proof.Worklist.ConstructorAllocationSyntax
public import Gstyle.Proof.Worklist.ConstructorValidation
public import Gstyle.Proof.Worklist.ConstructorLayout
public import Gstyle.Proof.Worklist.ConstructorPackages
@[expose] public section
noncomputable section
set_option maxHeartbeats 0
set_option maxRecDepth 100000
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

/-- The composite literal sets five fields; the other four retain their Go zero value. -/
def constructorAllocatedView [FfiSyntax] (graph : Loc)
    (operands unstable kinds tracked : GoSlice) : graphView :=
  { graph' := graph, operands' := operands, unstable' := unstable, kinds' := kinds,
    tracked' := tracked, fields' := slice.nil, paths' := map.nil, fieldRoots' := map.nil,
    dereferences' := map.nil }

section execution
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [pdg.query.Assumptions] [graphView.TypeAssumptions]
variable [pdg.Graph.TypeAssumptions] [pdg.AttributeTables.TypeAssumptions]
variable [pdg.Producer.TypeAssumptions] [pdg.Function.TypeAssumptions] [pdg.Span.TypeAssumptions]
variable [pdg.NormalFlow.TypeAssumptions] [pdg.FlowBlock.TypeAssumptions]
variable [pdg.CountFact.TypeAssumptions] [pdg.Conformance.TypeAssumptions] [pdg.Fact.TypeAssumptions]
variable [pdg.ConformanceStatus.TypeAssumptions]
variable [FuncUnfold newGraphView [] newGraphView.impl]
variable [FuncUnfold validateFlow [] validateFlow.impl]
variable [FuncUnfold validateBlock [] validateBlock.impl]
variable [FuncUnfold validateRefs [] validateRefs.impl]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

/-- The input arrays and flow rows survive validation and allocation. -/
def constructorAllocationInput (graphPtr : Loc) (graph : pdg.Graph) (flow : pdg.NormalFlow)
    (nodes : List pdg.Node) (symbols : List pdg.Symbol) (rows : List ConstructorFlowRow) : IProp GF := iprop(
  graphPtr ↦□ graph ∗ graph.ValueFlow' ↦□ flow ∗ graph.Nodes' ↦*□ nodes ∗
  graph.Symbols' ↦*□ symbols ∗ flow.Blocks' ↦*□ rows.map ConstructorFlowRow.block ∗
  constructorFlowRowsOwn DFrac.discard rows)

/-- Fresh mutable locals, receiver and four arrays, with the represented inputs retained. -/
def constructorAllocationOwn (graphPtr : Loc) (graph : pdg.Graph) (flow : pdg.NormalFlow)
    (nodes : List pdg.Node) (symbols : List pdg.Symbol) (rows : List ConstructorFlowRow)
    (graphLocal viewLocal receiver : Loc) (operands unstable kinds tracked : GoSlice) : IProp GF := iprop(
  graphLocal ↦ graphPtr ∗ viewLocal ↦ receiver ∗
  receiver ↦ constructorAllocatedView graphPtr operands unstable kinds tracked ∗
  operands ↦* List.replicate nodes.length (zero_val (GoArray w32 2)) ∗
  unstable ↦* List.replicate symbols.length false ∗
  kinds ↦* List.replicate symbols.length (go!"") ∗
  tracked ↦* List.replicate symbols.length false ∗
  ownSliceCap (GoArray w32 2) operands (DFrac.own 1) ∗ ownSliceCap Bool unstable (DFrac.own 1) ∗
  ownSliceCap GoString kinds (DFrac.own 1) ∗ ownSliceCap Bool tracked (DFrac.own 1) ∗
  constructorAllocationInput graphPtr graph flow nodes symbols rows)

/-- Execute the real validator and allocation prefix, then use the exact remaining
    constructor expression. No constructor output or child Hoare contract is assumed. -/
theorem wp_newGraphView_allocations (graphPtr : Loc) (graph : pdg.Graph) (flow : pdg.NormalFlow)
    (nodes : List pdg.Node) (symbols : List pdg.Symbol) (rows : List ConstructorFlowRow)
    (htables : graph.Tables' ≠ null)
    (hentry : 0 < uint.Z flow.Entry' ∧ uint.Z flow.Entry' ≤ rows.length)
    (hrows : constructorFlowRowsValid nodes.length rows)
    (hnodes : nodes.length ≤ 2^45) (hsymbols : symbols.length ≤ 2^44)
    (Φ : val → IProp GF) :
    isPkgInit (PROP := IProp GF) pkg ∗ constructorAllocationInput graphPtr graph flow nodes symbols rows ∗
    (∀ (graphLocal viewLocal receiver : Loc) (operands unstable kinds tracked : GoSlice),
      constructorAllocationOwn graphPtr graph flow nodes symbols rows
        graphLocal viewLocal receiver operands unstable kinds tracked -∗
      WP (constructorSymbolSuffixRun graphLocal viewLocal) {{ Φ }}) ⊢
    WP (App (Val (@! newGraphView)) (Val #graphPtr)) {{ Φ }} := by
  iintro ⟨#Hpkg, Hinput, Hcontinue⟩
  iunfold constructorAllocationInput at Hinput
  icases Hinput with ⟨#Hgraph, #Hflow, #Hnodes, #Hsymbols, #Hblocks, Hrows⟩
  ihave %Hnlen := ownSlice_len $$ Hnodes
  ihave %Hslen := ownSlice_len $$ Hsymbols
  wp_func_call
  wp_call
  try wp_auto
  wp_apply wp_validateFlow_valid graphPtr graph flow nodes rows DFrac.discard htables hentry hrows
    $$ [Hgraph Hflow Hnodes Hblocks Hrows] with ⟨Hgraph, Hflow, Hnodes, Hblocks, Hrows⟩
  · iframe Hgraph Hflow Hnodes Hblocks Hrows
  try wp_auto
  wp_apply wp_slice_make2 (V := GoArray w32 2) graph.Nodes'.len (by
      intro _; simp only [go.typeSize_array w32 2 (by decide), go.typeSize_w32]; word)
    $$ [] with %operands ⟨Hoperands, HoperandsCap⟩
  · ipureintro; exact Hnlen.2
  try wp_auto
  wp_apply wp_slice_make2 (V := Bool) graph.Symbols'.len (by
      intro _; simp only [go.typeSize_bool]; word)
    $$ [] with %unstable ⟨Hunstable, HunstableCap⟩
  · ipureintro; exact Hslen.2
  try wp_auto
  wp_apply wp_slice_make2 (V := GoString) graph.Symbols'.len (by
      intro _; simp only [go.typeSize_string]; word)
    $$ [] with %kinds ⟨Hkinds, HkindsCap⟩
  · ipureintro; exact Hslen.2
  try wp_auto
  wp_apply +noauto wp_slice_make2 (V := Bool) graph.Symbols'.len (by
      intro _; simp only [go.typeSize_bool]; word)
    $$ [] with %tracked ⟨Htracked, HtrackedCap⟩
  · ipureintro; exact Hslen.2
  wp_pures
  wp_alloc receiver as Hview
  wp_pures
  wp_alloc viewLocal as HviewLocal
  wp_pures
  wp_store
  wp_pures
  ieval (change _ ⊢ WP (constructorSymbolSuffixRun g_ptr viewLocal) {{ Φ }})
  iapply Hcontinue $$ %g_ptr %viewLocal %receiver %operands %unstable %kinds %tracked
  iunfold constructorAllocationOwn, constructorAllocationInput, constructorAllocatedView
  isimp only [← Hnlen.1] at Hoperands
  isimp only [← Hslen.1] at Hunstable Hkinds Htracked
  isimp only [map.nil]
  isimp only [zero_val] at Hunstable Hkinds Htracked
  iframe g HviewLocal Hview Hoperands Hunstable Hkinds Htracked
    HoperandsCap HunstableCap HkindsCap HtrackedCap Hgraph Hflow Hnodes Hsymbols Hblocks Hrows

end execution
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
