module
public import Gstyle.Proof.Worklist.ConstructorWritesSyntax
public import Gstyle.Proof.Pdg.AccessLayout
public import Gstyle.Proof.RuntimePackage
public import Gstyle.Proof.Worklist.ConstructorTrackingWrites
@[expose] public section
noncomputable section
set_option maxHeartbeats 0
set_option maxRecDepth 100000
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std BigSepL
namespace github_com.rstyled.gstyle.domain.runtimeguard
section execution
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [pdg.query.Assumptions] [pdg.Access.TypeAssumptions] [pdg.Position.TypeAssumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

/-- Execute all assignment-write appends. Zero writes skip; repeated symbols append again. -/
theorem wp_tracking_writes_raw (definitionsLocal refLocal : Loc) (definitions writes : GoSlice)
    (rows : List GoSlice) (values : List pdg.Access) (ref : w32)
    (hvalid : ∀ write ∈ values, write.Symbol' = W32 0 ∨
      (0 < uint.Z write.Symbol' ∧ uint.Z write.Symbol' ≤ rows.length)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ definitionsLocal ↦ definitions ∗ refLocal ↦ ref ∗
      definitions ↦* rows ∗ trackingDefinitionRowsOwn rows ∗ writes ↦*□ values }}
      (trackingWritesRawRun definitionsLocal refLocal writes)
    {{ RET executeVal; definitionsLocal ↦ definitions ∗ refLocal ↦ ref ∗
      ∃ (out : List GoSlice), definitions ↦* out ∗ trackingDefinitionRowsOwn out ∗ ⌜out.length = rows.length⌝ }} := by
  wp_start_folded as ⟨HdefinitionsLocal, Href, Hdefinitions, Hrows, #Hwrites⟩
  unfold trackingWritesRawRun
  rw [tracking_writes_shape]
  wp_pures
  ieval (change _ ⊢ WP (trackingWritesNormalizedRun definitionsLocal refLocal writes) {{ Φ }})
  wp_apply +noauto wp_tracking_writes_normalized definitionsLocal refLocal definitions writes rows values ref hvalid
    $$ [HdefinitionsLocal Href Hdefinitions Hrows Hwrites]
    with ⟨HdefinitionsLocal, Href, %out, Hdefinitions, Hrows, %Hlen⟩
  · iframe HdefinitionsLocal Href Hdefinitions Hrows Hwrites
  wp_end

end execution
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
