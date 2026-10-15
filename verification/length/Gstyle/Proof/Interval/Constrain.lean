module
public import Gstyle.Proof.Interval.ModelConstrain
public import Gstyle.Proof.Interval.Strict
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
theorem wp_Constrain (r : Range) (operator : GoString) (bound : w64) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (App (Val (@! Constrain)) (Val #r)) (Val #operator)) (Val #bound))
    {{ RET #(constrained r operator bound); True }} := by
  wp_start
  wp_auto
  wp_if_destruct
  · simp [constrained]
    wp_apply wp_Lower r bound
    wp_end
  · wp_if_destruct
    · simp [constrained]
      wp_apply wp_StrictLower r bound
      wp_end
    · wp_if_destruct
      · simp [constrained]
        wp_apply wp_StrictUpper r bound
        wp_end
      · wp_if_destruct
        · simp [constrained]
          wp_apply wp_Upper r bound
          wp_end
        · wp_if_destruct
          · simp [constrained]
            wp_apply wp_Lower r bound
            wp_apply wp_Upper (lowered r bound) bound
            wp_end
          · simp_all [constrained]
            wp_end

theorem wp_ConstrainSound (r : Range) (operator : GoString) (bound : w64) (x : Int)
    (hr : contains r x) (hx : satisfies operator x (sint.Z bound)) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (App (Val (@! Constrain)) (Val #r)) (Val #operator)) (Val #bound))
    {{ (out : Range), RET #out; ⌜contains out x⌝ }} := by
  wp_start_folded
  wp_apply wp_Constrain r operator bound
  wp_end
  ipureintro
  exact constrain_sound r operator bound x hr hx


end proof
end github_com.rstyled.gstyle.domain.runtimeguard.interval
end Perennial
