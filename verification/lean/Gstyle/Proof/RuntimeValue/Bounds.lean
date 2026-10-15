module
public import Gstyle.Proof.RuntimeValueModel
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

variable [FfiSyntax]

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics] [value_sem : Value.TypeAssumptions]
variable [constraint_unfold : FuncUnfold integerConstraint [] integerConstraint.impl]
variable [interval_sem : interval.Assumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

theorem wp_Bounds (v : Value) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (Val #(methods Value.ty go!"Bounds" #v)) (Val #()))
    {{ RET #(bounds v); True }} := by
  unfold bounds
  wp_start
  wp_auto
  wp_end


end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
