module
public import Gstyle.Proof.Interval.Base
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard.interval

section model
variable [FfiSyntax]

def lowered (r : Range) (bound : w64) : Range :=
  { r with Lower' := if Bool.or (!r.HasLower') (decide (sint.Z r.Lower' < sint.Z bound)) then bound else r.Lower', HasLower' := true }

def uppered (r : Range) (bound : w64) : Range :=
  { r with Upper' := if Bool.or (!r.HasUpper') (decide (sint.Z r.Upper' > sint.Z bound)) then bound else r.Upper', HasUpper' := true }

theorem lower_intersection (r : Range) (bound : w64) (x : Int) :
    contains (lowered r bound) x ↔ contains r x ∧ sint.Z bound ≤ x := by
  cases hl : r.HasLower' <;> cases hu : r.HasUpper' <;> by_cases hc : sint.Z r.Lower' < sint.Z bound
  all_goals simp [contains, lowered, hl, hu, hc] <;> omega

theorem upper_intersection (r : Range) (bound : w64) (x : Int) :
    contains (uppered r bound) x ↔ contains r x ∧ x ≤ sint.Z bound := by
  cases hu : r.HasUpper' <;> cases hl : r.HasLower' <;> by_cases hc : sint.Z r.Upper' > sint.Z bound
  all_goals simp [contains, uppered, hu, hl, hc] <;> omega


end model
end github_com.rstyled.gstyle.domain.runtimeguard.interval
end Perennial
