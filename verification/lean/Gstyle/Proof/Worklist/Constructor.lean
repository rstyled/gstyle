module
public import Gstyle.Proof.Worklist.ConstructorInput
public import Gstyle.Proof.Worklist.ConstructorTrackingBindings
@[expose] public section
noncomputable section
set_option maxHeartbeats 0
set_option maxRecDepth 100000
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [FfiSyntax]

theorem constructorSnapshot_tracking (input : ConstructorInput) (view : graphView)
    (valid : ScalarConstructorInput input) : ScalarTrackingData (input.snapshot view) := by
  refine ⟨constructorSnapshot_tracking_roots input view valid,
    constructorSnapshot_tracking_symbols input view valid, valid.nodeBound, ?_, ?_⟩
  · simpa [ConstructorInput.snapshot, constructorKinds] using valid.symbolBound
  · intro ref _ write hw
    exact constructorSnapshot_writeBounds input view valid ref write hw

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
variable [pdg.Symbol.TypeAssumptions] [pdg.Role.TypeAssumptions] [pdg.Edge.TypeAssumptions] [pdg.EdgeKind.TypeAssumptions]
variable [pdg.NormalFlow.TypeAssumptions] [pdg.FlowBlock.TypeAssumptions]
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
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

/-- Successful actual construction yields the snapshot and its owned empty roots. -/
theorem wp_newGraphView_scalar (graphPtr : Loc) (input : ConstructorInput)
    (inverseLoc : Loc) (inverseMap : GMap GoString GoString) (valid : ScalarConstructorInput input) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ constructorInputOwn graphPtr input ∗
       globalAddr inverse ↦□ inverseLoc ∗ inverseLoc ↦${DFrac.discard} inverseMap }}
      (App (Val (@! newGraphView)) (Val #graphPtr))
    {{ (receiver : Loc), RET (PairV #receiver #interface.nil);
       ∃ s : QuerySnapshot, querySnapshotOwn receiver s ∗ s.view.fieldRoots' ↦$ (∅ : GMap w32 w32) ∗
         ⌜ValueSnapshotValid s ∧ s.graph = input.graph ∧ s.view.graph' = graphPtr⌝ }} := by
  wp_start_folded as ⟨#Hinput, #HinvAddr, #HinvMap⟩
  ihave #Hctx := Hinput
  iunfold constructorInputOwn at Hctx
  icases Hctx with ⟨Hallocation, Hdata, Hedges, Hentries⟩
  ihave #Hmeta := Hdata
  iunfold constructorGraphOwn at Hmeta
  icases Hmeta with ⟨Hgraph, Hnodes, Hattributes, Hreferences, Haccesses, Hextensions,
    HrefRows, HaccessRows, HextensionRows, Htable, Hstrings⟩
  isimp only [ConstructorInput.metadata, ConstructorInput.snapshot] at Hgraph Hnodes Hattributes Hreferences Haccesses Hextensions HrefRows HaccessRows HextensionRows Htable Hstrings
  iapply wp_newGraphView_allocations graphPtr input.graph input.flow input.nodes input.symbols input.flowRows
    valid.tables valid.entry valid.flow (by have h := valid.nodeBound; omega)
    (by have h := valid.symbolBound; omega) Φ
  iframe Hpkg Hallocation
  iintro %graphLocal %viewLocal %receiver %operands %unstable %kinds %tracked Hallocated
  iunfold constructorAllocationOwn at Hallocated
  icases Hallocated with ⟨Hg, Hv, Hview, Hoperands, Hunstable, Hkinds, Htracked,
    HoperandsCap, HunstableCap, HkindsCap, HtrackedCap, _⟩
  rw [constructor_symbol_suffix_run_exact]
  wp_apply +noauto wp_constructor_symbols_raw graphLocal viewLocal receiver
    (constructorAllocatedView graphPtr operands unstable kinds tracked)
    input.graph input.table input.strings input.symbols input.symbolExtensions
    $$ [Hg Hv Hview Hkinds Hunstable] with ⟨Hg, Hv, Hview, Hkinds, Hunstable⟩
  · isimp only [constructorAllocatedView] at Hview
    isimp only [constructorAllocatedView]
    iframe Hg Hv Hview Hgraph Htable Hstrings Hentries Hkinds Hunstable
    ihave #Ha := Hallocation
    iunfold constructorAllocationInput at Ha
    icases Ha with ⟨_, _, _, Hsymbols, _, _⟩
    iexact Hsymbols
  wp_pures
  ieval (change _ ⊢ WP (App (Val exceptionDo) (constructorAfterSymbolsRawRun graphLocal viewLocal)) {{ Φ }})
  rw [constructor_after_symbols_run_exact]
  wp_apply +noauto wp_constructor_edges_filtered_raw graphLocal viewLocal receiver
    (constructorAllocatedView graphPtr operands unstable kinds tracked)
    input.graph input.table input.strings input.nodes input.edges
    (List.replicate input.nodes.length (zero_val (GoArray w32 2)))
    (constructorRowsValid_zero _) $$ [Hg Hv Hview Hoperands]
    with ⟨Hg, Hv, Hview, Hoperands⟩
  · isimp only [constructorAllocatedView] at Hview
    isimp only [constructorAllocatedView]
    iframe Hg Hv Hview Hgraph Htable Hstrings Hnodes Hedges Hoperands
  wp_pures
  ieval (change _ ⊢ WP (App (Val exceptionDo) (constructorAfterEdgesRawRun graphLocal viewLocal)) {{ Φ }})
  unfold constructorAfterEdgesRawRun constructorAfterEdges constructorAfterSymbols constructorSymbolSuffix
    constructorBody newGraphView.impl
  dsimp only [constructorSymbolSuffixSite, constructorSymbolSuffixStart, Option.or, Option.getD]
  try wp_auto
  wp_apply wp_fieldBindings_scalar receiver (constructorAllocatedView graphPtr operands unstable kinds tracked)
    (input.metadata graphPtr) rfl valid.nodeBound valid.outcomeBound
    (constructorSnapshot_fields input _ valid) $$ [Hview] with %fields %dereferences %paths %roots
      ⟨Hview, _, Hfields, Hdereferences, Hpaths, Hroots⟩
  · iframe Hview Hdata
  try wp_auto
  let finalView := constructorFieldsView
    (constructorAllocatedView graphPtr operands unstable kinds tracked) fields dereferences paths roots
  let s := input.snapshot finalView
  have HdataEq : constructorGraphOwn (GF := GF) (input.metadata graphPtr) = constructorGraphOwn s := by rfl
  ihave #HfinalData : (constructorGraphOwn s : IProp GF) $$ [Hdata]
  · rw [← HdataEq]; iexact Hdata
  wp_apply +noauto wp_trackBindings_scalar receiver finalView s inverseLoc inverseMap
    (DFrac.own 1) (DFrac.own 1) (DFrac.own 1) (DFrac.own 1) rfl
    (constructorSnapshot_tracking input finalView valid)
    $$ [Hview Hfields Hkinds Hoperands] with ⟨Hview, _, Hfields, Hkinds, Hoperands⟩
  · isimp only [s, finalView, ConstructorInput.snapshot, ConstructorInput.metadata, constructorFieldsView, constructorAllocatedView] at Hview HfinalData
    isimp only [s, finalView, ConstructorInput.snapshot, ConstructorInput.metadata, constructorFieldsView, constructorAllocatedView]
    iframe Hview Hfields Hkinds Hoperands HfinalData HinvAddr HinvMap
  isimp only [finalView, constructorFieldsView, constructorAllocatedView] at Hview
  isimp only [s, ConstructorInput.snapshot, finalView, constructorFieldsView, constructorAllocatedView] at Hfields Hkinds Hoperands
  ipersist Hview
  ipersist Hfields
  ipersist Hkinds
  ipersist Hoperands
  ipersist Hunstable
  ipersist Htracked
  try wp_auto
  iapply HΦ $$ %receiver
  iexists s
  isplitl []
  · iunfold querySnapshotOwn
    isimp only [s, finalView, ConstructorInput.snapshot, ConstructorInput.metadata,
      constructorFieldsView, constructorAllocatedView]
    iframe Hview Hfields Hoperands Hkinds Hunstable Htracked Hgraph Hnodes Hattributes
      Hreferences Haccesses Hextensions HrefRows HaccessRows HextensionRows Htable Hstrings
  isimp only [s, finalView, ConstructorInput.snapshot, constructorFieldsView, constructorAllocatedView]
  iframe Hroots
  ipureintro
  exact ⟨constructorSnapshot_valid input finalView valid, trivial, trivial⟩

end execution
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
