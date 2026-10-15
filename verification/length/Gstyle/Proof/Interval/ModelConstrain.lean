module
public import Gstyle.Proof.Interval.ModelStrict
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard.interval

section model
variable [FfiSyntax]

def constrained (r : Range) (operator : GoString) (bound : w64) : Range :=
  if operator = go!">=" then lowered r bound else
  if operator = go!">" then strictLowered r bound else
  if operator = go!"<" then strictUppered r bound else
  if operator = go!"<=" then uppered r bound else
  if operator = go!"==" then uppered (lowered r bound) bound else r

def satisfies (operator : GoString) (x bound : Int) : Prop :=
  if operator = go!">=" then bound ≤ x else
  if operator = go!">" then bound < x else
  if operator = go!"<" then x < bound else
  if operator = go!"<=" then x ≤ bound else
  if operator = go!"==" then x = bound else True

theorem constrain_sound (r : Range) (operator : GoString) (bound : w64) (x : Int)
    (hr : contains r x) (hx : satisfies operator x (sint.Z bound)) :
    contains (constrained r operator bound) x := by
  by_cases h0 : operator = go!">=" <;> by_cases h1 : operator = go!">" <;>
    by_cases h2 : operator = go!"<" <;> by_cases h3 : operator = go!"<=" <;>
    by_cases h4 : operator = go!"=="
  all_goals simp_all [constrained, satisfies, lower_intersection, upper_intersection]
  all_goals first | exact strict_lower_sound r bound x hr hx | exact strict_upper_sound r bound x hr hx


end model
end github_com.rstyled.gstyle.domain.runtimeguard.interval
end Perennial
