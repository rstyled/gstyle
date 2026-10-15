module
public import Gstyle.Proof.Pdg.ReadonlyRows
public import Gstyle.Proof.Transfer.BindingRead
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

structure QuerySnapshot [FfiSyntax] where
  view : graphView
  graph : pdg.Graph
  table : pdg.Tables
  fields : List w32
  nodes : List pdg.Node
  attributes : List pdg.AttributeSet
  operands : List (GoArray w32 2)
  kinds : List GoString
  unstable : List Bool
  tracked : List Bool
  references : List GoSlice
  accesses : List GoSlice
  extensions : List GoSlice
  referenceRows : List (List w32)
  accessRows : List (List pdg.Access)
  extensionRows : List (List pdg.Extension)
  strings : List GoString

def validSymbolRef [FfiSyntax] (s : QuerySnapshot) (ref : w32) : Prop :=
  ref = W32 0 ∨ (0 < uint.Z ref ∧ uint.Z ref ≤ s.nodes.length)


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

def querySnapshotOwn (receiver : Loc) (s : QuerySnapshot) : IProp GF := iprop(
  receiver ↦{DFrac.discard} s.view ∗ s.view.graph' ↦{DFrac.discard} s.graph ∗
  s.view.fields' ↦*□ s.fields ∗ s.view.operands' ↦*□ s.operands ∗
  s.view.kinds' ↦*□ s.kinds ∗ s.view.unstable' ↦*□ s.unstable ∗ s.view.tracked' ↦*□ s.tracked ∗
  s.graph.Nodes' ↦*□ s.nodes ∗ s.graph.Attributes' ↦*□ s.attributes ∗
  s.graph.AttributeTables'.References' ↦*□ s.references ∗
  s.graph.AttributeTables'.Accesses' ↦*□ s.accesses ∗
  s.graph.AttributeTables'.Extensions' ↦*□ s.extensions ∗
  pdg.readonlyRows s.references s.referenceRows ∗ pdg.readonlyRows s.accesses s.accessRows ∗
  pdg.readonlyRows s.extensions s.extensionRows ∗
  s.graph.Tables' ↦{DFrac.discard} s.table ∗ s.table.Strings' ↦*□ s.strings)

instance querySnapshotOwn_persistent (receiver : Loc) (s : QuerySnapshot) :
    Persistent (querySnapshotOwn (GF := GF) receiver s) := by
  unfold querySnapshotOwn
  infer_instance

theorem querySnapshotOwn_rows (receiver : Loc) (s : QuerySnapshot) (ref : w32) :
    querySnapshotOwn (GF := GF) receiver s ⊢
      pdg.listRow s.references ref ↦*□ pdg.rowContents s.referenceRows ref ∗
      pdg.listRow s.accesses ref ↦*□ pdg.rowContents s.accessRows ref ∗
      pdg.listRow s.extensions ref ↦*□ pdg.rowContents s.extensionRows ref := by
  iintro Hsnapshot
  iunfold querySnapshotOwn at Hsnapshot
  icases Hsnapshot with ⟨_, _, _, _, _, _, _, _, _, _, _, _, Hrefs, Haccesses, Hextensions, _, _⟩
  ihave Href := pdg.readonlyRows_at_ref s.references s.referenceRows ref $$ Hrefs
  ihave Haccess := pdg.readonlyRows_at_ref s.accesses s.accessRows ref $$ Haccesses
  ihave Hextension := pdg.readonlyRows_at_ref s.extensions s.extensionRows ref $$ Hextensions
  iframe

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
