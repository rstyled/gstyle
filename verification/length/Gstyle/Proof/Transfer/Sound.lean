module
public import Gstyle.Proof.Transfer.Numeric
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [FfiSyntax]

theorem length_bounds (v : Value) (op : GoString) (b : Value) :
    bounds (lengthConstrained v op b) = bounds v := by
  unfold lengthConstrained
  split <;> rfl

theorem numeric_constraint_sound (v : Value) (op : GoString) (b : Value) (x : Int)
    (hx : interval.contains (bounds v) x)
    (hrel : isExact b = true → interval.satisfies op x (sint.Z b.Lower')) :
    interval.contains (bounds (numericConstrained v op b)) x := by
  unfold numericConstrained
  split
  · rename_i he
    simpa [constrainedValue, bounds, length_bounds] using
      interval.constrain_sound (bounds (lengthConstrained v op b)) op b.Lower' x
        (by simpa [length_bounds] using hx) (hrel he)
  · simpa [length_bounds] using hx

theorem represented_not_empty (v : Value) (x : Int)
    (hx : interval.contains (bounds v) x) : isEmpty v = false := by
  cases hl : v.HasLower' <;> cases hu : v.HasUpper'
  all_goals simp [isEmpty, hl, hu]
  simp [interval.contains, bounds, hl, hu] at hx
  omega

theorem represented_comparison_possible (kind op : GoString) (b : Value) (x : Int)
    (hword : -9223372036854775808 ≤ x ∧ x ≤ 9223372036854775807)
    (hrel : isExact b = true → interval.satisfies op x (sint.Z b.Lower')) :
    impossibleComparison kind op b = false := by
  by_cases hk : kind = go!"\"signed\"" <;> cases he : isExact b
  all_goals simp [impossibleComparison, hk, he]
  by_cases hl : op = go!"<" <;> by_cases hg : op = go!">"
  all_goals simp [hl, hg]
  all_goals have hr := hrel he
  all_goals simp [interval.satisfies, hl, hg] at hr
  all_goals intro hb
  all_goals rw [hb] at hr
  all_goals first | change x < (-9223372036854775808 : Int) at hr | change (9223372036854775807 : Int) < x at hr
  all_goals omega

theorem constraint_preserves_other_keys (active : Bool) (kind : GoString) (changing : Bool)
    (key other : w32) (m : StateData) (op : GoString) (b : Value) (hne : key ≠ other) :
    constraintState active kind changing key m op b !! other = m !! other := by
  unfold constraintState
  split
  · simp [hne]
  · rfl

theorem constraint_state_sound (active : Bool) (kind : GoString) (changing : Bool)
    (key : w32) (m : StateData) (op : GoString) (b : Value) (x : Int)
    (hx : interval.contains (bounds (bindingValue kind changing key m)) x)
    (hword : -9223372036854775808 ≤ x ∧ x ≤ 9223372036854775807)
    (hrel : isExact b = true → interval.satisfies op x (sint.Z b.Lower')) :
    (active && constraintRejected kind changing key m op b) = false ∧
    interval.contains (bounds (bindingValue kind changing key
      (constraintState active kind changing key m op b))) x := by
  cases ha : active
  · simpa [constraintState] using hx
  · have hv : interval.contains (bounds (constraintValue kind changing key m op b)) x :=
      numeric_constraint_sound _ op b x hx hrel
    have hr : constraintRejected kind changing key m op b = false := by
      simp [constraintRejected, represented_not_empty _ x hv, represented_comparison_possible kind op b x hword hrel]
    simp [hr, constraintState]
    cases hc : changing
    · simpa [bindingValue, hc] using hv
    · simpa [bindingValue, hc] using hx

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [graphView.TypeAssumptions] [Value.TypeAssumptions] [State.TypeAssumptions] [pdg.query.Assumptions]
variable [FuncUnfold integerConstraint [] integerConstraint.impl]
variable [FuncUnfold lengthConstraint [] lengthConstraint.impl]
variable [FuncUnfold exactInteger [] exactInteger.impl]
variable [FuncUnfold emptyRange [] emptyRange.impl] [interval.Assumptions]

theorem wp_constrain_numeric_sound (receiver : Loc) (view : graphView)
    (kinds : List GoString) (unstable tracked : List Bool)
    (state : Loc) (m : StateData) (key : w32) (kind op : GoString) (changing active : Bool) (b : Value)
    (hk : 0 < uint.Z key ∧ uint.Z key ≤ kinds.length)
    (hkind : kinds[(uint.Z key - 1).toNat]? = some kind)
    (hu : unstable.length = kinds.length) (ht : tracked.length = kinds.length)
    (hunstable : unstable[(uint.Z key - 1).toNat]? = some changing)
    (htracked : tracked[(uint.Z key - 1).toNat]? = some active) (hnil : b.Nil' = false)
    (x : Int) (hx : interval.contains (bounds (bindingValue kind changing key m)) x)
    (hword : -9223372036854775808 ≤ x ∧ x ≤ 9223372036854775807)
    (hrel : isExact b = true → interval.satisfies op x (sint.Z b.Lower')) :
    {{ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard ∗
      isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
      (receiver ↦ view ∗ view.kinds' ↦* kinds ∗ view.unstable' ↦* unstable ∗ view.tracked' ↦* tracked ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"constrain" #receiver)) (Val #key)) (Val #op)) (Val #b)) (Val #state))
    {{ RET #state;
      receiver ↦ view ∗ view.kinds' ↦* kinds ∗ view.unstable' ↦* unstable ∗ view.tracked' ↦* tracked ∗
      state ↦$ (constraintState active kind changing key m op b) ∗
      ⌜interval.contains (bounds (bindingValue kind changing key
        (constraintState active kind changing key m op b))) x⌝ }} := by
  obtain ⟨hr, hs⟩ := constraint_state_sound active kind changing key m op b x hx hword hrel
  wp_start_folded as Hpre
  wp_apply wp_constrain_numeric receiver view kinds unstable tracked state m key kind op changing active b hk hkind hu ht hunstable htracked hnil $$ [Hpre] with ⟨Hview, Hkinds, Hunstable, Htracked, Hstate⟩
  · iexact Hpre
  simp [hr]
  wp_end

end proof

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
