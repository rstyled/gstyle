module
public import Gstyle.Proof.Transfer.ConstantDispatch
public import Gstyle.Proof.Transfer.ValueModel
@[expose] public section
noncomputable section
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
variable [FuncUnfold pdg.listAt [pdg.Ref.ty] (pdg.listAt.impl pdg.Ref.ty)]
variable [FuncUnfold pdg.listAt [pdg.Access.ty] (pdg.listAt.impl pdg.Access.ty)]
variable [FuncUnfold pdg.listAt [pdg.Extension.ty] (pdg.listAt.impl pdg.Extension.ty)]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

variable [pdg.Access.TypeAssumptions] [pdg.Tables.TypeAssumptions] [pdg.Extension.TypeAssumptions]
variable [FuncUnfold directBindingRead [] directBindingRead.impl]
variable [FuncUnfold operatorText [] operatorText.impl]


variable [FuncUnfold extension [] extension.impl]

variable [Value.TypeAssumptions]

attribute [local irreducible] snapshotFact decimalInt64 signedDecimal decimalMagnitude decimalDigits snapshotConstant constantRecord parseInt64Result

set_option maxHeartbeats 2400000 in
theorem wp_constantValue_snapshot (receiver : Loc) (s : QuerySnapshot) (ref : w32)
    (hvalid : validSymbolRef s ref)
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46)
    (hparse : parseInt64Contract (GF := GF)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ querySnapshotOwn receiver s }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"constantValue" #receiver)) (Val #ref))
    {{ RET #(snapshotConstant s ref); querySnapshotOwn receiver s }} := by
  simpa only [snapshotConstant, constantResult] using wp_constantValue_contract receiver ref (snapshotFact s ref) (querySnapshotOwn receiver s) hparse
    (fun key => wp_fact_snapshot receiver s ref key hvalid halloc)

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
