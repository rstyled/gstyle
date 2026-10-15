module
public import Gstyle.Proof.Interval.ModelRisk
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard.interval

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics] [package_sem : Assumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval
theorem wp_NegativeIndex (index upperChecked : Bool) (r : Range) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (App (Val (@! NegativeIndex)) (Val #index)) (Val #upperChecked)) (Val #r))
    {{ RET #(risk index upperChecked r); True }} := by
  wp_start
  wp_auto
  cases index <;> cases upperChecked <;> cases hl : r.HasLower' <;>
    by_cases hn : sint.Z r.Lower' < 0
  all_goals iterate 5 (try simp [risk, hl, hn]; try wp_auto)
  all_goals wp_end


theorem wp_NegativeDetected (r : Range) (x : Int) (h : contains r x) (hx : x < 0) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (App (Val (@! NegativeIndex)) (Val #true)) (Val #true)) (Val #r))
    {{ RET #true; True }} := by
  wp_start_folded
  wp_apply wp_NegativeIndex true true r
  rw [negative_detected r x h hx]
  wp_end


end proof
end github_com.rstyled.gstyle.domain.runtimeguard.interval
end Perennial
