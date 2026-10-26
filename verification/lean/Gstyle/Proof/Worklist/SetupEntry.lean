module
public import Gstyle.Proof.Worklist.Setup
public import Gstyle.Proof.Worklist.Constructor
@[expose] public section
noncomputable section
set_option maxHeartbeats 0
set_option maxRecDepth 100000
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
section proof
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
variable [Value.TypeAssumptions] [State.TypeAssumptions]
variable [analysis.TypeAssumptions] [pdg.NormalFlow.TypeAssumptions] [pdg.FlowBlock.TypeAssumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

/-- A composition helper for the actual constructor call. Its constructor contract
    is a parameter; the final entry theorem supplies the checked implementation. -/
theorem wp_solve_setup_entry_bind (graph : Loc) (g : pdg.Graph)
    (flow : pdg.NormalFlow) (blocks : List pdg.FlowBlock)
    (entry : w32) (hentry : flow.Entry' = entry)
    (href : 0 < uint.Z entry ∧ uint.Z entry ≤ blocks.length)
    (halloc : blocks.length ≤ 2^45)
    (P : IProp GF) (Q : Loc → IProp GF)
    (hconstructor :
      {{ isPkgInit (PROP := IProp GF) pkg ∗ P }}
        (App (Val (@! newGraphView)) (Val #graph))
      {{ (view : Loc), RET (PairV #view #interface.nil); Q view }}) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ graph ↦□ g ∗
       g.ValueFlow' ↦□ flow ∗ flow.Blocks' ↦*□ blocks ∗ P }}
      (solveSetupEntry graph)
    {{ (receiver : Loc), RET (PairV #receiver #interface.nil);
       ∃ (view : Loc) (inputs flags queue : GoSlice) (pointers : List Loc),
         Q view ∗ workStateOwn receiver view inputs flags queue pointers
           (initialWorkState blocks.length entry) }} := by
  wp_start_folded as ⟨#Hgraph, #Hflow, #Hblocks, HP⟩
  ihave %HgraphNonNull := typedPointsto_not_null _ _ _ $$ Hgraph
  ihave %HflowNonNull := typedPointsto_not_null _ _ _ $$ Hflow
  unfold solveSetupEntry solveWithCursorTail solve.impl
  dsimp only [setupContext, cursorHead, cursorStart]
  wp_call
  try wp_auto
  rw [decide_eq_false HgraphNonNull]
  try wp_auto
  rw [decide_eq_false HflowNonNull]
  try wp_auto
  wp_apply +noauto hconstructor $$ [HP] with %view HQ
  · iframe HP
  wp_pures
  wp_alloc errLocal as Herr
  wp_pures
  wp_alloc viewLocal as Hview
  wp_pures
  wp_store
  wp_pures
  wp_store
  wp_pures
  wp_load
  wp_pures
  ieval (change _ ⊢ WP (solveSetupRun g_ptr viewLocal) {{ Φ }})
  unfold solveSetupRun
  wp_apply wp_solve_setup_raw g_ptr viewLocal graph view g flow blocks entry hentry href halloc
      $$ [g Hview] with %receiver ⟨Hg, Hview, %inputs, %flags, %queue, %pointers, Hwork⟩
  · iframe g Hview Hgraph Hflow Hblocks
  try wp_auto
  iapply HΦ $$ %receiver
  iexists view, inputs, flags, queue, pointers
  iframe HQ Hwork

section constructor
variable [pdg.Symbol.TypeAssumptions] [pdg.Role.TypeAssumptions]
variable [pdg.Edge.TypeAssumptions] [pdg.EdgeKind.TypeAssumptions]
variable [FuncUnfold newGraphView [] newGraphView.impl]
variable [FuncUnfold validateFlow [] validateFlow.impl]
variable [FuncUnfold validateBlock [] validateBlock.impl]
variable [FuncUnfold validateRefs [] validateRefs.impl]
variable [FuncUnfold extension [] extension.impl]
variable [FuncUnfold sourceOperandEdge [] sourceOperandEdge.impl]
variable [FuncUnfold operandReferences [] operandReferences.impl]
variable [FuncUnfold directBindingRead [] directBindingRead.impl]
variable [FuncUnfold operatorText [] operatorText.impl]
variable [FuncUnfold decodeFact [] decodeFact.impl]
variable [FuncUnfold pdg.listAt [pdg.Ref.ty] (pdg.listAt.impl pdg.Ref.ty)]
variable [FuncUnfold pdg.listAt [pdg.Access.ty] (pdg.listAt.impl pdg.Access.ty)]
variable [FuncUnfold pdg.listAt [pdg.Extension.ty] (pdg.listAt.impl pdg.Extension.ty)]

/-- The actual solver entry constructs its snapshot, empty roots and exact initial
    work state from represented scalar graph data, stopping at the cursor boundary. -/
theorem wp_solve_setup_entry (graph : Loc) (input : ConstructorInput)
    (inverseLoc : Loc) (inverseMap : GMap GoString GoString)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (valid : ScalarConstructorInput input) (halloc : input.flowRows.length ≤ 2^45) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ constructorInputOwn graph input ∗
       globalAddr inverse ↦□ inverseLoc ∗ inverseLoc ↦${DFrac.discard} inverseMap }}
      (solveSetupEntry graph)
    {{ (receiver : Loc), RET (PairV #receiver #interface.nil);
       ∃ (view : Loc) (s : QuerySnapshot) (inputs flags queue : GoSlice) (pointers : List Loc),
         querySnapshotOwn view s ∗ s.view.fieldRoots' ↦$ (∅ : GMap w32 w32) ∗
         workStateOwn receiver view inputs flags queue pointers
           (initialWorkState input.flowRows.length input.flow.Entry') ∗
         ⌜ValueSnapshotValid s ∧ s.graph = input.graph ∧ s.view.graph' = graph ∧
           scalarWorkInvariant s kind changing 0
             (initialWorkState input.flowRows.length input.flow.Entry') ∧
           (initialWorkState input.flowRows.length input.flow.Entry').inputs.length = input.flowRows.length ∧
           (initialWorkState input.flowRows.length input.flow.Entry').flags.length = input.flowRows.length ∧
           reachableQueue (initialWorkState input.flowRows.length input.flow.Entry')⌝ }} := by
  wp_start_folded as ⟨#Hinput, #HinvAddr, #HinvMap⟩
  ihave ⟨#Hgraph, #Hflow, #Hblocks⟩ := constructorInputOwn_flow graph input $$ Hinput
  wp_apply wp_solve_setup_entry_bind graph input.graph input.flow
      (input.flowRows.map ConstructorFlowRow.block) input.flow.Entry' rfl
      (by simpa using valid.entry) (by simpa using halloc)
      (iprop(constructorInputOwn graph input ∗ globalAddr inverse ↦□ inverseLoc ∗
        inverseLoc ↦${DFrac.discard} inverseMap))
      (fun view => iprop(∃ s : QuerySnapshot, querySnapshotOwn view s ∗
        s.view.fieldRoots' ↦$ (∅ : GMap w32 w32) ∗
        ⌜ValueSnapshotValid s ∧ s.graph = input.graph ∧ s.view.graph' = graph⌝))
      (wp_newGraphView_scalar graph input inverseLoc inverseMap valid)
      $$ [] with %receiver ⟨%view, %inputs, %flags, %queue, %pointers, Hsnapshot, Hwork⟩
  · iframe Hgraph Hflow Hblocks Hinput HinvAddr HinvMap
  icases Hsnapshot with ⟨%s, Hsnapshot, Hroots, %Hvalid⟩
  isimp only [List.length_map] at Hwork
  iapply HΦ $$ %receiver
  iexists view, s, inputs, flags, queue, pointers
  iframe Hsnapshot Hroots Hwork
  ipureintro
  exact ⟨Hvalid.1, Hvalid.2.1, Hvalid.2.2,
    initialWorkState_scalarWorkInvariant s kind changing _ _ valid.entry,
    (initialWorkState_shape _ _).1, (initialWorkState_shape _ _).2,
    initialWorkState_reachableQueue _ _ valid.entry⟩

end constructor
end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
