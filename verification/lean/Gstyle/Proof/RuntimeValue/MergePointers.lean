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

theorem wp_mergePointers (v other out : Value) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val #(methods Value.ty go!"mergePointers" #v)) (Val #other)) (Val #out))
    {{ RET #(mergedPointers v other out); True }} := by
  wp_start
  wp_auto
  cases hn : v.NonNil' <;> cases hz : v.Nil' <;> cases hg : v.Guarded' <;>
    by_cases hi : v.Identity' = other.Identity'
  all_goals iterate 10 (try simp [mergedPointers, hn, hz, hg, hi]; try wp_auto)
  all_goals wp_end


end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
