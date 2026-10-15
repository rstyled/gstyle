module
public import Gstyle.Code.github_com.rstyled.gstyle.domain.pdg
public import Gstyle.Proof.Queries
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI
namespace github_com.rstyled.gstyle.domain.pdg
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics]
instance isPkgInit_inst : IsPkgInit (IProp GF) pkg_id.github_com.rstyled.gstyle.domain.pdg := define_is_pkg_init iprop(True)
instance get_isPkgInit_wf_inst : GetIsPkgInitWf (IProp GF) pkg_id.github_com.rstyled.gstyle.domain.pdg := build_get_is_pkg_init_wf
end github_com.rstyled.gstyle.domain.pdg
end Perennial
