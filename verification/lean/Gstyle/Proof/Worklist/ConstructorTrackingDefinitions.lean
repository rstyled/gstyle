module
public import Gstyle.Proof.Worklist.ConstructorTracking
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
section ownership
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics]

/-- The definition rows are local scratch data. The scalar queue never consumes
their contents, but every append still needs ownership of its real buffer. -/
def trackingDefinitionOwn (row : GoSlice) : IProp GF := iprop(
  ∃ values : List w32, row ↦* values ∗ ownSliceCap w32 row (DFrac.own 1))

def trackingDefinitionRowsOwn : List GoSlice → IProp GF
  | [] => iprop(True)
  | row :: rows => iprop(trackingDefinitionOwn row ∗ trackingDefinitionRowsOwn rows)

theorem trackingDefinitionRowsOwn_acc (rows : List GoSlice) (index : Nat) (row : GoSlice)
    (hrow : rows[index]? = some row) :
    trackingDefinitionRowsOwn (GF := GF) rows ⊢ trackingDefinitionOwn row ∗
      (∀ next : GoSlice, trackingDefinitionOwn next -∗ trackingDefinitionRowsOwn (rows.set index next)) := by
  induction rows generalizing index with
  | nil => simp at hrow
  | cons head rows ih =>
    cases index with
    | zero =>
      simp only [List.getElem?_cons_zero, Option.some.injEq] at hrow
      subst row
      simp only [List.set_cons_zero, trackingDefinitionRowsOwn]
      iintro ⟨Hhead, Htail⟩
      iframe Hhead
      iintro %next Hnext
      iframe Hnext Htail
    | succ index =>
      simp only [List.getElem?_cons_succ] at hrow
      simp only [trackingDefinitionRowsOwn]
      iintro ⟨Hhead, Htail⟩
      ihave ⟨Hslot, Hrestore⟩ := ih index hrow $$ Htail
      iframe Hslot
      iintro %next Hnext
      ihave Htail := Hrestore $$ %next Hnext
      simp only [List.set_cons_succ, trackingDefinitionRowsOwn]
      iframe Hhead Htail

theorem trackingDefinitionRowsOwn_nil (count : Nat) :
    ⊢ trackingDefinitionRowsOwn (GF := GF) (List.replicate count slice.nil) := by
  induction count with
  | zero => simp only [List.replicate_zero, trackingDefinitionRowsOwn]; ipureintro; trivial
  | succ count ih =>
    simp only [List.replicate_succ, trackingDefinitionRowsOwn, trackingDefinitionOwn]
    isplitr
    · iexists ([] : List w32)
      isplitr
      · iapply ownSlice_nil
      · iapply ownSliceCap_nil
    · exact ih

end ownership
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
