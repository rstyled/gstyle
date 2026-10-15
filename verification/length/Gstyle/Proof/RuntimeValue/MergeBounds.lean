module
public import Gstyle.Proof.RuntimeValue.Bounds
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

theorem wp_mergeBounds (v other : Value) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval }}
      (App (Val #(methods Value.ty go!"mergeBounds" #v)) (Val #other))
    {{ RET #(mergedBounds v other); True }} := by
  wp_start
  wp_auto
  wp_apply wp_Bounds v
  wp_apply wp_Bounds other
  wp_apply interval.wp_Join (bounds v) (bounds other)
  dsimp [mergedBounds, bounds, interval.joined]
  wp_end


end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
