module
public import Gstyle.Proof.Output.Base
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.patterns.outputcheck

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics] [package_sem : Assumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.patterns.outputcheck
def advisory (function : Function) (site : Site) : Advisory :=
  { Path' := function.Path', Line' := site.Line', EndLine' := site.Line',
    ID' := function.ID', Name' := function.Name',
    Kind' := go!"unchecked_output", ScoreMilli' := W32 800,
    Observation' := go!"formatted output error is discarded: " ++ site.Callee',
    Inference' := go!"a failed write can be reported as success; propagate or explicitly handle the output error",
    CounterEvidence' := go!"best-effort output may deliberately ignore failures; no automatic fix" }

theorem wp_describe (function : Function) (site : Site) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! describe)) (Val #function)) (Val #site))
    {{ RET #(advisory function site); True }} := by
  unfold advisory
  wp_start
  wp_auto
  wp_end

end proof
end github_com.rstyled.gstyle.domain.patterns.outputcheck
end Perennial
