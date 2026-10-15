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

theorem wp_integerConstraint (v : Value) (operator : GoString) (bound : w64) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval }}
      (App (App (App (Val (@! integerConstraint)) (Val #v)) (Val #operator)) (Val #bound))
    {{ RET #(constrainedValue v operator bound); True }} := by
  wp_start
  wp_auto
  wp_apply wp_Bounds v
  wp_apply interval.wp_Constrain (bounds v) operator bound
  unfold constrainedValue
  wp_end

theorem wp_integerConstraintSound (v : Value) (operator : GoString) (bound : w64) (x : Int)
    (hr : interval.contains (bounds v) x) (hx : interval.satisfies operator x (sint.Z bound)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval }}
      (App (App (App (Val (@! integerConstraint)) (Val #v)) (Val #operator)) (Val #bound))
    {{ (out : Value), RET #out; ⌜interval.contains (bounds out) x⌝ }} := by
  wp_start_folded as Hinit
  wp_apply wp_integerConstraint v operator bound
  · iframe; iassumption
  wp_end
  ipureintro
  simpa [bounds, constrainedValue] using interval.constrain_sound (bounds v) operator bound x hr hx


end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
