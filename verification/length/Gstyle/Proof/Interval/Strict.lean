module
public import Gstyle.Proof.Interval.ModelStrict
public import Gstyle.Proof.Interval.Bounds
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
theorem wp_StrictLower (r : Range) (bound : w64) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! StrictLower)) (Val #r)) (Val #bound))
    {{ RET #(strictLowered r bound); True }} := by
  wp_start
  wp_auto
  wp_if_destruct
  · have hlimit : (~~~(W64 0) >>> W64 1 : w64) = 9223372036854775807#64 := by word
    simp only [strictLowered, hlimit, ↓reduceIte]
    wp_end
  · have hb : bound ≠ 9223372036854775807#64 := by word
    simp only [strictLowered, if_neg hb]
    try wp_auto
    have h1 : (W64 1 : w64) = 1#64 := by word
    rw [← h1]
    wp_apply wp_Lower r (bound + W64 1)
    wp_end

theorem wp_StrictUpper (r : Range) (bound : w64) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! StrictUpper)) (Val #r)) (Val #bound))
    {{ RET #(strictUppered r bound); True }} := by
  wp_start
  wp_auto
  wp_if_destruct
  · have hlimit : (- (~~~(W64 0) >>> W64 1) - W64 1 : w64) = 9223372036854775808#64 := by word
    simp only [strictUppered, hlimit, ↓reduceIte]
    wp_end
  · have hb : bound ≠ 9223372036854775808#64 := by word
    simp only [strictUppered, if_neg hb]
    try wp_auto
    have h1 : (W64 1 : w64) = 1#64 := by word
    rw [← h1]
    wp_apply wp_Upper r (bound - W64 1)
    wp_end

theorem wp_StrictLowerSound (r : Range) (bound : w64) (x : Int)
    (hr : contains r x) (hx : sint.Z bound < x) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! StrictLower)) (Val #r)) (Val #bound))
    {{ (out : Range), RET #out; ⌜contains out x⌝ }} := by
  wp_start_folded
  wp_apply wp_StrictLower r bound
  wp_end
  ipureintro
  exact strict_lower_sound r bound x hr hx

theorem wp_StrictUpperSound (r : Range) (bound : w64) (x : Int)
    (hr : contains r x) (hx : x < sint.Z bound) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! StrictUpper)) (Val #r)) (Val #bound))
    {{ (out : Range), RET #out; ⌜contains out x⌝ }} := by
  wp_start_folded
  wp_apply wp_StrictUpper r bound
  wp_end
  ipureintro
  exact strict_upper_sound r bound x hr hx


end proof
end github_com.rstyled.gstyle.domain.runtimeguard.interval
end Perennial
