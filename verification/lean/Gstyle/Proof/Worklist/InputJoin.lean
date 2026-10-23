module
public import Gstyle.Proof.Worklist.Inputs
public import Gstyle.Proof.Worklist.Successor
public import Gstyle.Proof.StateMap.Equality
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

/-- An unreachable input acquires the incoming map; a reachable input joins it. -/
def joinedInput [FfiSyntax] (kind : w32 → GoString) (changing : w32 → Bool)
    (previous : Option StateData) (next : StateData) : StateData :=
  match previous with
  | none => next
  | some p => joinedState kind changing p next

theorem joinedInput_valid [FfiSyntax] (s : QuerySnapshot) (kind : w32 → GoString) (changing : w32 → Bool)
    (previous : Option StateData) (next : StateData)
    (hp : validBranchOption s.kinds s.unstable kind changing previous)
    (hn : validBranchState s.kinds s.unstable kind changing next) :
    validBranchState s.kinds s.unstable kind changing (joinedInput kind changing previous next) := by
  cases previous with
  | none => exact hn
  | some p =>
    exact joined_branch_valid s.kinds s.unstable kind changing p next
      (joinedState kind changing p next) hp hn (fun _ => rfl)

theorem joinedInput_sound [FfiSyntax] (kind : w32 → GoString) (changing : w32 → Bool)
    (previous : Option StateData) (next : StateData) (env : w32 → Int)
    (hin : branchDenotes kind changing previous env ∨ stateDenotes kind changing next env) :
    stateDenotes kind changing (joinedInput kind changing previous next) env := by
  cases previous with
  | none => simpa [branchDenotes, joinedInput] using hin
  | some p =>
    exact fun key => state_join_sound kind changing p next (joinedState kind changing p next) key (env key)
      (fun _ => rfl) (hin.elim (fun h => Or.inl (h key)) (fun h => Or.inr (h key)))

local instance inputModelDecidable [FfiSyntax] : DecidableEq StateData := Classical.typeDecidableEq _

def inputEqual [FfiSyntax] (previous : Option StateData) (next : StateData) : Bool :=
  match previous with
  | none => decide (∅ = next)
  | some p => decide (p = next)

theorem inputEqual_of_same [FfiSyntax] (previous : Option StateData) (next : StateData)
    (same : previous = some next) : inputEqual previous next = true := by
  rw [same]
  simp [inputEqual]

theorem inputEqual_changed_nil [FfiSyntax] (previous : Option StateData) (next : StateData)
    (equal : inputEqual previous next = true) (changed : previous ≠ some next) : previous = none := by
  cases previous with
  | none => rfl
  | some p =>
    have he : p = next := by simpa [inputEqual] using equal
    exact False.elim (changed (congrArg some he))

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [graphView.TypeAssumptions] [pdg.query.Assumptions]
variable [pdg.Graph.TypeAssumptions] [pdg.Node.TypeAssumptions] [pdg.Attributes.TypeAssumptions]
variable [pdg.AttributeSet.TypeAssumptions] [pdg.AttributeTables.TypeAssumptions]
variable [pdg.Producer.TypeAssumptions] [pdg.Function.TypeAssumptions] [pdg.Span.TypeAssumptions]
variable [pdg.CountFact.TypeAssumptions] [pdg.Conformance.TypeAssumptions] [pdg.Fact.TypeAssumptions]
variable [pdg.Position.TypeAssumptions] [pdg.CallForm.TypeAssumptions] [pdg.ExitKind.TypeAssumptions]
variable [pdg.ConformanceStatus.TypeAssumptions] [pdg.Category.TypeAssumptions]
variable [FuncUnfold pdg.listAt [pdg.Ref.ty] (pdg.listAt.impl pdg.Ref.ty)]
variable [FuncUnfold pdg.listAt [pdg.Access.ty] (pdg.listAt.impl pdg.Access.ty)]
variable [FuncUnfold pdg.listAt [pdg.Extension.ty] (pdg.listAt.impl pdg.Extension.ty)]
variable [FuncUnfold equalState [] equalState.impl]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

variable [pdg.Access.TypeAssumptions] [pdg.Tables.TypeAssumptions] [pdg.Extension.TypeAssumptions]
variable [FuncUnfold directBindingRead [] directBindingRead.impl]
variable [FuncUnfold operatorText [] operatorText.impl]


variable [FuncUnfold extension [] extension.impl]

variable [Value.TypeAssumptions] [State.TypeAssumptions]

variable [FuncUnfold exactInteger [] exactInteger.impl]
variable [FuncUnfold integerConstraint [] integerConstraint.impl]
variable [FuncUnfold lengthConstraint [] lengthConstraint.impl]
variable [FuncUnfold emptyRange [] emptyRange.impl]
variable [interval.Assumptions]
variable [FuncUnfold compareExactOrder [] compareExactOrder.impl]
variable [FuncUnfold compareExactIntegers [] compareExactIntegers.impl]
variable [FuncUnfold booleanContradiction [] booleanContradiction.impl]
variable [FuncUnfold copyState [] copyState.impl]


/-- Actual join, with nil reachability and exact map contents preserved. -/
theorem wp_join_input (receiver : Loc) (s : QuerySnapshot)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (previous next : Loc) (p : Option StateData) (n : StateData)
    (hlen : s.unstable.length = s.kinds.length)
    (hp : validBranchOption s.kinds s.unstable kind changing p)
    (hn : validBranchState s.kinds s.unstable kind changing n) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗
       isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
       (querySnapshotOwn receiver s ∗ ownsBranch p previous ∗ next ↦$ n) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"join" #receiver))
        (Val #previous)) (Val #next))
    {{ (out : Loc), RET #out; querySnapshotOwn receiver s ∗ ownsBranch p previous ∗ next ↦$ n ∗
        out ↦$ (joinedInput kind changing p n) }} := by
  cases p with
  | none =>
    simp only [ownsBranch, joinedInput]
    wp_start_folded as ⟨#Hsnapshot, %hprevious, Hnext⟩
    subst previous
    wp_apply wp_join_left_nil receiver next n $$ [Hnext] with %out ⟨Hnext, Hout⟩
    · iframe Hnext
    wp_end
    all_goals iframe Hsnapshot Hnext Hout
    all_goals ipureintro; rfl
  | some p =>
    simp only [ownsBranch, joinedInput]
    wp_start_folded as ⟨#Hsnapshot, Hprevious, Hnext⟩
    ihave #Hctx := Hsnapshot
    iunfold querySnapshotOwn at Hctx
    icases Hctx with ⟨Hview, _, _, _, Hkinds, Hunstable, _, _, _, _, _, _, _, _, _, _, _⟩
    have hmeta : ∀ key, (p !! key).isSome ∨ (n !! key).isSome →
        (0 < uint.Z key ∧ uint.Z key ≤ s.kinds.length) ∧
        s.kinds[(uint.Z key - 1).toNat]? = some (kind key) ∧
        s.unstable[(uint.Z key - 1).toNat]? = some (changing key) := by
      intro key hkey
      exact hkey.elim (hp key) (hn key)
    wp_apply wp_join receiver s.view s.kinds s.unstable kind changing previous next p n hlen hmeta DFrac.discard
        $$ [Hview Hkinds Hunstable Hprevious Hnext]
        with %out ⟨%om, Hview, Hkinds, Hunstable, Hprevious, Hnext, Hout, %hcontents⟩
    · iframe Hview Hkinds Hunstable Hprevious Hnext
    have hmap : om = joinedState kind changing p n := GMap.ext hcontents
    subst om
    wp_end
    all_goals iframe Hsnapshot Hprevious Hnext Hout


/-- The real map equality call, including Go's nil-versus-empty equality behavior. -/
theorem wp_equal_input (previous next : Loc) (p : Option StateData) (n : StateData) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (ownsBranch p previous ∗ next ↦$ n) }}
      (App (App (Val (@! equalState)) (Val #previous)) (Val #next))
    {{ RET #(inputEqual p n); ownsBranch p previous ∗ next ↦$ n ∗ ⌜previous = map.nil ↔ p = none⌝ }} := by
  cases p with
  | none =>
    simp only [ownsBranch, inputEqual]
    wp_start_folded as ⟨%hprevious, Hnext⟩
    subst previous
    wp_apply wp_equalState_left_nil next n (DFrac.own 1) $$ [Hnext] with Hnext
    · iframe Hnext
    wp_end
    all_goals iframe Hnext
    all_goals ipureintro; simp
  | some p =>
    simp only [ownsBranch, inputEqual]
    wp_start_folded as ⟨Hprevious, Hnext⟩
    ihave %Hnon := ownMap_not_nil previous p (DFrac.own 1) $$ Hprevious
    wp_apply wp_equalState previous next p n (DFrac.own 1) (DFrac.own 1)
        $$ [Hprevious Hnext] with ⟨Hprevious, Hnext⟩
    · iframe Hprevious Hnext
    wp_end
    all_goals iframe Hprevious Hnext
    all_goals ipureintro; simp [Hnon]

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
