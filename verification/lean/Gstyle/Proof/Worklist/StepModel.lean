module
public import Gstyle.Proof.Worklist.InputJoin
public import Gstyle.Proof.Worklist.Queue
@[expose] public section
noncomputable section
namespace Perennial
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [FfiSyntax]
local instance workStateDataDecidable : DecidableEq StateData := Classical.typeDecidableEq _

/-- Logical contents of the three mutable worklist containers. -/
structure WorkState where
  inputs : List (Option StateData)
  flags : List Bool
  queue : List w32

def WorkState.enqueue (state : WorkState) (ref : w32) : WorkState :=
  if state.flags[queueIndex ref]?.getD false then state else
    { state with flags := state.flags.set (queueIndex ref) true, queue := state.queue ++ [ref] }

/-- The production nil guard precedes join; an equal reachable join does not enqueue. -/
def WorkState.edge (state : WorkState) (kind : w32 → GoString) (changing : w32 → Bool)
    (next : w32) (branch : Option StateData) : WorkState :=
  match branch with
  | none => state
  | some n =>
    let previous := state.inputs[queueIndex next]?.getD none
    let joined := joinedInput kind changing previous n
    if previous = some joined then state else
      { state with inputs := state.inputs.set (queueIndex next) (some joined) }.enqueue next

/-- Prefixes follow the actual successor order; outcome zero is the true edge. -/
def edgePrefix (initial : WorkState) (nexts : List w32)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (decision : Bool → StateData → Option StateData) (source : StateData) : Nat → WorkState
  | 0 => initial
  | i + 1 =>
    let previous := edgePrefix initial nexts kind changing decision source i
    match nexts[i]? with
    | none => previous
    | some next => previous.edge kind changing next (decision (decide (i = 0)) source)

theorem edgePrefix_step (initial : WorkState) (nexts : List w32)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (decision : Bool → StateData → Option StateData) (source : StateData)
    (i : Nat) (next : w32) (hnext : nexts[i]? = some next) :
    edgePrefix initial nexts kind changing decision source (i + 1) =
      (edgePrefix initial nexts kind changing decision source i).edge kind changing next
        (decision (decide (i = 0)) source) := by
  simp only [edgePrefix, hnext]

theorem WorkState.enqueue_shape (state : WorkState) (ref : w32) :
    (state.enqueue ref).inputs = state.inputs ∧
      (state.enqueue ref).flags.length = state.flags.length := by
  unfold enqueue
  split <;> simp

theorem WorkState.enqueue_invariant (state : WorkState) (ref : w32) (cursor : Nat)
    (h : QueueInvariant state.flags cursor state.queue) (href : queueRefValid state.flags ref) :
    QueueInvariant (state.enqueue ref).flags cursor (state.enqueue ref).queue := by
  have hi := queueIndex_lt state.flags ref href
  have hf := List.getElem?_eq_getElem hi
  cases hv : state.flags[queueIndex ref]
  · have hfalse : state.flags[queueIndex ref]? = some false := by simpa [hv] using hf
    simpa [enqueue, hfalse] using QueueInvariant.enqueue state.flags cursor state.queue ref h href hfalse
  · have htrue : state.flags[queueIndex ref]? = some true := by simpa [hv] using hf
    simpa [enqueue, htrue] using h

theorem WorkState.edge_shape (state : WorkState) (kind : w32 → GoString) (changing : w32 → Bool)
    (next : w32) (branch : Option StateData) :
    (state.edge kind changing next branch).inputs.length = state.inputs.length ∧
      (state.edge kind changing next branch).flags.length = state.flags.length := by
  cases branch with
  | none => exact ⟨rfl, rfl⟩
  | some n =>
    simp only [edge]
    split
    · exact ⟨rfl, rfl⟩
    · have h := enqueue_shape { state with inputs := (state.inputs.set (queueIndex next)
        (some (joinedInput kind changing (state.inputs[queueIndex next]?.getD none) n))) } next
      simpa using And.intro (congrArg List.length h.1) h.2

theorem WorkState.edge_queue_invariant (state : WorkState) (kind : w32 → GoString) (changing : w32 → Bool)
    (next : w32) (branch : Option StateData) (cursor : Nat)
    (h : QueueInvariant state.flags cursor state.queue) (href : queueRefValid state.flags next) :
    QueueInvariant (state.edge kind changing next branch).flags cursor
      (state.edge kind changing next branch).queue := by
  cases branch with
  | none => exact h
  | some n =>
    simp only [edge]
    split
    · exact h
    · exact enqueue_invariant _ next cursor h href


def validWorkInputs (s : QuerySnapshot) (kind : w32 → GoString) (changing : w32 → Bool)
    (state : WorkState) : Prop :=
  ∀ p ∈ state.inputs, validBranchOption s.kinds s.unstable kind changing p

theorem WorkState.edge_valid (s : QuerySnapshot) (state : WorkState)
    (kind : w32 → GoString) (changing : w32 → Bool) (next : w32) (branch : Option StateData)
    (hm : validWorkInputs s kind changing state)
    (hb : validBranchOption s.kinds s.unstable kind changing branch) :
    validWorkInputs s kind changing (state.edge kind changing next branch) := by
  cases branch with
  | none => exact hm
  | some n =>
    simp only [edge]
    split
    · exact hm
    · have hp : validBranchOption s.kinds s.unstable kind changing
          (state.inputs[queueIndex next]?.getD none) := by
        cases h : state.inputs[queueIndex next]? with
        | none => simp [h, validBranchOption]
        | some p => simpa [h] using hm p (List.mem_of_getElem? h)
      have hj := joinedInput_valid s kind changing _ n hp hb
      have he := enqueue_shape { state with inputs := (state.inputs.set (queueIndex next)
        (some (joinedInput kind changing (state.inputs[queueIndex next]?.getD none) n))) } next
      unfold validWorkInputs
      rw [he.1]
      intro p hmem
      obtain h | h := List.mem_or_eq_of_mem_set hmem
      · exact hm p h
      · subst p; exact hj


theorem edgePrefix_shape (initial : WorkState) (nexts : List w32)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (decision : Bool → StateData → Option StateData) (source : StateData) (i : Nat) :
    (edgePrefix initial nexts kind changing decision source i).inputs.length = initial.inputs.length ∧
    (edgePrefix initial nexts kind changing decision source i).flags.length = initial.flags.length := by
  induction i with
  | zero => exact ⟨rfl, rfl⟩
  | succ i ih =>
    simp only [edgePrefix]
    cases hn : nexts[i]? with
    | none => exact ih
    | some next =>
      have hs := WorkState.edge_shape (edgePrefix initial nexts kind changing decision source i)
        kind changing next (decision (decide (i = 0)) source)
      exact ⟨hs.1.trans ih.1, hs.2.trans ih.2⟩

theorem edgePrefix_valid (s : QuerySnapshot) (initial : WorkState) (nexts : List w32)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (decision : Bool → StateData → Option StateData) (source : StateData) (i : Nat)
    (hm : validWorkInputs s kind changing initial)
    (hb : ∀ truth, validBranchOption s.kinds s.unstable kind changing (decision truth source)) :
    validWorkInputs s kind changing (edgePrefix initial nexts kind changing decision source i) := by
  induction i with
  | zero => exact hm
  | succ i ih =>
    simp only [edgePrefix]
    cases hn : nexts[i]? with
    | none => exact ih
    | some next => exact WorkState.edge_valid s _ kind changing next _ ih (hb _)


theorem edgePrefix_queue_invariant (initial : WorkState) (nexts : List w32)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (decision : Bool → StateData → Option StateData) (source : StateData) (i cursor : Nat)
    (hq : QueueInvariant initial.flags cursor initial.queue)
    (hrefs : ∀ next ∈ nexts, queueRefValid initial.flags next) :
    QueueInvariant (edgePrefix initial nexts kind changing decision source i).flags cursor
      (edgePrefix initial nexts kind changing decision source i).queue := by
  induction i with
  | zero => exact hq
  | succ i ih =>
    simp only [edgePrefix]
    cases hn : nexts[i]? with
    | none => exact ih
    | some next =>
      have hvalid := hrefs next (List.mem_of_getElem? hn)
      have hs := (edgePrefix_shape initial nexts kind changing decision source i).2
      have hr : queueRefValid (edgePrefix initial nexts kind changing decision source i).flags next := by
        unfold queueRefValid
        rw [hs]
        exact hvalid
      exact WorkState.edge_queue_invariant _ kind changing next _ cursor ih hr

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
