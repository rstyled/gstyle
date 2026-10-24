module
public import Gstyle.Proof.Worklist.ScalarDecision
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [FfiSyntax]
local instance inputCoverageDecidable : DecidableEq StateData := Classical.typeDecidableEq _

def WorkState.input (state : WorkState) (index : Nat) : Option StateData :=
  state.inputs[index]?.getD none

/-- Every environment represented before propagation stays represented. -/
def inputsExtend (kind : w32 → GoString) (changing : w32 → Bool)
    (before after : WorkState) : Prop :=
  ∀ index env, branchDenotes kind changing (before.input index) env →
    branchDenotes kind changing (after.input index) env

theorem inputsExtend_refl (kind : w32 → GoString) (changing : w32 → Bool) (state : WorkState) :
    inputsExtend kind changing state state := fun _ _ h => h

theorem inputsExtend_trans (kind : w32 → GoString) (changing : w32 → Bool)
    (a b c : WorkState) (hab : inputsExtend kind changing a b) (hbc : inputsExtend kind changing b c) :
    inputsExtend kind changing a c := fun index env h => hbc index env (hab index env h)

theorem WorkState.edge_extends (state : WorkState) (kind : w32 → GoString) (changing : w32 → Bool)
    (next : w32) (branch : Option StateData) (hi : queueIndex next < state.inputs.length) :
    inputsExtend kind changing state (state.edge kind changing next branch) := by
  cases branch with
  | none => exact inputsExtend_refl kind changing state
  | some n =>
    simp only [WorkState.edge]
    split
    · exact inputsExtend_refl kind changing state
    · intro index env hin
      have hinputs := (WorkState.enqueue_shape
        { state with inputs := (state.inputs.set (queueIndex next)
          (some (joinedInput kind changing (state.input (queueIndex next)) n))) } next).1
      unfold WorkState.input at hinputs ⊢
      rw [hinputs]
      by_cases he : queueIndex next = index
      · subst index
        simp only [List.getElem?_set_self hi, Option.getD_some]
        exact joinedInput_sound kind changing (state.input (queueIndex next)) n env (Or.inl hin)
      · simp only [List.getElem?_set_ne he]
        exact hin

theorem WorkState.edge_covers (state : WorkState) (kind : w32 → GoString) (changing : w32 → Bool)
    (next : w32) (branch : Option StateData) (hi : queueIndex next < state.inputs.length)
    (env : w32 → Int) (hin : branchDenotes kind changing branch env) :
    branchDenotes kind changing ((state.edge kind changing next branch).input (queueIndex next)) env := by
  cases branch with
  | none => exact False.elim hin
  | some n =>
    have hj := joinedInput_sound kind changing (state.input (queueIndex next)) n env (Or.inr hin)
    simp only [WorkState.edge]
    split
    · rename_i he
      change stateDenotes kind changing (joinedInput kind changing (state.input (queueIndex next)) n) env at hj
      change state.inputs[queueIndex next]?.getD none = some _ at he
      change branchDenotes kind changing (state.inputs[queueIndex next]?.getD none) env
      rw [he]
      exact hj
    · have hinputs := (WorkState.enqueue_shape
        { state with inputs := (state.inputs.set (queueIndex next)
          (some (joinedInput kind changing (state.input (queueIndex next)) n))) } next).1
      unfold WorkState.input at hinputs ⊢
      rw [hinputs]
      simpa only [List.getElem?_set_self hi, Option.getD_some, branchDenotes, WorkState.input] using hj

def scalarInputs (state : WorkState) : Prop := ∀ p ∈ state.inputs, scalarOption p

theorem WorkState.edge_scalar (state : WorkState) (kind : w32 → GoString) (changing : w32 → Bool)
    (next : w32) (branch : Option StateData) (hs : scalarInputs state) (hb : scalarOption branch) :
    scalarInputs (state.edge kind changing next branch) := by
  cases branch with
  | none => exact hs
  | some n =>
    simp only [WorkState.edge]
    split
    · exact hs
    · have hp : scalarOption (state.input (queueIndex next)) := by
        cases h : state.inputs[queueIndex next]? with
        | none => simp only [WorkState.input, h, Option.getD_none, scalarOption]
        | some p => simpa only [WorkState.input, h, Option.getD_some] using hs p (List.mem_of_getElem? h)
      have hj := joinedInput_scalar kind changing (state.input (queueIndex next)) n hp hb
      have hinputs := (WorkState.enqueue_shape
        { state with inputs := (state.inputs.set (queueIndex next)
          (some (joinedInput kind changing (state.input (queueIndex next)) n))) } next).1
      unfold WorkState.input at hinputs
      unfold scalarInputs
      rw [hinputs]
      intro p hmem
      obtain old | new := List.mem_or_eq_of_mem_set hmem
      · exact hs p old
      · subst p; exact hj

theorem edgePrefix_scalar (initial : WorkState) (nexts : List w32)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (decision : Bool → StateData → Option StateData) (source : StateData) (i : Nat)
    (hs : scalarInputs initial) (hb : ∀ truth, scalarOption (decision truth source)) :
    scalarInputs (edgePrefix initial nexts kind changing decision source i) := by
  induction i with
  | zero => exact hs
  | succ i ih =>
    simp only [edgePrefix]
    cases hn : nexts[i]? with
    | none => exact ih
    | some next => exact WorkState.edge_scalar _ kind changing next _ ih (hb _)

theorem edgePrefix_extends_step (initial : WorkState) (nexts : List w32)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (decision : Bool → StateData → Option StateData) (source : StateData) (i : Nat)
    (hrefs : ∀ next ∈ nexts, queueIndex next < initial.inputs.length) :
    inputsExtend kind changing (edgePrefix initial nexts kind changing decision source i)
      (edgePrefix initial nexts kind changing decision source (i + 1)) := by
  simp only [edgePrefix]
  cases hn : nexts[i]? with
  | none => exact inputsExtend_refl kind changing _
  | some next =>
    apply WorkState.edge_extends
    rw [(edgePrefix_shape initial nexts kind changing decision source i).1]
    exact hrefs next (List.mem_of_getElem? hn)

theorem edgePrefix_extends (initial : WorkState) (nexts : List w32)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (decision : Bool → StateData → Option StateData) (source : StateData) (i j : Nat)
    (hle : i ≤ j) (hrefs : ∀ next ∈ nexts, queueIndex next < initial.inputs.length) :
    inputsExtend kind changing (edgePrefix initial nexts kind changing decision source i)
      (edgePrefix initial nexts kind changing decision source j) := by
  induction j generalizing i with
  | zero =>
    have hi : i = 0 := by omega
    subst i
    exact inputsExtend_refl kind changing _
  | succ j ih =>
    by_cases he : i = j + 1
    · subst i; exact inputsExtend_refl kind changing _
    · apply inputsExtend_trans kind changing _ (edgePrefix initial nexts kind changing decision source j) _
      · exact ih i (by omega)
      · exact edgePrefix_extends_step initial nexts kind changing decision source j hrefs

/-- Repeated targets preserve earlier incoming environments because every later
    edge extends the input. Outcome zero is the actual true-edge convention. -/
theorem edgePrefix_covers (initial : WorkState) (nexts : List w32)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (decision : Bool → StateData → Option StateData) (source : StateData)
    (i count : Nat) (next : w32) (env : w32 → Int)
    (hrefs : ∀ next ∈ nexts, queueIndex next < initial.inputs.length)
    (hnext : nexts[i]? = some next) (hi : i < count)
    (hin : branchDenotes kind changing (decision (decide (i = 0)) source) env) :
    branchDenotes kind changing
      ((edgePrefix initial nexts kind changing decision source count).input (queueIndex next)) env := by
  have hb : queueIndex next < (edgePrefix initial nexts kind changing decision source i).inputs.length := by
    rw [(edgePrefix_shape initial nexts kind changing decision source i).1]
    exact hrefs next (List.mem_of_getElem? hnext)
  have he := WorkState.edge_covers (edgePrefix initial nexts kind changing decision source i)
    kind changing next (decision (decide (i = 0)) source) hb env hin
  rw [← edgePrefix_step initial nexts kind changing decision source i next hnext] at he
  exact edgePrefix_extends initial nexts kind changing decision source (i + 1) count (by omega) hrefs
    (queueIndex next) env he

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
