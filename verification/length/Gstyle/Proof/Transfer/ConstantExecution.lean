module
public import Gstyle.Proof.Transfer.ParseIntContract
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
  wp_start as #Hsnapshot
  try wp_auto
  wp_apply wp_fact_snapshot receiver s ref (go!"non-nil") hvalid halloc $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  try wp_auto
  wp_apply wp_fact_snapshot receiver s ref (go!"nil") hvalid halloc $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  try wp_auto
  wp_apply wp_fact_snapshot receiver s ref (go!"integer") hvalid halloc $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  try wp_auto
  wp_apply hparse (snapshotFact s ref (go!"integer")) with %integer %err %hresult
  try wp_auto
  cases err with
  | nil =>
    obtain ⟨z, hz, hv, _⟩ := parseInt64Result_success _ _ hresult
    try wp_auto
    wp_apply wp_fact_snapshot receiver s ref (go!"literal-extent") hvalid halloc $$ [Hsnapshot] with Hsnapshot
    · iframe Hsnapshot
    try wp_auto
    wp_apply hparse (snapshotFact s ref (go!"literal-extent")) with %extent %extentErr %hextent
    cases extentErr with
    | nil =>
      obtain ⟨ez, hez, hev, _⟩ := parseInt64Result_success (snapshotFact s ref (go!"literal-extent")) extent hextent
      simp only [snapshotConstant, constantRecord, hz, hez]
      try simp only [← hv]
      try simp only [← hev]
      try wp_auto
      wp_end
      all_goals iexact Hsnapshot
    | ok extentErr =>
      have hez := parseInt64Result_error _ _ _ hextent (by intro h; cases h)
      simp only [snapshotConstant, constantRecord, hz, hez]
      try simp only [← hv]
      try simp only [← hev]
      try wp_auto
      wp_end
      all_goals iexact Hsnapshot

  | ok err =>
    have hz := parseInt64Result_error _ _ _ hresult (by intro h; cases h)
    try wp_auto
    wp_apply wp_fact_snapshot receiver s ref (go!"literal-extent") hvalid halloc $$ [Hsnapshot] with Hsnapshot
    · iframe Hsnapshot
    try wp_auto
    wp_apply hparse (snapshotFact s ref (go!"literal-extent")) with %extent %extentErr %hextent
    cases extentErr with
    | nil =>
      obtain ⟨ez, hez, hev, _⟩ := parseInt64Result_success (snapshotFact s ref (go!"literal-extent")) extent hextent
      simp only [snapshotConstant, constantRecord, hz, hez]
      try simp only [← hv]
      try simp only [← hev]
      try wp_auto
      wp_end
      all_goals iexact Hsnapshot
    | ok extentErr =>
      have hez := parseInt64Result_error _ _ _ hextent (by intro h; cases h)
      simp only [snapshotConstant, constantRecord, hz, hez]
      try simp only [← hv]
      try simp only [← hev]
      try wp_auto
      wp_end
      all_goals iexact Hsnapshot


end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
