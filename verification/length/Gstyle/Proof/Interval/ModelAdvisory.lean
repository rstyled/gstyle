module
public import Gstyle.Proof.Interval.Base
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard.interval

section model
variable [FfiSyntax]

def advisory (path functionName name : GoString) (line : w32) : Advisory :=
  { Path' := path, Line' := W64 (uint.Z line), EndLine' := W64 (uint.Z line),
    ID' := functionName, Name' := functionName, Kind' := go!"negative_index", ScoreMilli' := W32 800,
    Observation' := (((go!"upper-bound check on " ++ name) ++ go!" has no lower-bound check; ") ++ name) ++ go!" indexes a collection",
    Inference' := go!"verify negative indices are rejected or impossible before the index is used",
    CounterEvidence' := go!"intent unknown; no automatic fix — verify against the contract" }

theorem advisory_line (path functionName name : GoString) (line : w32) :
    sint.Z (advisory path functionName name line).Line' = uint.Z line ∧
      0 ≤ sint.Z (advisory path functionName name line).Line' := by
  simp only [advisory]
  constructor <;> word

end model
end github_com.rstyled.gstyle.domain.runtimeguard.interval
end Perennial
