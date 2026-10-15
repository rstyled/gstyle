module
public import Gstyle.Proof.Interval.ModelBounds
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
theorem wp_Lower (r : Range) (bound : w64) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! Lower)) (Val #r)) (Val #bound))
    {{ RET #(lowered r bound); True }} := by
  wp_start
  wp_auto
  cases hl : r.HasLower' <;> by_cases hc : sint.Z r.Lower' < sint.Z bound
  all_goals iterate 5 (try simp [lowered, hl, hc]; try wp_auto)
  all_goals wp_end

theorem wp_Upper (r : Range) (bound : w64) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! Upper)) (Val #r)) (Val #bound))
    {{ RET #(uppered r bound); True }} := by
  wp_start
  wp_auto
  cases hu : r.HasUpper' <;> by_cases hc : sint.Z r.Upper' > sint.Z bound
  all_goals iterate 5 (try simp [uppered, hu, hc]; try wp_auto)
  all_goals wp_end

theorem wp_LowerIntersection (r : Range) (bound : w64) (x : Int) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! Lower)) (Val #r)) (Val #bound))
    {{ (out : Range), RET #out;
      ⌜contains out x ↔ contains r x ∧ sint.Z bound ≤ x⌝ }} := by
  wp_start_folded
  wp_apply wp_Lower r bound
  wp_end
  ipureintro
  exact lower_intersection r bound x

theorem wp_UpperIntersection (r : Range) (bound : w64) (x : Int) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! Upper)) (Val #r)) (Val #bound))
    {{ (out : Range), RET #out;
      ⌜contains out x ↔ contains r x ∧ x ≤ sint.Z bound⌝ }} := by
  wp_start_folded
  wp_apply wp_Upper r bound
  wp_end
  ipureintro
  exact upper_intersection r bound x


end proof
end github_com.rstyled.gstyle.domain.runtimeguard.interval
end Perennial
