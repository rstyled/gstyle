module
public import Gstyle.Proof.Interval.Base
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard.interval

section model
variable [FfiSyntax]

def risk (index upperChecked : Bool) (r : Range) : Bool :=
  index && upperChecked && Bool.or (!r.HasLower') (decide (sint.Z r.Lower' < 0))

theorem negative_detected (r : Range) (x : Int) (h : contains r x) (hx : x < 0) :
    risk true true r = true := by
  cases hl : r.HasLower'
  · simp [risk, hl]
  · have hn : sint.Z r.Lower' < 0 := by have := h.1 hl; omega
    simp [risk, hl, hn]


abbrev risky (site : Access) : Bool := risk site.Index' site.UpperChecked' site.Bounds'

end model
end github_com.rstyled.gstyle.domain.runtimeguard.interval
end Perennial
