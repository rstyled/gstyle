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
variable [go.Semantics] [graphView.TypeAssumptions] [Value.TypeAssumptions] [pdg.query.Assumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

attribute [local irreducible] decimalInt64 signedDecimal decimalMagnitude decimalDigits constantRecord constantResult parseInt64Result

set_option maxHeartbeats 2400000 in
theorem wp_constantValue_contract (receiver : Loc) (ref : w32)
    (facts : GoString → GoString) (resources : IProp GF) [Persistent resources]
    (hparse : parseInt64Contract (GF := GF))
    (hfact : ∀ key : GoString,
      {{ isPkgInit (PROP := IProp GF) pkg ∗ resources }}
        (App (App (Val #(methods (go.PointerType graphView.ty) go!"fact" #receiver)) (Val #ref)) (Val #key))
      {{ RET #(facts key); resources }}) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ resources }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"constantValue" #receiver)) (Val #ref))
    {{ RET #(constantResult facts); resources }} := by
  wp_start as #Hresources
  try wp_auto
  wp_apply hfact (go!"non-nil") $$ [Hresources] with Hresources
  · iframe Hresources
  try wp_auto
  wp_apply hfact (go!"nil") $$ [Hresources] with Hresources
  · iframe Hresources
  try wp_auto
  wp_apply hfact (go!"integer") $$ [Hresources] with Hresources
  · iframe Hresources
  try wp_auto
  wp_apply hparse (facts (go!"integer")) with %integer %err %hresult
  try wp_auto
  cases err with
  | nil =>
    obtain ⟨z, hz, hv, _⟩ := parseInt64Result_success _ _ hresult
    try wp_auto
    wp_apply hfact (go!"literal-extent") $$ [Hresources] with Hresources
    · iframe Hresources
    try wp_auto
    wp_apply hparse (facts (go!"literal-extent")) with %extent %extentErr %hextent
    cases extentErr with
    | nil =>
      obtain ⟨ez, hez, hev, _⟩ := parseInt64Result_success (facts (go!"literal-extent")) extent hextent
      simp only [constantResult, constantRecord, hz, hez]
      try simp only [← hv]
      try simp only [← hev]
      try wp_auto
      wp_end
      all_goals iexact Hresources
    | ok extentErr =>
      have hez := parseInt64Result_error _ _ _ hextent (by intro h; cases h)
      simp only [constantResult, constantRecord, hz, hez]
      try simp only [← hv]
      try simp only [← hev]
      try wp_auto
      wp_end
      all_goals iexact Hresources

  | ok err =>
    have hz := parseInt64Result_error _ _ _ hresult (by intro h; cases h)
    try wp_auto
    wp_apply hfact (go!"literal-extent") $$ [Hresources] with Hresources
    · iframe Hresources
    try wp_auto
    wp_apply hparse (facts (go!"literal-extent")) with %extent %extentErr %hextent
    cases extentErr with
    | nil =>
      obtain ⟨ez, hez, hev, _⟩ := parseInt64Result_success (facts (go!"literal-extent")) extent hextent
      simp only [constantResult, constantRecord, hz, hez]
      try simp only [← hv]
      try simp only [← hev]
      try wp_auto
      wp_end
      all_goals iexact Hresources
    | ok extentErr =>
      have hez := parseInt64Result_error _ _ _ hextent (by intro h; cases h)
      simp only [constantResult, constantRecord, hz, hez]
      try simp only [← hv]
      try simp only [← hev]
      try wp_auto
      wp_end
      all_goals iexact Hresources


end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
