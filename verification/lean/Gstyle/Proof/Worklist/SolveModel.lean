module
public import Gstyle.Proof.Worklist.ScalarTraversal
public import Gstyle.Proof.Worklist.ScalarBlockMeaning
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

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


variable [analysis.TypeAssumptions] [pdg.NormalFlow.TypeAssumptions] [pdg.FlowBlock.TypeAssumptions]
variable [FuncUnfold equalState [] equalState.impl]



/-- The actual dequeue changes only the selected pending flag. -/
def WorkState.dequeue (state : WorkState) (ref : w32) : WorkState :=
  { state with flags := state.flags.set (queueIndex ref) false }

/-- Every queued block, including processed entries, has a reachable input.
    Flags alone are insufficient to establish this relationship. -/
def reachableQueue (state : WorkState) : Prop :=
  ∀ ref ∈ state.queue, ∃ m, state.input (queueIndex ref) = some m

/-- Operation/value and decision execution data for one real solver iteration.
    No child correctness theorem or report acceptance is stored here. -/
structure SolverStep (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (blocks : List pdg.FlowBlock) (operations nexts : w32 → List w32)
    (cursor : Nat) (initial : WorkState) where
  ref : w32
  block : pdg.FlowBlock
  sourceMap : StateData
  targetMap : StateData
  result : Bool → StateData → Option StateData
  head : initial.queue[cursor]? = some ref
  blockAt : blocks[queueIndex ref]? = some block
  inputAt : initial.input (queueIndex ref) = some sourceMap
  transfer : ScalarExecution s (operations ref) sourceMap targetMap
  decision : DecisionEvidence s o kind changing block.Condition' result

def SolverStep.next (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (blocks : List pdg.FlowBlock) (operations nexts : w32 → List w32)
    (cursor : Nat) (initial : WorkState)
    (step : SolverStep s o kind changing blocks operations nexts cursor initial) : WorkState :=
  edgePrefix (initial.dequeue step.ref) (nexts step.ref) kind changing step.result step.targetMap
    (nexts step.ref).length

inductive SolverExecution (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (blocks : List pdg.FlowBlock) (operations nexts : w32 → List w32) :
    Nat → WorkState → WorkState → Prop where
  | done (cursor : Nat) (state : WorkState) (exhausted : cursor = state.queue.length) :
      SolverExecution s o kind changing blocks operations nexts cursor state state
  | step (cursor : Nat) (initial final : WorkState)
      (current : SolverStep s o kind changing blocks operations nexts cursor initial)
      (rest : SolverExecution s o kind changing blocks operations nexts (cursor + 1)
        (current.next s o kind changing blocks operations nexts cursor initial) final) :
      SolverExecution s o kind changing blocks operations nexts cursor initial final

variable (s : QuerySnapshot) (o : AssumeOperators)
variable (kind : w32 → GoString) (changing : w32 → Bool)
variable (blocks : List pdg.FlowBlock) (operations nexts : w32 → List w32)

theorem WorkState.dequeue_invariant (s : QuerySnapshot) (kind : w32 → GoString)
    (changing : w32 → Bool) (cursor : Nat) (state : WorkState) (ref : w32)
    (h : scalarWorkInvariant s kind changing cursor state)
    (head : state.queue[cursor]? = some ref) :
    scalarWorkInvariant s kind changing (cursor + 1) (state.dequeue ref) := by
  exact ⟨h.1.advance state.flags cursor state.queue ref head, h.2.1, h.2.2⟩

theorem SolverStep.reference
    (cursor : Nat) (initial : WorkState)
    (step : SolverStep s o kind changing blocks operations nexts cursor initial)
    (hq : QueueInvariant initial.flags cursor initial.queue) : queueRefValid initial.flags step.ref := by
  have hc : cursor < initial.queue.length := (List.getElem?_eq_some_iff.mp step.head).choose
  apply hq.references
  rw [List.drop_eq_getElem_cons hc, (List.getElem?_eq_some_iff.mp step.head).choose_spec]
  simp

theorem SolverStep.input_valid
    (cursor : Nat) (initial : WorkState)
    (step : SolverStep s o kind changing blocks operations nexts cursor initial)
    (hm : validWorkInputs s kind changing initial) :
    validBranchState s.kinds s.unstable kind changing step.sourceMap := by
  have h : some step.sourceMap ∈ initial.inputs := by
    have hi := step.inputAt
    unfold WorkState.input at hi
    cases hv : initial.inputs[queueIndex step.ref]? with
    | none => simp only [hv, Option.getD_none] at hi; cases hi
    | some value =>
      simp only [hv, Option.getD_some] at hi
      exact hi ▸ List.mem_of_getElem? hv
  exact hm (some step.sourceMap) h


theorem ScalarExecution.scalar (s : QuerySnapshot) (operations : List w32) (m n : StateData)
    (execution : ScalarExecution s operations m n) (hm : scalarState m) : scalarState n := by
  induction execution with
  | nil => exact hm
  | cons ref tail m middle n step rest ih => exact ih (step.preserves_scalar s ref m middle hm)

theorem SolverStep.input_scalar
    (cursor : Nat) (initial : WorkState)
    (step : SolverStep s o kind changing blocks operations nexts cursor initial)
    (hm : scalarInputs initial) : scalarState step.sourceMap := by
  have hi := step.inputAt
  unfold WorkState.input at hi
  cases hv : initial.inputs[queueIndex step.ref]? with
  | none => simp only [hv, Option.getD_none] at hi; cases hi
  | some value =>
    simp only [hv, Option.getD_some] at hi
    have h := hm value (List.mem_of_getElem? hv)
    simpa only [hi, scalarOption] using h

theorem SolverStep.next_shape
    (cursor : Nat) (initial : WorkState)
    (step : SolverStep s o kind changing blocks operations nexts cursor initial) :
    (step.next s o kind changing blocks operations nexts cursor initial).inputs.length = initial.inputs.length ∧
    (step.next s o kind changing blocks operations nexts cursor initial).flags.length = initial.flags.length := by
  simpa only [SolverStep.next, WorkState.dequeue, List.length_set] using
    edgePrefix_shape (initial.dequeue step.ref) (nexts step.ref) kind changing step.result step.targetMap
      (nexts step.ref).length

theorem SolverStep.next_invariant
    (cursor : Nat) (initial : WorkState)
    (step : SolverStep s o kind changing blocks operations nexts cursor initial)
    (hm : scalarWorkInvariant s kind changing cursor initial)
    (agreement : metadataAgreement s kind changing)
    (hk : ∀ key, kind key ≠ go!"\"pointer\"")
    (hrefs : ∀ next ∈ nexts step.ref, queueRefValid initial.flags next) :
    scalarWorkInvariant s kind changing (cursor + 1)
      (step.next s o kind changing blocks operations nexts cursor initial) := by
  have hd := WorkState.dequeue_invariant s kind changing cursor initial step.ref hm step.head
  have hi := step.input_valid s o kind changing blocks operations nexts cursor initial hm.2.1
  have hn := step.transfer.valid s kind changing (operations step.ref) step.sourceMap step.targetMap agreement hi
  have hs := step.transfer.scalar s (operations step.ref) step.sourceMap step.targetMap
    (step.input_scalar s o kind changing blocks operations nexts cursor initial hm.2.2)
  have hnexts : ∀ next ∈ nexts step.ref, queueRefValid (initial.dequeue step.ref).flags next := by
    simpa only [WorkState.dequeue, queueRefValid, List.length_set] using hrefs
  exact ⟨edgePrefix_queue_invariant _ _ _ _ _ _ _ _ hd.1 hnexts,
    edgePrefix_valid _ _ _ _ _ _ _ _ hd.2.1
      (fun truth => DecisionEvidence.valid s o kind changing step.block.Condition' step.result step.decision truth step.targetMap hn),
    edgePrefix_scalar _ _ _ _ _ _ _ hd.2.2
      (fun truth => DecisionEvidence.scalar s o kind changing step.block.Condition' step.result step.decision truth step.targetMap hn hs hk)⟩


theorem WorkState.enqueue_queue_length (state : WorkState) (ref : w32) :
    state.queue.length ≤ (state.enqueue ref).queue.length := by
  unfold WorkState.enqueue
  split <;> simp

theorem WorkState.edge_queue_length (state : WorkState) (kind : w32 → GoString)
    (changing : w32 → Bool) (ref : w32) (branch : Option StateData) :
    state.queue.length ≤ (state.edge kind changing ref branch).queue.length := by
  cases branch with
  | none => exact Nat.le_refl _
  | some n =>
    simp only [WorkState.edge]
    split
    · exact Nat.le_refl _
    · exact WorkState.enqueue_queue_length
        { state with inputs := (state.inputs.set (queueIndex ref)
          (some (joinedInput kind changing (state.inputs[queueIndex ref]?.getD none) n))) } ref

theorem edgePrefix_queue_length (initial : WorkState) (nexts : List w32)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (decision : Bool → StateData → Option StateData) (source : StateData) (i : Nat) :
    initial.queue.length ≤ (edgePrefix initial nexts kind changing decision source i).queue.length := by
  induction i with
  | zero => exact Nat.le_refl _
  | succ i ih =>
    simp only [edgePrefix]
    cases hn : nexts[i]? with
    | none => exact ih
    | some ref => exact Nat.le_trans ih (WorkState.edge_queue_length _ kind changing ref _)

theorem SolverExecution.queue_length
    (cursor : Nat) (initial final : WorkState)
    (execution : SolverExecution s o kind changing blocks operations nexts cursor initial final) :
    initial.queue.length ≤ final.queue.length := by
  induction execution with
  | done => exact Nat.le_refl _
  | step cursor initial final current rest ih =>
    exact Nat.le_trans (edgePrefix_queue_length (initial.dequeue current.ref) (nexts current.ref)
      kind changing current.result current.targetMap (nexts current.ref).length) ih


theorem SolverStep.next_extends
    (cursor : Nat) (initial : WorkState)
    (step : SolverStep s o kind changing blocks operations nexts cursor initial)
    (hrefs : ∀ next ∈ nexts step.ref, queueIndex next < initial.inputs.length) :
    inputsExtend kind changing initial (step.next s o kind changing blocks operations nexts cursor initial) := by
  exact edgePrefix_extends (initial.dequeue step.ref) (nexts step.ref) kind changing step.result
    step.targetMap 0 (nexts step.ref).length (by omega) hrefs

theorem SolverExecution.inputs_extend
    (cursor : Nat) (initial final : WorkState)
    (execution : SolverExecution s o kind changing blocks operations nexts cursor initial final)
    (hshape : initial.inputs.length = blocks.length)
    (hrefs : ∀ ref block, blocks[queueIndex ref]? = some block →
      ∀ next ∈ nexts ref, queueIndex next < blocks.length) :
    inputsExtend kind changing initial final := by
  induction execution with
  | done => exact inputsExtend_refl kind changing _
  | step cursor initial final current rest ih =>
    have hnexts : ∀ next ∈ nexts current.ref, queueIndex next < initial.inputs.length := by
      intro next hn
      rw [hshape]
      exact hrefs current.ref current.block current.blockAt next hn
    have hnextshape := current.next_shape s o kind changing blocks operations nexts cursor initial
    exact inputsExtend_trans kind changing _ _ _
      (current.next_extends s o kind changing blocks operations nexts cursor initial hnexts)
      (ih (hnextshape.1.trans hshape))


theorem WorkState.enqueue_reachable (state : WorkState) (ref : w32)
    (h : reachableQueue state) (href : ∃ m, state.input (queueIndex ref) = some m) :
    reachableQueue (state.enqueue ref) := by
  unfold WorkState.enqueue
  split
  · exact h
  · intro key hkey
    rcases List.mem_append.mp hkey with old | new
    · exact h key old
    · have he : key = ref := by simpa using new
      subst key
      exact href

theorem WorkState.edge_reachable (state : WorkState) (kind : w32 → GoString)
    (changing : w32 → Bool) (ref : w32) (branch : Option StateData)
    (h : reachableQueue state) (hi : queueIndex ref < state.inputs.length) :
    reachableQueue (state.edge kind changing ref branch) := by
  cases branch with
  | none => exact h
  | some n =>
    simp only [WorkState.edge]
    split
    · exact h
    · let joined := joinedInput kind changing (state.input (queueIndex ref)) n
      let changed : WorkState := { state with inputs := state.inputs.set (queueIndex ref) (some joined) }
      have hslot : changed.input (queueIndex ref) = some joined := by
        simp only [changed, WorkState.input, List.getElem?_set_self hi, Option.getD_some]
      apply WorkState.enqueue_reachable changed ref
      · intro key hkey
        obtain ⟨m, hm⟩ := h key hkey
        by_cases he : queueIndex ref = queueIndex key
        · exact ⟨joined, by simpa only [he] using hslot⟩
        · exact ⟨m, by simpa only [changed, WorkState.input, List.getElem?_set_ne he] using hm⟩
      · exact ⟨joined, hslot⟩

theorem edgePrefix_reachable (initial : WorkState) (nexts : List w32)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (decision : Bool → StateData → Option StateData) (source : StateData) (i : Nat)
    (h : reachableQueue initial) (hrefs : ∀ ref ∈ nexts, queueIndex ref < initial.inputs.length) :
    reachableQueue (edgePrefix initial nexts kind changing decision source i) := by
  induction i with
  | zero => exact h
  | succ i ih =>
    simp only [edgePrefix]
    cases hn : nexts[i]? with
    | none => exact ih
    | some ref =>
      apply WorkState.edge_reachable _ kind changing ref _ ih
      rw [(edgePrefix_shape initial nexts kind changing decision source i).1]
      exact hrefs ref (List.mem_of_getElem? hn)

theorem SolverStep.next_reachable
    (cursor : Nat) (initial : WorkState)
    (step : SolverStep s o kind changing blocks operations nexts cursor initial)
    (h : reachableQueue initial) (hrefs : ∀ ref ∈ nexts step.ref, queueIndex ref < initial.inputs.length) :
    reachableQueue (step.next s o kind changing blocks operations nexts cursor initial) := by
  exact edgePrefix_reachable (initial.dequeue step.ref) (nexts step.ref) kind changing step.result
    step.targetMap (nexts step.ref).length h hrefs

theorem SolverExecution.reachable
    (cursor : Nat) (initial final : WorkState)
    (execution : SolverExecution s o kind changing blocks operations nexts cursor initial final)
    (h : reachableQueue initial) (hshape : initial.inputs.length = blocks.length)
    (hrefs : ∀ ref block, blocks[queueIndex ref]? = some block →
      ∀ next ∈ nexts ref, queueIndex next < blocks.length) : reachableQueue final := by
  induction execution with
  | done => exact h
  | step cursor initial final current rest ih =>
    have hr := current.next_reachable s o kind changing blocks operations nexts cursor initial h (by
      intro ref hn
      rw [hshape]
      exact hrefs current.ref current.block current.blockAt ref hn)
    have sizes := current.next_shape s o kind changing blocks operations nexts cursor initial
    exact ih hr (sizes.1.trans hshape)

local instance solverTerminationStateDecidable : DecidableEq StateData := Classical.typeDecidableEq _

def queueWeight (weight : w32 → Nat) (queue : List w32) : Nat := (queue.map weight).sum

theorem queueWeight_append (weight : w32 → Nat) (left right : List w32) :
    queueWeight weight (left ++ right) = queueWeight weight left + queueWeight weight right := by
  simp [queueWeight, List.map_append, List.sum_append]

theorem enqueue_weight (weight : w32 → Nat) (state : WorkState) (ref : w32) :
    queueWeight weight (state.enqueue ref).queue ≤ queueWeight weight state.queue + weight ref := by
  unfold WorkState.enqueue
  split <;> simp [queueWeight_append, queueWeight]

theorem edge_weight (weight : w32 → Nat) (state : WorkState) (kind : w32 → GoString)
    (changing : w32 → Bool) (ref : w32) (branch : Option StateData) :
    queueWeight weight (state.edge kind changing ref branch).queue ≤ queueWeight weight state.queue + weight ref := by
  cases branch with
  | none => simp only [WorkState.edge]; omega
  | some n =>
    simp only [WorkState.edge]
    split
    · omega
    · exact enqueue_weight weight
        { state with inputs := (state.inputs.set (queueIndex ref)
          (some (joinedInput kind changing (state.inputs[queueIndex ref]?.getD none) n))) } ref

theorem edgePrefix_weight (weight : w32 → Nat) (initial : WorkState) (nexts : List w32)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (decision : Bool → StateData → Option StateData) (source : StateData) (i : Nat) :
    queueWeight weight (edgePrefix initial nexts kind changing decision source i).queue ≤
      queueWeight weight initial.queue + queueWeight weight (nexts.take i) := by
  induction i with
  | zero => simp [edgePrefix, queueWeight]
  | succ i ih =>
    simp only [edgePrefix]
    rw [List.take_add_one]
    cases hn : nexts[i]? with
    | none => simpa only [hn, Option.toList_none, List.append_nil] using ih
    | some ref =>
      have he := edge_weight weight (edgePrefix initial nexts kind changing decision source i)
        kind changing ref (decision (decide (i = 0)) source)
      simp only [hn, Option.toList_some, queueWeight_append] at *
      simp only [queueWeight, List.map_cons, List.map_nil, List.sum_cons, List.sum_nil, Nat.add_zero] at *
      omega


theorem enqueue_take (state : WorkState) (ref : w32) (count : Nat)
    (hcount : count ≤ state.queue.length) :
    (state.enqueue ref).queue.take count = state.queue.take count := by
  unfold WorkState.enqueue
  split
  · rfl
  · simp [List.take_append, Nat.sub_eq_zero_of_le hcount]

theorem edge_take (state : WorkState) (kind : w32 → GoString) (changing : w32 → Bool)
    (ref : w32) (branch : Option StateData) (count : Nat) (hcount : count ≤ state.queue.length) :
    (state.edge kind changing ref branch).queue.take count = state.queue.take count := by
  cases branch with
  | none => rfl
  | some n =>
    simp only [WorkState.edge]
    split
    · rfl
    · exact enqueue_take
        { state with inputs := (state.inputs.set (queueIndex ref)
          (some (joinedInput kind changing (state.inputs[queueIndex ref]?.getD none) n))) } ref count hcount

theorem edgePrefix_take (initial : WorkState) (nexts : List w32)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (decision : Bool → StateData → Option StateData) (source : StateData) (i count : Nat)
    (hcount : count ≤ initial.queue.length) :
    (edgePrefix initial nexts kind changing decision source i).queue.take count = initial.queue.take count := by
  induction i with
  | zero => rfl
  | succ i ih =>
    simp only [edgePrefix]
    cases hn : nexts[i]? with
    | none => exact ih
    | some ref =>
      rw [edge_take _ kind changing ref _ count
        (Nat.le_trans hcount (edgePrefix_queue_length _ _ _ _ _ _ _))]
      exact ih

def pendingWeight (weight : w32 → Nat) (cursor : Nat) (state : WorkState) : Nat :=
  queueWeight weight (state.queue.drop cursor)

theorem queueWeight_split (weight : w32 → Nat) (queue : List w32) (cursor : Nat) :
    queueWeight weight queue = queueWeight weight (queue.take cursor) + queueWeight weight (queue.drop cursor) := by
  rw [← queueWeight_append, List.take_append_drop]


/-- A block costs more than all possible successor insertions. This is an
    acyclic graph ranking obligation, independent of the numeric precision. -/
theorem SolverStep.pending_decreases
    (cursor : Nat) (initial : WorkState)
    (step : SolverStep s o kind changing blocks operations nexts cursor initial)
    (weight : w32 → Nat) (budget : queueWeight weight (nexts step.ref) < weight step.ref) :
    pendingWeight weight (cursor + 1) (step.next s o kind changing blocks operations nexts cursor initial) <
      pendingWeight weight cursor initial := by
  have hhead := (List.getElem?_eq_some_iff.mp step.head).choose
  have hbudget := edgePrefix_weight weight (initial.dequeue step.ref) (nexts step.ref)
    kind changing step.result step.targetMap (nexts step.ref).length
  rw [List.take_length] at hbudget
  have htake := edgePrefix_take (initial.dequeue step.ref) (nexts step.ref)
    kind changing step.result step.targetMap (nexts step.ref).length (cursor + 1) (by
      simpa only [WorkState.dequeue] using Nat.succ_le_of_lt hhead)
  have old := queueWeight_split weight initial.queue cursor
  have newer := queueWeight_split weight
    (step.next s o kind changing blocks operations nexts cursor initial).queue (cursor + 1)
  have hprefix : queueWeight weight (initial.queue.take (cursor + 1)) =
      queueWeight weight (initial.queue.take cursor) + weight step.ref := by
    rw [List.take_add_one, step.head]
    simp [queueWeight_append, queueWeight]
  change queueWeight weight (step.next s o kind changing blocks operations nexts cursor initial).queue ≤
    queueWeight weight initial.queue + queueWeight weight (nexts step.ref) at hbudget
  change (step.next s o kind changing blocks operations nexts cursor initial).queue.take (cursor + 1) =
    initial.queue.take (cursor + 1) at htake
  rw [htake, hprefix] at newer
  unfold pendingWeight
  omega


/-- Only transitions whose operation/value and branch calls have concrete
    execution derivations are included. Constructor/source totality is separate. -/
def SolverTransition (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (blocks : List pdg.FlowBlock) (operations nexts : w32 → List w32)
    (after before : Nat × WorkState) : Prop :=
  ∃ step : SolverStep s o kind changing blocks operations nexts before.1 before.2,
    after = (before.1 + 1, step.next s o kind changing blocks operations nexts before.1 before.2)

/-- A strictly decreasing acyclic budget excludes infinite solver-step chains.
    It is not an assertion that the producer establishes this graph budget. -/
theorem solverTransition_wellFounded
    (weight : w32 → Nat) (budget : ∀ ref, queueWeight weight (nexts ref) < weight ref) :
    WellFounded (SolverTransition s o kind changing blocks operations nexts) := by
  have accessible : ∀ n, ∀ pair : Nat × WorkState,
      pendingWeight weight pair.1 pair.2 = n →
      Acc (SolverTransition s o kind changing blocks operations nexts) pair := by
    intro n
    induction n using Nat.strongRecOn with
    | ind n ih =>
      intro pair hn
      apply Acc.intro pair
      intro previous hprevious
      obtain ⟨step, he⟩ := hprevious
      subst previous
      have decreases := step.pending_decreases s o kind changing blocks operations nexts
        pair.1 pair.2 weight (budget step.ref)
      exact ih (pendingWeight weight (pair.1 + 1)
        (step.next s o kind changing blocks operations nexts pair.1 pair.2))
        (by simpa only [hn] using decreases) _ rfl
  exact ⟨fun pair => accessible _ pair rfl⟩


/-- Data formation is explicit: scalar write sites/value derivations and branch
    derivations must exist for each pending block. No child WP/soundness is assumed. -/
def SolverDataComplete (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (blocks : List pdg.FlowBlock) (operations nexts : w32 → List w32) : Prop :=
  ∀ cursor state, reachableQueue state → scalarWorkInvariant s kind changing cursor state →
    state.flags.length = state.inputs.length → state.flags.length = blocks.length →
    cursor < state.queue.length →
    Nonempty (SolverStep s o kind changing blocks operations nexts cursor state)

/-- Well-founded recursion supplies a finite execution derivation. Source
    construction still has to establish data formation and the acyclic budget. -/
theorem solverExecution_exists
    (weight : w32 → Nat) (budget : ∀ ref, queueWeight weight (nexts ref) < weight ref)
    (complete : SolverDataComplete s o kind changing blocks operations nexts)
    (agreement : metadataAgreement s kind changing)
    (hk : ∀ key, kind key ≠ go!"\"pointer\"")
    (hrefs : ∀ ref block, blocks[queueIndex ref]? = some block →
      ∀ next ∈ nexts ref, 0 < uint.Z next ∧ uint.Z next ≤ blocks.length)
    (cursor : Nat) (initial : WorkState)
    (hinvariant : scalarWorkInvariant s kind changing cursor initial)
    (hshape : initial.flags.length = initial.inputs.length)
    (hblocks : initial.flags.length = blocks.length) (hreachable : reachableQueue initial) :
    ∃ final, SolverExecution s o kind changing blocks operations nexts cursor initial final := by
  have wf := solverTransition_wellFounded s o kind changing blocks operations nexts weight budget
  have total : ∀ pair : Nat × WorkState,
      reachableQueue pair.2 → scalarWorkInvariant s kind changing pair.1 pair.2 →
      pair.2.flags.length = pair.2.inputs.length → pair.2.flags.length = blocks.length →
      ∃ final, SolverExecution s o kind changing blocks operations nexts pair.1 pair.2 final := by
    intro pair
    induction pair using wf.induction with
    | h pair ih =>
      obtain ⟨cursor, state⟩ := pair
      intro reachable inv shape count
      by_cases done : cursor = state.queue.length
      · exact ⟨state, .done cursor state done⟩
      · have pending : cursor < state.queue.length := by
          have hpos : cursor ≤ state.queue.length := inv.1.position
          omega
        obtain ⟨step⟩ := complete cursor state reachable inv shape count pending
        have nextinv := step.next_invariant s o kind changing blocks operations nexts cursor state
          inv agreement hk (by
            intro next hn
            simpa only [queueRefValid, count] using hrefs step.ref step.block step.blockAt next hn)
        have nextreachable := step.next_reachable s o kind changing blocks operations nexts cursor state
          reachable (by
            intro next hn
            have hvalid := hrefs step.ref step.block step.blockAt next hn
            have size : state.inputs.length = blocks.length := shape.symm.trans count
            unfold queueIndex
            omega)
        have sizes := step.next_shape s o kind changing blocks operations nexts cursor state
        obtain ⟨final, rest⟩ := ih
          (cursor + 1, step.next s o kind changing blocks operations nexts cursor state)
          ⟨step, rfl⟩ nextreachable nextinv ((sizes.2.trans shape).trans sizes.1.symm) (sizes.2.trans count)
        exact ⟨final, .step cursor state final step rest⟩
  exact total (cursor, initial) hreachable hinvariant hshape hblocks


/-- A positive formation control: the completeness premise is satisfiable for
    a terminal zero-operation block, for every admissible input map and queue.
    This is not a substitute for source-construction correctness. -/
theorem solverDataComplete_terminal (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool) (block : pdg.FlowBlock)
    (hcondition : block.Condition' = W32 0) :
    SolverDataComplete s o kind changing [block] (fun _ => []) (fun _ => []) := by
  intro cursor state reachable inv shape count pending
  list_elem state.queue cursor as ref
  have hd : state.queue.drop cursor = ref :: state.queue.drop (cursor + 1) := by
    rw [List.drop_eq_getElem_cons pending, (List.getElem?_eq_some_iff.mp Href_lookup).choose_spec]
  have hr := inv.1.references ref (by rw [hd]; simp)
  have size : state.flags.length = 1 := count
  have hzero : queueIndex ref = 0 := by
    unfold queueRefValid at hr
    unfold queueIndex
    omega
  obtain ⟨m, hm⟩ := reachable ref (List.mem_of_getElem? Href_lookup)
  refine ⟨⟨ref, block, m, m, (fun _ n => some n), Href_lookup, ?_, hm, .nil m, ?_⟩⟩
  · simp only [hzero, List.getElem?_cons_zero]
  · simpa only [hcondition] using (DecisionEvidence.zero (s := s) (o := o) (kind := kind) (changing := changing))

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
