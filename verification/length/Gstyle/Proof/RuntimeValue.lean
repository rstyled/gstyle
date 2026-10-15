module
public import Gstyle.Proof.ValueLayout
public import Gstyle.Proof.Interval
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

variable [FfiSyntax]

def bounds (v : Value) : interval.Range :=
  { Lower' := v.Lower', Upper' := v.Upper', HasLower' := v.HasLower', HasUpper' := v.HasUpper' }

def constrainedValue (v : Value) (operator : GoString) (bound : w64) : Value :=
  let r := interval.constrained (bounds v) operator bound
  { v with Lower' := r.Lower', Upper' := r.Upper', HasLower' := r.HasLower', HasUpper' := r.HasUpper' }

def mergedBounds (v other : Value) : Value :=
  let r := interval.joined (bounds v) (bounds other)
  { Identity' := 0, Lower' := r.Lower', Upper' := r.Upper', HasLower' := r.HasLower', HasUpper' := r.HasUpper',
    NonNil' := false, Nil' := false, Guarded' := false, UpperChecked' := false, Integer' := false,
    Length' := false, BelowLength' := false, Extent' := 0, HasExtent' := false }

def mergedPointers (v other out : Value) : Value :=
  { out with
    NonNil' := Bool.and v.NonNil' other.NonNil', Nil' := Bool.and v.Nil' other.Nil',
    Guarded' := Bool.and v.Guarded' other.Guarded',
    Identity' := if v.Identity' = other.Identity' then v.Identity' else out.Identity' }

def merged (v other : Value) : Value :=
  mergedPointers v other { (mergedBounds v other) with
    UpperChecked' := Bool.or v.UpperChecked' other.UpperChecked',
    Length' := v.Length' && other.Length', BelowLength' := v.BelowLength' && other.BelowLength' }


section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics] [value_sem : Value.TypeAssumptions]
variable [constraint_unfold : FuncUnfold integerConstraint [] integerConstraint.impl]
variable [interval_sem : interval.Assumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard
instance isPkgInit_inst : IsPkgInit (IProp GF) pkg := define_is_pkg_init iprop(True)
instance get_isPkgInit_wf_inst : GetIsPkgInitWf (IProp GF) pkg := build_get_is_pkg_init_wf

theorem wp_Bounds (v : Value) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (Val #(methods Value.ty go!"Bounds" #v)) (Val #()))
    {{ RET #(bounds v); True }} := by
  unfold bounds
  wp_start
  wp_auto
  wp_end

theorem wp_integerConstraint (v : Value) (operator : GoString) (bound : w64) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval }}
      (App (App (App (Val (@! integerConstraint)) (Val #v)) (Val #operator)) (Val #bound))
    {{ RET #(constrainedValue v operator bound); True }} := by
  wp_start
  wp_auto
  wp_apply wp_Bounds v
  wp_apply interval.wp_Constrain (bounds v) operator bound
  unfold constrainedValue
  wp_end

theorem wp_integerConstraintSound (v : Value) (operator : GoString) (bound : w64) (x : Int)
    (hr : interval.contains (bounds v) x) (hx : interval.satisfies operator x (sint.Z bound)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval }}
      (App (App (App (Val (@! integerConstraint)) (Val #v)) (Val #operator)) (Val #bound))
    {{ (out : Value), RET #out; ⌜interval.contains (bounds out) x⌝ }} := by
  wp_start_folded as Hinit
  wp_apply wp_integerConstraint v operator bound
  · iframe; iassumption
  wp_end
  ipureintro
  simpa [bounds, constrainedValue] using interval.constrain_sound (bounds v) operator bound x hr hx

theorem wp_mergeBounds (v other : Value) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval }}
      (App (Val #(methods Value.ty go!"mergeBounds" #v)) (Val #other))
    {{ RET #(mergedBounds v other); True }} := by
  wp_start
  wp_auto
  wp_apply wp_Bounds v
  wp_apply wp_Bounds other
  wp_apply interval.wp_Join (bounds v) (bounds other)
  dsimp [mergedBounds, bounds, interval.joined]
  wp_end

theorem wp_mergePointers (v other out : Value) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val #(methods Value.ty go!"mergePointers" #v)) (Val #other)) (Val #out))
    {{ RET #(mergedPointers v other out); True }} := by
  wp_start
  wp_auto
  cases hn : v.NonNil' <;> cases hz : v.Nil' <;> cases hg : v.Guarded' <;>
    by_cases hi : v.Identity' = other.Identity'
  all_goals iterate 10 (try simp [mergedPointers, hn, hz, hg, hi]; try wp_auto)
  all_goals wp_end

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
