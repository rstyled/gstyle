module
public import Gstyle.Proof.Interval.ModelBounds
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard.interval

section model
variable [FfiSyntax]

def strictLowered (r : Range) (bound : w64) : Range :=
  if bound = 9223372036854775807#64 then r else lowered r (bound + 1#64)

def strictUppered (r : Range) (bound : w64) : Range :=
  if bound = 9223372036854775808#64 then r else uppered r (bound - 1#64)

theorem strict_lower_sound (r : Range) (bound : w64) (x : Int)
    (hr : contains r x) (hx : sint.Z bound < x) :
    contains (strictLowered r bound) x := by
  by_cases hb : bound = 9223372036854775807#64
  · simpa [strictLowered, hb] using hr
  · simp only [strictLowered, if_neg hb]
    apply (lower_intersection r (bound + 1#64) x).mpr
    have hstep : sint.Z (bound + 1#64) = sint.Z bound + 1 := by word
    exact ⟨hr, by omega⟩

theorem strict_upper_sound (r : Range) (bound : w64) (x : Int)
    (hr : contains r x) (hx : x < sint.Z bound) :
    contains (strictUppered r bound) x := by
  by_cases hb : bound = 9223372036854775808#64
  · simpa [strictUppered, hb] using hr
  · simp only [strictUppered, if_neg hb]
    apply (upper_intersection r (bound - 1#64) x).mpr
    have hstep : sint.Z (bound - 1#64) = sint.Z bound - 1 := by word
    exact ⟨hr, by omega⟩


end model
end github_com.rstyled.gstyle.domain.runtimeguard.interval
end Perennial
