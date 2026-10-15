module
public import Gstyle.Proof.Interval.ModelAdvisory
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard.interval

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics] [package_sem : Assumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval
theorem wp_Describe (path functionName name : GoString) (line : w32) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (App (App (Val (@! Describe)) (Val #path)) (Val #functionName)) (Val #name)) (Val #line))
    {{ RET #(advisory path functionName name line); True }} := by
  unfold advisory
  wp_start
  wp_auto
  wp_end

end proof
end github_com.rstyled.gstyle.domain.runtimeguard.interval
end Perennial
