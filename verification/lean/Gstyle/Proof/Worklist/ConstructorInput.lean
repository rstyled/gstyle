module
public import Gstyle.Proof.Worklist.ConstructorAllocation
public import Gstyle.Proof.Worklist.ConstructorSymbols
public import Gstyle.Proof.Worklist.ConstructorFilteredEdges
public import Gstyle.Proof.Worklist.ConstructorFields
public import Gstyle.Proof.Worklist.ConstructorTrackingRoots
public import Gstyle.Proof.Worklist.ConstructorTrackingSymbols
@[expose] public section
noncomputable section
set_option maxHeartbeats 0
set_option maxRecDepth 100000
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

/-- Input graph contents. No receiver, constructor slice, or output invariant is supplied. -/
structure ConstructorInput [FfiSyntax] where
  graph : pdg.Graph
  flow : pdg.NormalFlow
  table : pdg.Tables
  nodes : List pdg.Node
  symbols : List pdg.Symbol
  symbolExtensions : List (List pdg.Extension)
  edges : List pdg.Edge
  flowRows : List ConstructorFlowRow
  attributes : List pdg.AttributeSet
  references : List GoSlice
  accesses : List GoSlice
  extensions : List GoSlice
  referenceRows : List (List w32)
  accessRows : List (List pdg.Access)
  extensionRows : List (List pdg.Extension)
  strings : List GoString

variable [FfiSyntax]

/-- Logical contents derived from the source graph, paired with the allocated view. -/
def ConstructorInput.snapshot (input : ConstructorInput) (view : graphView) : QuerySnapshot where
  view := view
  graph := input.graph
  table := input.table
  fields := List.replicate input.nodes.length (W32 0)
  nodes := input.nodes
  attributes := input.attributes
  operands := input.edges.foldl (constructorFilteredOperandSet input.strings input.nodes.length)
    (List.replicate input.nodes.length (zero_val (GoArray w32 2)))
  kinds := constructorKinds input.strings input.symbols input.symbolExtensions
  unstable := constructorUnstable input.strings input.symbols input.symbolExtensions
  tracked := List.replicate input.symbols.length false
  references := input.references
  accesses := input.accesses
  extensions := input.extensions
  referenceRows := input.referenceRows
  accessRows := input.accessRows
  extensionRows := input.extensionRows
  strings := input.strings

def ConstructorInput.metadata (input : ConstructorInput) (graphPtr : Loc) : QuerySnapshot :=
  input.snapshot { (zero_val graphView) with graph' := graphPtr }

def constructorAfterEdges [GoGlobalContext] : Expr :=
  match constructorAfterSymbols with
  | App (App _ (Rec _ BAnon body)) _ => body
  | _ => Val #()

def constructorAfterEdgesRawRun [GoGlobalContext] (graphLocal viewLocal : Loc) : Expr :=
  subst "g" #graphLocal (subst "view" #viewLocal constructorAfterEdges)

theorem constructor_after_symbols_run_exact [GoGlobalContext] (graphLocal viewLocal : Loc) :
    constructorAfterSymbolsRawRun graphLocal viewLocal =
      App (App (Val exceptionSeq) (Lam BAnon (constructorAfterEdgesRawRun graphLocal viewLocal)))
        (constructorEdgesRawRun graphLocal viewLocal) := by rfl

/-- Structural source restrictions for the first scalar constructor proof. -/
structure ScalarConstructorInput (input : ConstructorInput) : Prop where
  tables : input.graph.Tables' ≠ null
  entry : 0 < uint.Z input.flow.Entry' ∧ uint.Z input.flow.Entry' ≤ input.flowRows.length
  flow : constructorFlowRowsValid input.nodes.length input.flowRows
  nodeBound : input.nodes.length < 2^32
  symbolBound : input.symbols.length ≤ 2^43
  outcomeBound : ∀ a ∈ input.attributes,
    (pdg.rowContents input.referenceRows a.Outcomes').length ≤ 2^46
  emptyFacts : ∀ a ∈ input.attributes, ∀ name ∈
      [go!"field-path", go!"dereference-paths", go!"index", go!"dereference", go!"nil"],
    extensionValue input.strings name (pdg.rowContents input.extensionRows a.Extensions') = go!""
  noParentheses : ∀ a ∈ input.attributes,
    (if a.Operator'.Status' = W8 1 then pdg.tableText input.strings a.Operator'.Value' else go!"") ≠ go!"parentheses"
  noCalls : ∀ node ∈ input.nodes, node.Category' ≠ W8 5
  writeBounds : ∀ a ∈ input.attributes, ∀ write ∈ pdg.rowContents input.accessRows a.Writes',
    trackingSymbolBound input.symbols.length write.Symbol'

theorem constructorSnapshot_rows (input : ConstructorInput) (view : graphView)
    (valid : ScalarConstructorInput input) :
    constructorRowsValid input.nodes.length (input.snapshot view).operands := by
  exact constructorRowsValid_filtered_zero input.nodes.length input.strings input.edges

theorem constructorSnapshot_valid (input : ConstructorInput) (view : graphView)
    (valid : ScalarConstructorInput input) : ValueSnapshotValid (input.snapshot view) := by
  apply constructorValueSnapshotValid
  · simp [ConstructorInput.snapshot]
  · exact constructorSnapshot_rows input view valid
  · simp [ConstructorInput.snapshot, constructorKinds, constructorUnstable]
  · exact valid.outcomeBound

theorem constructorSnapshot_fields (input : ConstructorInput) (view : graphView)
    (valid : ScalarConstructorInput input) : scalarFieldMetadata (input.snapshot view) := by
  intro a ha
  exact ⟨valid.emptyFacts a ha (go!"field-path") (by simp),
    valid.emptyFacts a ha (go!"dereference-paths") (by simp)⟩

theorem constructorSnapshot_record (input : ConstructorInput) (view : graphView) (ref : w32) :
    snapshotRecord (input.snapshot view) ref = zero_val pdg.AttributeSet ∨
      snapshotRecord (input.snapshot view) ref ∈ input.attributes := by
  unfold snapshotRecord
  split
  · exact Or.inl rfl
  · split
    · exact Or.inl rfl
    · split
      · exact Or.inl rfl
      · rename_i node _ _
        dsimp only [ConstructorInput.snapshot]
        cases h : input.attributes[(uint.Z node.Attributes' - 1).toNat]? with
        | none => exact Or.inl rfl
        | some a => exact Or.inr (List.mem_of_getElem? h)

theorem constructorSnapshot_tracking_roots (input : ConstructorInput) (view : graphView)
    (valid : ScalarConstructorInput input) : ScalarTrackingRoots (input.snapshot view) := by
  have rows := constructorSnapshot_rows input view valid
  refine ⟨rows.1, ?_, ?_, ?_, ?_, ?_, valid.noCalls, valid.outcomeBound⟩
  · intro row hr; exact (rows.2 row hr).1
  · intro row hr child hc; exact (rows.2 row hr).2 child hc
  · intro ref _
    rw [snapshotFact_empty_of_extensions (input.snapshot view) ref (go!"index")
      (fun a ha => valid.emptyFacts a ha (go!"index") (by simp))]
    decide
  · intro ref _
    rw [snapshotFact_empty_of_extensions (input.snapshot view) ref (go!"dereference")
      (fun a ha => valid.emptyFacts a ha (go!"dereference") (by simp))]
    decide
  · intro ref _
    rw [snapshotFact_empty_of_extensions (input.snapshot view) ref (go!"nil")
      (fun a ha => valid.emptyFacts a ha (go!"nil") (by simp))]
    decide

theorem constructorSnapshot_tracking_symbols (input : ConstructorInput) (view : graphView)
    (valid : ScalarConstructorInput input) : ScalarTrackingSymbols (input.snapshot view) :=
  ⟨rfl, valid.noParentheses, valid.outcomeBound⟩

theorem constructorSnapshot_writeBounds (input : ConstructorInput) (view : graphView)
    (valid : ScalarConstructorInput input) (ref : w32)
    (write : pdg.Access) (hw : write ∈ pdg.rowContents input.accessRows (snapshotRecord (input.snapshot view) ref).Writes') :
    trackingSymbolBound (input.snapshot view).kinds.length write.Symbol' := by
  rcases constructorSnapshot_record input view ref with hz | ha
  · rw [hz] at hw
    simp [pdg.rowContents, show (zero_val pdg.AttributeSet).Writes' = W32 0 from rfl] at hw
  · simpa [ConstructorInput.snapshot, constructorKinds] using valid.writeBounds _ ha write hw

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

instance constructorFlowRowsOwn_persistent (rows : List ConstructorFlowRow) :
    Persistent (constructorFlowRowsOwn (GF := GF) DFrac.discard rows) := by
  induction rows with
  | nil => unfold constructorFlowRowsOwn; infer_instance
  | cons row rows ih => unfold constructorFlowRowsOwn; infer_instance

def constructorInputOwn (graphPtr : Loc) (input : ConstructorInput) : IProp GF := iprop(
  constructorAllocationInput graphPtr input.graph input.flow input.nodes input.symbols input.flowRows ∗
  constructorGraphOwn (input.metadata graphPtr) ∗ input.graph.Edges' ↦*□ input.edges ∗
  pdg.readonlyRows (input.symbols.map pdg.Symbol.Extensions') input.symbolExtensions)

instance constructorInputOwn_persistent (graphPtr : Loc) (input : ConstructorInput) :
    Persistent (constructorInputOwn (GF := GF) graphPtr input) := by
  unfold constructorInputOwn constructorAllocationInput
  infer_instance

theorem constructorInputOwn_flow (graphPtr : Loc) (input : ConstructorInput) :
    constructorInputOwn (GF := GF) graphPtr input ⊢ graphPtr ↦□ input.graph ∗
      input.graph.ValueFlow' ↦□ input.flow ∗
      input.flow.Blocks' ↦*□ input.flowRows.map ConstructorFlowRow.block := by
  iintro Hinput
  iunfold constructorInputOwn, constructorAllocationInput at Hinput
  icases Hinput with ⟨⟨Hgraph, Hflow, _, _, Hblocks, _⟩, _, _, _⟩
  iframe

end execution
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
