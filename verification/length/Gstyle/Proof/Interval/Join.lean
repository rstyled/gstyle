module
public import Gstyle.Proof.Interval.ModelJoin
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
theorem wp_minInt64 (x y : w64) :
    {{ (True : IProp GF) }}
      (App (App (Val #(functions go.min [go.int64, go.int64])) (Val #x)) (Val #y))
    {{ RET #(if sint.Z x < sint.Z y then x else y); True }} := by
  haveI : FuncUnfold go.min [go.int64, go.int64] (_root_.Perennial.min.impl go.int64 2) :=
    go.PredeclaredSemantics.min_unfold 2 go.int64
  wp_start_folded as _
  wp_func_call
  wp_call
  by_cases h : sint.Z x < sint.Z y
  all_goals iterate 3 (try simp [h]; try wp_auto)
  all_goals wp_end

theorem wp_maxInt64 (x y : w64) :
    {{ (True : IProp GF) }}
      (App (App (Val #(functions go.max [go.int64, go.int64])) (Val #x)) (Val #y))
    {{ RET #(if sint.Z x > sint.Z y then x else y); True }} := by
  haveI : FuncUnfold go.max [go.int64, go.int64] (_root_.Perennial.max.impl go.int64 2) :=
    go.PredeclaredSemantics.max_unfold 2 go.int64
  wp_start_folded as _
  wp_func_call
  wp_call
  by_cases h : sint.Z x > sint.Z y
  all_goals iterate 3 (try simp [h]; try wp_auto)
  all_goals wp_end

theorem wp_Join (a b : Range) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! Join)) (Val #a)) (Val #b))
    {{ RET #(joined a b); True }} := by
  wp_start
  wp_auto
  wp_apply wp_minInt64 a.Lower' b.Lower'
  wp_apply wp_maxInt64 a.Upper' b.Upper'
  cases ha : a.HasLower' <;> cases hb : b.HasLower' <;>
    cases hc : a.HasUpper' <;> cases hd : b.HasUpper'
  all_goals iterate 5 (try simp [joined, ha, hb, hc, hd]; try wp_auto)
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

end proof
end github_com.rstyled.gstyle.domain.runtimeguard.interval
end Perennial
