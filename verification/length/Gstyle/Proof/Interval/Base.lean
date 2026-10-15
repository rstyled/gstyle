module
public import Gstyle.GeneratedProof.github_com.rstyled.gstyle.domain.runtimeguard.interval
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard.interval

section model
variable [FfiSyntax]

def contains (r : Range) (x : Int) : Prop :=
  (r.HasLower' = true → sint.Z r.Lower' ≤ x) ∧
  (r.HasUpper' = true → x ≤ sint.Z r.Upper')

end model

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics] [package_sem : Assumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval
instance isPkgInit_inst : IsPkgInit (IProp GF) pkg := define_is_pkg_init iprop(True)
instance get_isPkgInit_wf_inst : GetIsPkgInitWf (IProp GF) pkg := build_get_is_pkg_init_wf


end proof
end github_com.rstyled.gstyle.domain.runtimeguard.interval
end Perennial
