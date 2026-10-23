module
public import Gstyle.Proof.Transfer.Branch
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [Value.TypeAssumptions] [State.TypeAssumptions] [pdg.query.Assumptions]

/-- Every non-nil input map has separate ownership. Nil means unreachable, not an empty map. -/
def inputStatesOwn : List Loc → List (Option StateData) → IProp GF
  | [], [] => iprop(True)
  | pointer :: pointers, model :: models => iprop(ownsBranch model pointer ∗ inputStatesOwn pointers models)
  | _, _ => iprop(⌜False⌝)

theorem inputStatesOwn_length (pointers : List Loc) (models : List (Option StateData)) :
    inputStatesOwn (GF := GF) pointers models ⊢ ⌜pointers.length = models.length⌝ := by
  induction pointers generalizing models with
  | nil =>
    cases models with
    | nil => unfold inputStatesOwn; iintro _; ipureintro; rfl
    | cons model models => unfold inputStatesOwn; iintro %h; exact h.elim
  | cons pointer pointers ih =>
    cases models with
    | nil => unfold inputStatesOwn; iintro %h; exact h.elim
    | cons model models =>
      unfold inputStatesOwn
      iintro ⟨_, Htail⟩
      ihave %hlen := ih models $$ Htail
      ipureintro
      simpa using congrArg Nat.succ hlen

/-- Update one input slot without losing the maps owned by all other slots. -/
theorem inputStatesOwn_acc (pointers : List Loc) (models : List (Option StateData))
    (index : Nat) (pointer : Loc) (model : Option StateData)
    (hp : pointers[index]? = some pointer) (hm : models[index]? = some model) :
    inputStatesOwn (GF := GF) pointers models ⊢ ownsBranch model pointer ∗
      (∀ (next : Loc) (value : Option StateData), ownsBranch value next -∗
        inputStatesOwn (pointers.set index next) (models.set index value)) := by
  induction pointers generalizing models index with
  | nil => simp at hp
  | cons head pointers ih =>
    cases models with
    | nil => simp at hm
    | cons current models =>
      cases index with
      | zero =>
        simp only [List.getElem?_cons_zero, Option.some.injEq] at hp hm
        subst pointer
        subst model
        simp only [List.set_cons_zero, inputStatesOwn]
        iintro ⟨Hhead, Htail⟩
        iframe Hhead
        iintro %next %value Hnext
        iframe Hnext Htail
      | succ index =>
        simp only [List.getElem?_cons_succ] at hp hm
        simp only [inputStatesOwn]
        iintro ⟨Hhead, Htail⟩
        ihave ⟨Hslot, Hrestore⟩ := ih models index hp hm $$ Htail
        iframe Hslot
        iintro %next %value Hnext
        ihave Htail := Hrestore $$ %next %value Hnext
        simp only [List.set_cons_succ, inputStatesOwn]
        iframe Hhead Htail

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
