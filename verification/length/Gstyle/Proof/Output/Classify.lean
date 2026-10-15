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
theorem wp_Formatted (pkgName name : GoString) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! Formatted)) (Val #pkgName)) (Val #name))
    {{ RET #(decide (pkgName = go!"fmt" ∧
      (name = go!"Fprint" ∨ name = go!"Fprintf" ∨ name = go!"Fprintln"))); True }} := by
  wp_start
  wp_auto
  by_cases hp : pkgName = go!"fmt" <;>
    by_cases h0 : name = go!"Fprint" <;>
    by_cases h1 : name = go!"Fprintf" <;>
    by_cases h2 : name = go!"Fprintln"
  all_goals iterate 5 (try simp [hp, h0, h1, h2]; try wp_auto)
  all_goals wp_end

theorem wp_Memory (pkgName name : GoString) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! Memory)) (Val #pkgName)) (Val #name))
    {{ RET #(decide ((pkgName = go!"bytes" ∧ name = go!"Buffer") ∨
      (pkgName = go!"strings" ∧ name = go!"Builder"))); True }} := by
  wp_start
  wp_auto
  by_cases hp : pkgName = go!"bytes" <;>
    by_cases hq : pkgName = go!"strings" <;>
    by_cases h0 : name = go!"Buffer" <;>
    by_cases h1 : name = go!"Builder"
  all_goals iterate 5 (try simp [hp, hq, h0, h1]; try wp_auto)
  all_goals wp_end


def discarded (input : Decision) : Bool :=
  input.Formatted' && !input.Memory' &&
    (input.Standalone' || (decide (input.Values' = (1#64)) &&
      decide (input.Results' = (2#64)) && input.ErrorBlank'))

theorem wp_Discards (input : Decision) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (Val (@! Discards)) (Val #input))
    {{ RET #(discarded input); True }} := by
  wp_start
  wp_auto
  cases hf : input.Formatted' <;> cases hm : input.Memory' <;>
    cases hs : input.Standalone' <;> cases hb : input.ErrorBlank' <;>
    by_cases hv : input.Values' = (1#64) <;> by_cases hr : input.Results' = (2#64)
  all_goals iterate 5 (try simp [discarded, hf, hm, hs, hb, hv, hr]; try wp_auto)
  all_goals try simp only [hb]
  all_goals wp_end


def classified (callee writer : Identity) (use : Consumption) : Bool :=
  decide (callee.Package' = go!"fmt" ∧
    (callee.Name' = go!"Fprint" ∨ callee.Name' = go!"Fprintf" ∨ callee.Name' = go!"Fprintln")) &&
  !decide ((writer.Package' = go!"bytes" ∧ writer.Name' = go!"Buffer") ∨
    (writer.Package' = go!"strings" ∧ writer.Name' = go!"Builder")) &&
  (use.Standalone' || (decide (use.Values' = (1#64)) && decide (use.Results' = (2#64)) &&
    decide (use.ErrorName' = go!"_")))

theorem wp_Classify (callee writer : Identity) (use : Consumption) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (App (Val (@! Classify)) (Val #callee)) (Val #writer)) (Val #use))
    {{ RET #(classified callee writer use); True }} := by
  wp_start
  wp_auto
  wp_apply wp_Formatted
  wp_apply wp_Memory
  wp_apply wp_Discards
  unfold classified discarded
  wp_end

end proof
end github_com.rstyled.gstyle.domain.patterns.outputcheck
end Perennial
