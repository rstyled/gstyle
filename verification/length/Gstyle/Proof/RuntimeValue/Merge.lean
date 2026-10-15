module
public import Gstyle.Proof.RuntimeValue.MergeBounds
public import Gstyle.Proof.RuntimeValue.MergePointers
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

variable [FfiSyntax]

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics] [value_sem : Value.TypeAssumptions]
variable [constraint_unfold : FuncUnfold integerConstraint [] integerConstraint.impl]
variable [interval_sem : interval.Assumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

theorem wp_merge (v other : Value) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval }}
      (App (Val #(methods Value.ty go!"merge" #v)) (Val #other))
    {{ RET #(merged v other); True }} := by
  wp_start
  wp_auto
  wp_apply wp_mergeBounds v other $$ []
  · iframe; iassumption
  dsimp [mergedBounds, bounds, interval.joined]
  cases hu : v.UpperChecked' <;> cases hl : v.Length' <;> cases hb : v.BelowLength'
  all_goals iterate 10 (try simp [hu, hl, hb]; try wp_pures; try wp_auto)
  all_goals wp_apply wp_mergePointers v other _
  all_goals dsimp [merged, mergedBounds, bounds, interval.joined]
  all_goals simp_all
  all_goals wp_end

theorem wp_mergeSound (v other : Value) (x : Int)
    (hr : interval.contains (bounds v) x ∨ interval.contains (bounds other) x) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval }}
      (App (Val #(methods Value.ty go!"merge" #v)) (Val #other))
    {{ (out : Value), RET #out; ⌜interval.contains (bounds out) x⌝ }} := by
  wp_start_folded as Hinit
  wp_apply wp_merge v other
  · iframe; iassumption
  wp_end
  ipureintro
  simpa [bounds, merged, mergedPointers, mergedBounds] using interval.join_sound (bounds v) (bounds other) x hr


end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
