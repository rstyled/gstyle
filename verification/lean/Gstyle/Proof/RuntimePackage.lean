module
public import Gstyle.Proof.Interval.Base
public import Gstyle.Proof.ValueLayout
public import Gstyle.Proof.StrconvPackage
public import Gstyle.Proof.Worklist.ConstructorPackages
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard
instance isPkgInit_inst : IsPkgInit (IProp GF) pkg := define_is_pkg_init iprop(True)
instance get_isPkgInit_wf_inst : GetIsPkgInitWf (IProp GF) pkg := build_get_is_pkg_init_wf
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
