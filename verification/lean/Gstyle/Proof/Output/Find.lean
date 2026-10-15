module
public import Gstyle.Proof.Output.Base
public import Gstyle.Proof.Output.Select
public import Gstyle.Proof.Output.Suppression
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.patterns.outputcheck

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics] [package_sem : Assumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.patterns.outputcheck
theorem wp_Find (sites suppressions : GoSlice) (records : List Site) (names : List GoString) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ sites ↦* records ∗ suppressions ↦* names }}
      (App (App (Val (@! Find)) (Val #sites)) (Val #suppressions))
    {{ (out : GoSlice), RET #out;
      sites ↦* records ∗ suppressions ↦* names ∗
        out ↦* (if go!"unchecked_output" ∈ names then [] else records.filter Site.DiscardedOutput') }} := by
  wp_start as ⟨Hsites, Hnames⟩
  wp_auto
  wp_apply wp_hasSuppression suppressions names $$ [Hnames] with Hnames
  · iframe
  wp_apply wp_Select sites records (decide (go!"unchecked_output" ∈ names)) $$ [Hsites] with %out ⟨Hsites, Hout⟩
  · iframe
  wp_end
  simp only [decide_eq_true_eq]
  iframe


end proof
end github_com.rstyled.gstyle.domain.patterns.outputcheck
end Perennial
