module
public import Gstyle.GeneratedProof.github_com.rstyled.gstyle.domain.runtimeguard.interval
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard.interval

section model
variable [FfiSyntax]

def contains (r : Range) (x : Int) : Prop :=
  (r.HasLower' = true → sint.Z r.Lower' ≤ x) ∧
  (r.HasUpper' = true → x ≤ sint.Z r.Upper')

def joined (a b : Range) : Range :=
  { HasLower' := a.HasLower' && b.HasLower',
    HasUpper' := a.HasUpper' && b.HasUpper',
    Lower' := if a.HasLower' && b.HasLower' then
      (if sint.Z a.Lower' < sint.Z b.Lower' then a.Lower' else b.Lower') else W64 0,
    Upper' := if a.HasUpper' && b.HasUpper' then
      (if sint.Z a.Upper' > sint.Z b.Upper' then a.Upper' else b.Upper') else W64 0 }

theorem join_sound (a b : Range) (x : Int) :
    contains a x ∨ contains b x → contains (joined a b) x := by
  intro h
  rcases h with h | h
  all_goals
    cases ha : a.HasLower' <;> cases hb : b.HasLower' <;>
      cases hc : a.HasUpper' <;> cases hd : b.HasUpper' <;>
      by_cases hl : sint.Z a.Lower' < sint.Z b.Lower' <;>
      by_cases hu : sint.Z a.Upper' > sint.Z b.Upper'
  all_goals simp [contains, joined, ha, hb, hc, hd, hl, hu] at h ⊢ <;> omega

def risk (index upperChecked : Bool) (r : Range) : Bool :=
  index && upperChecked && Bool.or (!r.HasLower') (decide (sint.Z r.Lower' < 0))

theorem negative_detected (r : Range) (x : Int) (h : contains r x) (hx : x < 0) :
    risk true true r = true := by
  cases hl : r.HasLower'
  · simp [risk, hl]
  · have hn : sint.Z r.Lower' < 0 := by have := h.1 hl; omega
    simp [risk, hl, hn]

end model

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics] [package_sem : Assumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval
instance isPkgInit_inst : IsPkgInit (IProp GF) pkg := define_is_pkg_init iprop(True)
instance get_isPkgInit_wf_inst : GetIsPkgInitWf (IProp GF) pkg := build_get_is_pkg_init_wf

set_option maxHeartbeats 2000000 in
theorem wp_Join (a b : Range) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! Join)) (Val #a)) (Val #b))
    {{ RET #(joined a b); True }} := by
  haveI : FuncUnfold go.min [go.int64, go.int64] (_root_.Perennial.min.impl go.int64 2) :=
    go.PredeclaredSemantics.min_unfold 2 go.int64
  haveI : FuncUnfold go.max [go.int64, go.int64] (_root_.Perennial.max.impl go.int64 2) :=
    go.PredeclaredSemantics.max_unfold 2 go.int64
  wp_start
  wp_auto
  wp_func_call
  wp_call
  try wp_auto
  by_cases hl : sint.Z a.Lower' < sint.Z b.Lower' <;>
    by_cases hu : sint.Z a.Upper' > sint.Z b.Upper' <;>
    cases ha : a.HasLower' <;> cases hb : b.HasLower' <;>
    cases hc : a.HasUpper' <;> cases hd : b.HasUpper'
  all_goals iterate 10 (try simp [joined, hl, hu, ha, hb, hc, hd]; try wp_auto; try wp_func_call; try wp_call; try wp_auto)
  all_goals wp_end

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


theorem wp_JoinSound (a b : Range) (x : Int)
    (h : contains a x ∨ contains b x) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! Join)) (Val #a)) (Val #b))
    {{ (r : Range), RET #r; ⌜contains r x⌝ }} := by
  wp_start_folded
  wp_apply wp_Join a b
  wp_end
  ipureintro
  exact join_sound a b x h

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
