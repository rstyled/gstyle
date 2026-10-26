module
public import Gstyle.Proof.Worklist.ConstructorTracking
public import Gstyle.Proof.Worklist.ConstructorTrackingSyntax
@[expose] public section
noncomputable section
set_option maxRecDepth 100000
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
section execution
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [graphView.TypeAssumptions] [pdg.query.Assumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

/-- An empty scalar queue executes the actual cursor guard and never reads the
definition array, graph receiver, or visited flags. -/
theorem wp_trackingCursor_empty (queueLocal : Loc) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ queueLocal ↦ (slice.nil : GoSlice) }}
      (trackingCursorRun queueLocal)
    {{ RET #(); queueLocal ↦ (slice.nil : GoSlice) }} := by
  wp_start_folded as Hqueue
  unfold trackingCursorRun trackingCursorBody trackingBody graphView.trackBindings.impl
  dsimp only [trackingCursorSite, Option.or, Option.getD]
  wp_auto
  wp_for
  wp_end

end execution
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
