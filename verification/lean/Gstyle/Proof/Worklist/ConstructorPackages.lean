module
public import Gstyle.Code.encoding.json
public import Perennial.Proof.fmt.Base
public import Perennial.Proof.strings.Base
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI
namespace encoding.json
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics]
/-- JSON stays opaque. Scalar construction never invokes Unmarshal. -/
instance isPkgInit_inst : IsPkgInit (IProp GF) pkg_id.encoding.json :=
  define_is_pkg_init iprop(True)
instance get_isPkgInit_wf_inst : GetIsPkgInitWf (IProp GF) pkg_id.encoding.json :=
  build_get_is_pkg_init_wf
end encoding.json
end Perennial
