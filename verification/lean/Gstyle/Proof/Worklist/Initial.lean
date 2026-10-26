module
public import Gstyle.Proof.Worklist.SolveModel
@[expose] public section
noncomputable section
namespace Perennial
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [FfiSyntax]

/-- The entry has a fresh empty input map before the first enqueue. -/
def unqueuedWorkState (count : Nat) (entry : w32) : WorkState where
  inputs := (List.replicate count none).set (queueIndex entry) (some ∅)
  flags := List.replicate count false
  queue := []

/-- Exact contents after the production setup enqueues its one-based entry. -/
def initialWorkState (count : Nat) (entry : w32) : WorkState where
  inputs := (List.replicate count none).set (queueIndex entry) (some ∅)
  flags := (List.replicate count false).set (queueIndex entry) true
  queue := [entry]

theorem unqueuedWorkState_shape (count : Nat) (entry : w32) :
    (unqueuedWorkState count entry).inputs.length = count ∧
      (unqueuedWorkState count entry).flags.length = count := by
  simp [unqueuedWorkState]

theorem initialWorkState_shape (count : Nat) (entry : w32) :
    (initialWorkState count entry).inputs.length = count ∧
      (initialWorkState count entry).flags.length = count := by
  simp [initialWorkState]

theorem unqueuedWorkState_entry (count : Nat) (entry : w32)
    (href : 0 < uint.Z entry ∧ uint.Z entry ≤ count) :
    (unqueuedWorkState count entry).input (queueIndex entry) = some ∅ := by
  have hi : queueIndex entry < (List.replicate count false).length :=
    queueIndex_lt _ entry (by simpa [queueRefValid] using href)
  simp only [List.length_replicate] at hi
  simp [unqueuedWorkState, WorkState.input, hi]

theorem initialWorkState_entry (count : Nat) (entry : w32)
    (href : 0 < uint.Z entry ∧ uint.Z entry ≤ count) :
    (initialWorkState count entry).input (queueIndex entry) = some ∅ :=
  unqueuedWorkState_entry count entry href

/-- Every other input remains unreachable, including out-of-bounds lookups. -/
theorem initialWorkState_input_other (count : Nat) (entry : w32) (index : Nat)
    (hne : queueIndex entry ≠ index) :
    (initialWorkState count entry).input index = none := by
  simp only [initialWorkState, WorkState.input, List.getElem?_set_ne hne,
    List.getElem?_replicate]
  split <;> rfl

theorem initialWorkState_entry_flag (count : Nat) (entry : w32)
    (href : 0 < uint.Z entry ∧ uint.Z entry ≤ count) :
    (initialWorkState count entry).flags[queueIndex entry]? = some true := by
  have hi := queueIndex_lt (List.replicate count false) entry
    (by simpa [queueRefValid] using href)
  exact List.getElem?_set_self hi

theorem initialWorkState_flag_other (count : Nat) (entry : w32) (index : Nat)
    (hi : index < count) (hne : queueIndex entry ≠ index) :
    (initialWorkState count entry).flags[index]? = some false := by
  simp only [initialWorkState, List.getElem?_set_ne hne, List.getElem?_replicate,
    ite_eq_left hi]

theorem unqueuedWorkState_entry_flag (count : Nat) (entry : w32)
    (href : 0 < uint.Z entry ∧ uint.Z entry ≤ count) :
    (unqueuedWorkState count entry).flags[queueIndex entry]? = some false := by
  have hi := queueIndex_lt (List.replicate count false) entry
    (by simpa [queueRefValid] using href)
  simp only [List.length_replicate] at hi
  simp only [unqueuedWorkState, List.getElem?_replicate, ite_eq_left hi]

theorem unqueuedWorkState_enqueue (count : Nat) (entry : w32)
    (href : 0 < uint.Z entry ∧ uint.Z entry ≤ count) :
    (unqueuedWorkState count entry).enqueue entry = initialWorkState count entry := by
  simp only [WorkState.enqueue, unqueuedWorkState_entry_flag count entry href,
    Option.getD_some, Bool.false_eq_true, reduceIte]
  rfl

theorem validBranchState_empty (kinds : List GoString) (unstable : List Bool)
    (kind : w32 → GoString) (changing : w32 → Bool) :
    validBranchState kinds unstable kind changing (∅ : StateData) := by
  intro key hpresent
  simp at hpresent

theorem unqueuedWorkState_valid (s : QuerySnapshot) (kind : w32 → GoString)
    (changing : w32 → Bool) (count : Nat) (entry : w32) :
    validWorkInputs s kind changing (unqueuedWorkState count entry) := by
  intro p hp
  obtain old | fresh := List.mem_or_eq_of_mem_set hp
  · have he : p = none := List.eq_of_mem_replicate old
    subst p
    trivial
  · subst p
    exact validBranchState_empty s.kinds s.unstable kind changing

theorem unqueuedWorkState_scalar (count : Nat) (entry : w32) :
    scalarInputs (unqueuedWorkState count entry) := by
  intro p hp
  obtain old | fresh := List.mem_or_eq_of_mem_set hp
  · have he : p = none := List.eq_of_mem_replicate old
    subst p
    trivial
  · subst p
    exact scalarState_empty

theorem unqueuedWorkState_queueInvariant (count : Nat) (entry : w32) :
    QueueInvariant (unqueuedWorkState count entry).flags 0
      (unqueuedWorkState count entry).queue := by
  constructor
  · simp [unqueuedWorkState]
  · simp [unqueuedWorkState]
  · simp [unqueuedWorkState]
  · intro ref href
    have hi := queueIndex_lt (List.replicate count false) ref href
    simp only [List.length_replicate] at hi
    simp [unqueuedWorkState, hi]

theorem initialWorkState_queueInvariant (count : Nat) (entry : w32)
    (href : 0 < uint.Z entry ∧ uint.Z entry ≤ count) :
    QueueInvariant (initialWorkState count entry).flags 0
      (initialWorkState count entry).queue := by
  rw [← unqueuedWorkState_enqueue count entry href]
  apply WorkState.enqueue_invariant
  · exact unqueuedWorkState_queueInvariant count entry
  · simpa [queueRefValid, unqueuedWorkState] using href

theorem unqueuedWorkState_scalarWorkInvariant (s : QuerySnapshot)
    (kind : w32 → GoString) (changing : w32 → Bool) (count : Nat) (entry : w32) :
    scalarWorkInvariant s kind changing 0 (unqueuedWorkState count entry) :=
  ⟨unqueuedWorkState_queueInvariant count entry,
    unqueuedWorkState_valid s kind changing count entry,
    unqueuedWorkState_scalar count entry⟩

theorem initialWorkState_scalarWorkInvariant (s : QuerySnapshot)
    (kind : w32 → GoString) (changing : w32 → Bool) (count : Nat) (entry : w32)
    (href : 0 < uint.Z entry ∧ uint.Z entry ≤ count) :
    scalarWorkInvariant s kind changing 0 (initialWorkState count entry) :=
  ⟨initialWorkState_queueInvariant count entry href,
    unqueuedWorkState_valid s kind changing count entry,
    unqueuedWorkState_scalar count entry⟩

theorem unqueuedWorkState_reachableQueue (count : Nat) (entry : w32) :
    reachableQueue (unqueuedWorkState count entry) := by
  simp [reachableQueue, unqueuedWorkState]

theorem initialWorkState_reachableQueue (count : Nat) (entry : w32)
    (href : 0 < uint.Z entry ∧ uint.Z entry ≤ count) :
    reachableQueue (initialWorkState count entry) := by
  intro ref hmem
  have he : ref = entry := by simpa [initialWorkState] using hmem
  subst ref
  exact ⟨∅, initialWorkState_entry count entry href⟩

/-- Entry-last control: three blocks have exactly one reachable and pending input. -/
theorem initialWorkState_entry_last (s : QuerySnapshot) (kind : w32 → GoString)
    (changing : w32 → Bool) :
    (initialWorkState 3 (W32 3)).inputs = [none, none, some ∅] ∧
    (initialWorkState 3 (W32 3)).flags = [false, false, true] ∧
    (initialWorkState 3 (W32 3)).queue = [W32 3] ∧
    scalarWorkInvariant s kind changing 0 (initialWorkState 3 (W32 3)) ∧
    reachableQueue (initialWorkState 3 (W32 3)) := by
  have href : 0 < uint.Z (W32 3) ∧ uint.Z (W32 3) ≤ (3 : Nat) := by decide
  refine ⟨?_, ?_, rfl, initialWorkState_scalarWorkInvariant s kind changing 3 (W32 3) href,
    initialWorkState_reachableQueue 3 (W32 3) href⟩
  · rfl
  · rfl

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
