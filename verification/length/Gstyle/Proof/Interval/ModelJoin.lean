module
public import Gstyle.Proof.Interval.Base
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard.interval

section model
variable [FfiSyntax]

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

end model
end github_com.rstyled.gstyle.domain.runtimeguard.interval
end Perennial
