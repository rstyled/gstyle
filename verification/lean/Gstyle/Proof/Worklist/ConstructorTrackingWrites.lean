module
public import Gstyle.Proof.Worklist.ConstructorWritesSyntax
public import Gstyle.Proof.Pdg.AccessLayout
public import Gstyle.Proof.RuntimePackage
public import Gstyle.Proof.Worklist.ConstructorTrackingDefinitions
@[expose] public section
noncomputable section
set_option maxHeartbeats 0
set_option maxRecDepth 100000
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std BigSepL
namespace github_com.rstyled.gstyle.domain.runtimeguard
section selectedSyntax
variable [FfiSyntax] [GoGlobalContext]
/-- The actual write-loop residual after the pure Access zero-value step. -/
def trackingWritesNormalizedRun (definitionsLocal refLocal : Loc) (writes : GoSlice) : Expr :=
  subst "definitions" #definitionsLocal (subst "ref" #refLocal (subst "$range" #writes
    (Let "write" (App (Val (GoInstruction (GoAlloc pdg.Access.ty)))
      (Val #(zero_val pdg.Access))) trackingWritesLoop)))
end selectedSyntax

section execution
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [pdg.query.Assumptions] [pdg.Access.TypeAssumptions] [pdg.Position.TypeAssumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

/-- Execute all assignment-write appends. Zero writes skip; repeated symbols append again. -/
theorem wp_tracking_writes_normalized (definitionsLocal refLocal : Loc) (definitions writes : GoSlice)
    (rows : List GoSlice) (values : List pdg.Access) (ref : w32)
    (hvalid : ∀ write ∈ values, write.Symbol' = W32 0 ∨
      (0 < uint.Z write.Symbol' ∧ uint.Z write.Symbol' ≤ rows.length)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ definitionsLocal ↦ definitions ∗ refLocal ↦ ref ∗
      definitions ↦* rows ∗ trackingDefinitionRowsOwn rows ∗ writes ↦*□ values }}
      (trackingWritesNormalizedRun definitionsLocal refLocal writes)
    {{ RET executeVal; definitionsLocal ↦ definitions ∗ refLocal ↦ ref ∗
      ∃ (out : List GoSlice), definitions ↦* out ∗ trackingDefinitionRowsOwn out ∗ ⌜out.length = rows.length⌝ }} := by
  wp_start_folded as ⟨HdefinitionsLocal, Href, Hdefinitions, Hrows, #Hwrites⟩
  ihave %Hlen := ownSlice_len $$ Hwrites
  ihave %Hdlen := ownSlice_len $$ Hdefinitions
  unfold trackingWritesNormalizedRun trackingWritesLoop
  dsimp only [Let, Lam]
  wp_auto
  ihave HI : (∃ (iv : w64) (previous : pdg.Access) (current : List GoSlice),
      "i" ∷ i_ptr ↦ iv ∗ "write" ∷ write_ptr ↦ previous ∗
      "HdefinitionsLocal" ∷ definitionsLocal ↦ definitions ∗ "Href" ∷ refLocal ↦ ref ∗
      "Hdefinitions" ∷ definitions ↦* current ∗ "Hrows" ∷ trackingDefinitionRowsOwn current ∗
      "%Hshape" ∷ ⌜current.length = rows.length⌝ ∗
      "%Hbounds" ∷ ⌜0 ≤ sint.Z iv ∧ sint.Z iv ≤ sint.Z writes.len⌝ : IProp GF)
      $$ [i write HdefinitionsLocal Href Hdefinitions Hrows]
  · iexists (W64 0), (zero_val pdg.Access), rows
    iframe
    ipureintro; exact ⟨rfl, by constructor <;> word⟩
  wp_for HI
  wp_if_destruct
  · have hi : sint.nat iv < values.length := by word
    list_elem values (sint.nat iv) as entry
    rw [ite_eq_left (by constructor <;> word : 0 ≤ sint.Z iv ∧ sint.Z iv < sint.Z writes.len)]
    wp_apply wp_load_slice_index writes (sint.Z iv) values DFrac.discard entry (by word)
      $$ [Hwrites] with Hwrites
    · iframe Hwrites; ipureintro; exact Hentry_lookup
    try wp_auto
    by_cases hz : entry.Symbol' = W32 0
    · rw [decide_eq_true hz]
      try wp_auto
      wp_for_post
      iframe
      iexists (iv + W64 1), entry, current
      iframe
      ipureintro; exact ⟨Hshape, by constructor <;> word⟩
    · rw [decide_eq_false hz]
      try wp_auto
      have hk := (hvalid entry (List.mem_of_getElem? Hentry_lookup)).resolve_left hz
      have Hindex : sint.Z (W64 (uint.Z (entry.Symbol' - W32 1))) = uint.Z entry.Symbol' - 1 := by
        obtain ⟨_, _⟩ := hk; word
      rw [Hindex]
      rw [ite_eq_left (by obtain ⟨_, _⟩ := hk; constructor <;> word :
        0 ≤ uint.Z entry.Symbol' - 1 ∧ uint.Z entry.Symbol' - 1 < sint.Z definitions.len)]
      have hkindex : (uint.Z entry.Symbol' - 1).toNat < current.length := by omega
      list_elem current (uint.Z entry.Symbol' - 1).toNat as row
      wp_apply wp_load_slice_index definitions (uint.Z entry.Symbol' - 1) current (DFrac.own 1) row (by omega)
        $$ [Hdefinitions] with Hdefinitions
      · iframe; ipureintro; exact Hrow_lookup
      icases trackingDefinitionRowsOwn_acc current (uint.Z entry.Symbol' - 1).toNat row Hrow_lookup $$ Hrows
        with ⟨Hrow, Hrestore⟩
      iunfold trackingDefinitionOwn at Hrow
      icases Hrow with ⟨%old, Hrow, Hcap⟩
      try wp_auto
      wp_apply wp_slice_literal (V := w32) [ref]
      isplitr
      · ipureintro; rfl
      iintro %one ⟨Hone, -⟩
      try wp_auto
      wp_apply wp_slice_append $$ [Hrow Hcap Hone] with %out ⟨Hrow, Hcap, -⟩
      · iframe
      try wp_auto
      rw [Hindex]
      rw [ite_eq_left (by obtain ⟨_, _⟩ := hk; constructor <;> word :
        0 ≤ uint.Z entry.Symbol' - 1 ∧ uint.Z entry.Symbol' - 1 < sint.Z definitions.len)]
      wp_apply wp_store_slice_index definitions (uint.Z entry.Symbol' - 1) current out
        $$ [Hdefinitions] with Hdefinitions
      · iframe; ipureintro; constructor <;> omega
      ihave Hrows := Hrestore $$ %out [Hrow Hcap]
      · iunfold trackingDefinitionOwn
        iexists (old ++ [ref]); iframe
      wp_for_post
      iframe
      iexists (iv + W64 1), entry, (current.set (uint.Z entry.Symbol' - 1).toNat out)
      iframe
      ipureintro
      exact ⟨by simpa using Hshape, by constructor <;> word⟩
  · wp_end

end execution
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
