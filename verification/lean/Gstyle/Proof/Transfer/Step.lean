module
public import Gstyle.Proof.RuntimeValueModel
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [FfiSyntax]

def incrementedInteger (v : Value) : Value :=
  if Bool.and (Bool.not v.BelowLength') (Bool.or (Bool.not v.HasUpper') (decide (v.Upper' = 9223372036854775807#64))) then zero_val Value
  else { (zero_val Value) with HasLower' := v.HasLower', Lower' := v.Lower' }

def decrementedInteger (v : Value) : Value :=
  if Bool.or (Bool.not v.HasLower') (decide (v.Lower' = 9223372036854775808#64)) then zero_val Value
  else { (zero_val Value) with HasUpper' := v.HasUpper', Upper' := v.Upper' }

def steppedInteger (v : Value) (increment : Bool) : Value :=
  if increment then incrementedInteger v else decrementedInteger v

def stepWord (x : w64) (increment : Bool) : w64 :=
  if increment then x + 1#64 else x - 1#64

theorem increment_integer_sound (v : Value) (x : w64)
    (hx : interval.contains (bounds v) (sint.Z x))
    (hlen : v.BelowLength' = true → sint.Z x < 9223372036854775807) :
    interval.contains (bounds (incrementedInteger v)) (sint.Z (x + 1#64)) := by
  unfold incrementedInteger
  split
  · simp [bounds, interval.contains, zero_val]
  · rename_i hs
    have hmax : sint.Z x < 9223372036854775807 := by
      cases hb : v.BelowLength'
      · cases hu : v.HasUpper'
        · simp [hb, hu] at hs
        · have hn : v.Upper' ≠ 9223372036854775807#64 := by simpa [hb, hu] using hs
          have hxup : sint.Z x ≤ sint.Z v.Upper' := by
            simpa [interval.contains, bounds, hu] using hx.2
          word
      · exact hlen hb
    have hstep : sint.Z (x + 1#64) = sint.Z x + 1 := by word
    cases hl : v.HasLower'
    · simp [bounds, interval.contains, hl, zero_val]
    · have hxlo : sint.Z v.Lower' ≤ sint.Z x := by simpa [interval.contains, bounds, hl] using hx.1
      simp [bounds, interval.contains, hl, zero_val, hstep]
      omega

theorem decrement_integer_sound (v : Value) (x : w64)
    (hx : interval.contains (bounds v) (sint.Z x)) :
    interval.contains (bounds (decrementedInteger v)) (sint.Z (x - 1#64)) := by
  unfold decrementedInteger
  split
  · simp [bounds, interval.contains, zero_val]
  · rename_i hs
    have hl : v.HasLower' = true := by cases h : v.HasLower' <;> simp_all
    have hn : v.Lower' ≠ 9223372036854775808#64 := by simpa [hl] using hs
    have hxlo : sint.Z v.Lower' ≤ sint.Z x := by simpa [interval.contains, bounds, hl] using hx.1
    have hstep : sint.Z (x - 1#64) = sint.Z x - 1 := by word
    cases hu : v.HasUpper'
    · simp [bounds, interval.contains, hu, zero_val]
    · have hxup : sint.Z x ≤ sint.Z v.Upper' := by simpa [interval.contains, bounds, hu] using hx.2
      simp [bounds, interval.contains, hu, zero_val, hstep]
      omega

theorem integer_step_sound (v : Value) (increment : Bool) (x : w64)
    (hx : interval.contains (bounds v) (sint.Z x))
    (hlen : increment = true → v.BelowLength' = true → sint.Z x < 9223372036854775807) :
    interval.contains (bounds (steppedInteger v increment)) (sint.Z (stepWord x increment)) := by
  cases hi : increment
  · exact decrement_integer_sound v x hx
  · exact increment_integer_sound v x hx (hlen hi)

theorem integer_step_metadata (v : Value) (increment : Bool) :
    let out := steppedInteger v increment
    out.Identity' = 0 ∧ out.NonNil' = false ∧ out.Nil' = false ∧ out.Guarded' = false ∧
    out.UpperChecked' = false ∧ out.Integer' = false ∧ out.Length' = false ∧
    out.BelowLength' = false ∧ out.Extent' = 0 ∧ out.HasExtent' = false := by
  cases increment <;> dsimp [steppedInteger]
  all_goals first | unfold incrementedInteger | unfold decrementedInteger
  all_goals split <;> exact ⟨rfl, rfl, rfl, rfl, rfl, rfl, rfl, rfl, rfl, rfl⟩

theorem integer_step_interval_sound (v : Value) (increment : Bool) (x : w64)
    (hx : interval.contains (bounds v) (sint.Z x)) (hflag : v.BelowLength' = false) :
    interval.contains (bounds (steppedInteger v increment)) (sint.Z (stepWord x increment)) :=
  integer_step_sound v increment x hx (by simp [hflag])

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [Value.TypeAssumptions]
variable [FuncUnfold integerConstraint [] integerConstraint.impl]
variable [FuncUnfold integerStep [] integerStep.impl]
variable [FuncUnfold incrementInteger [] incrementInteger.impl]
variable [FuncUnfold decrementInteger [] decrementInteger.impl]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

theorem wp_incrementInteger (v : Value) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (Val (@! incrementInteger)) (Val #v))
    {{ RET #(incrementedInteger v); True }} := by
  wp_start
  cases hb : v.BelowLength' <;> cases hu : v.HasUpper' <;> by_cases he : v.Upper' = 9223372036854775807#64
  all_goals iterate 8 (try simp [incrementedInteger, hb, hu, he]; try wp_auto)
  all_goals wp_end

theorem wp_decrementInteger (v : Value) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (Val (@! decrementInteger)) (Val #v))
    {{ RET #(decrementedInteger v); True }} := by
  wp_start
  cases hl : v.HasLower' <;> by_cases he : v.Lower' = 9223372036854775808#64
  all_goals iterate 8 (try simp [decrementedInteger, hl, he]; try wp_auto)
  all_goals wp_end

theorem wp_integerStep (v : Value) (increment : Bool) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! integerStep)) (Val #v)) (Val #increment))
    {{ RET #(steppedInteger v increment); True }} := by
  wp_start
  cases hi : increment
  · wp_auto
    wp_apply wp_decrementInteger v
    simp [steppedInteger]
    wp_end
  · wp_auto
    wp_apply wp_incrementInteger v
    simp [steppedInteger]
    wp_end

theorem wp_integerStep_sound (v : Value) (increment : Bool) (x : w64)
    (hx : interval.contains (bounds v) (sint.Z x))
    (hlen : increment = true → v.BelowLength' = true → sint.Z x < 9223372036854775807) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! integerStep)) (Val #v)) (Val #increment))
    {{ (out : Value), RET #out;
      ⌜interval.contains (bounds out) (sint.Z (stepWord x increment))⌝ }} := by
  wp_start_folded
  wp_apply wp_integerStep v increment
  wp_end
  ipureintro
  exact integer_step_sound v increment x hx hlen

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
