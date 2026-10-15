module
public import Gstyle.Proof.Pdg.Text
public import Gstyle.Proof.Pdg.FactLayout
public import Gstyle.Proof.RuntimeValue
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [pdg.Fact.TypeAssumptions] [pdg.Tables.TypeAssumptions] [pdg.query.Assumptions]
variable [FuncUnfold operatorText [] operatorText.impl]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

theorem wp_operatorText_unavailable (receiver : Loc) (fact : pdg.Fact)
    (hstatus : fact.Status' ≠ 1#8) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! operatorText)) (Val #receiver)) (Val #fact))
    {{ RET #(go!""); True }} := by
  wp_start
  try wp_auto
  wp_if_destruct
  · exfalso; simp_all
  · wp_end

theorem wp_operatorText_known (receiver : Loc) (fact : pdg.Fact)
    (table : pdg.Tables) (strings : List GoString) (dq : DFrac)
    (hstatus : fact.Status' = 1#8) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{dq} table ∗ table.Strings' ↦*{dq} strings) }}
      (App (App (Val (@! operatorText)) (Val #receiver)) (Val #fact))
    {{ RET #(pdg.tableText strings fact.Value'); receiver ↦{dq} table ∗ table.Strings' ↦*{dq} strings }} := by
  wp_start as ⟨Htable, Hstrings⟩
  try wp_auto
  wp_if_destruct
  · wp_apply pdg.wp_Text receiver table strings dq fact.Value' $$ [Htable Hstrings] with ⟨Htable, Hstrings⟩
    · iframe
    wp_end
  · exfalso; simp_all

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
