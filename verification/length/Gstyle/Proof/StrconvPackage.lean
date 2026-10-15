module
public import Perennial.Code.strconv
public import Perennial.Golang.Theory
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI
namespace strconv
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics]
instance isPkgInit_inst : IsPkgInit (IProp GF) pkg_id.strconv := define_is_pkg_init iprop(True)
instance get_isPkgInit_wf_inst : GetIsPkgInitWf (IProp GF) pkg_id.strconv := build_get_is_pkg_init_wf
end strconv
end Perennial
