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

theorem WorkState.dequeue_invariant (s : QuerySnapshot) (kind : w32 → GoString)
    (changing : w32 → Bool) (cursor : Nat) (state : WorkState) (ref : w32)
    (h : scalarWorkInvariant s kind changing cursor state)
    (head : state.queue[cursor]? = some ref) :
    scalarWorkInvariant s kind changing (cursor + 1) (state.dequeue ref) := by
  exact ⟨h.1.advance state.flags cursor state.queue ref head, h.2.1, h.2.2⟩

theorem SolverStep.reference (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (blocks : List pdg.FlowBlock) (operations nexts : w32 → List w32)
    (cursor : Nat) (initial : WorkState)
    (step : SolverStep s o kind changing blocks operations nexts cursor initial)
    (hq : QueueInvariant initial.flags cursor initial.queue) : queueRefValid initial.flags step.ref := by
  have hc : cursor < initial.queue.length := (List.getElem?_eq_some_iff.mp step.head).choose
  apply hq.references
  rw [List.drop_eq_getElem_cons hc, (List.getElem?_eq_some_iff.mp step.head).choose_spec]
  simp

theorem SolverStep.input_valid (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (blocks : List pdg.FlowBlock) (operations nexts : w32 → List w32)
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

theorem SolverStep.input_scalar (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (blocks : List pdg.FlowBlock) (operations nexts : w32 → List w32)
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

theorem SolverStep.next_shape (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (blocks : List pdg.FlowBlock) (operations nexts : w32 → List w32)
    (cursor : Nat) (initial : WorkState)
    (step : SolverStep s o kind changing blocks operations nexts cursor initial) :
    (step.next s o kind changing blocks operations nexts cursor initial).inputs.length = initial.inputs.length ∧
    (step.next s o kind changing blocks operations nexts cursor initial).flags.length = initial.flags.length := by
  simpa only [SolverStep.next, WorkState.dequeue, List.length_set] using
    edgePrefix_shape (initial.dequeue step.ref) (nexts step.ref) kind changing step.result step.targetMap
      (nexts step.ref).length

theorem SolverStep.next_invariant (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (blocks : List pdg.FlowBlock) (operations nexts : w32 → List w32)
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

theorem SolverExecution.queue_length (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (blocks : List pdg.FlowBlock) (operations nexts : w32 → List w32)
    (cursor : Nat) (initial final : WorkState)
    (execution : SolverExecution s o kind changing blocks operations nexts cursor initial final) :
    initial.queue.length ≤ final.queue.length := by
  induction execution with
  | done => exact Nat.le_refl _
  | step cursor initial final current rest ih =>
    exact Nat.le_trans (edgePrefix_queue_length (initial.dequeue current.ref) (nexts current.ref)
      kind changing current.result current.targetMap (nexts current.ref).length) ih


theorem SolverStep.next_extends (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (blocks : List pdg.FlowBlock) (operations nexts : w32 → List w32)
    (cursor : Nat) (initial : WorkState)
    (step : SolverStep s o kind changing blocks operations nexts cursor initial)
    (hrefs : ∀ next ∈ nexts step.ref, queueIndex next < initial.inputs.length) :
    inputsExtend kind changing initial (step.next s o kind changing blocks operations nexts cursor initial) := by
  exact edgePrefix_extends (initial.dequeue step.ref) (nexts step.ref) kind changing step.result
    step.targetMap 0 (nexts step.ref).length (by omega) hrefs

theorem SolverExecution.inputs_extend (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (blocks : List pdg.FlowBlock) (operations nexts : w32 → List w32)
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

theorem SolverStep.next_reachable (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (blocks : List pdg.FlowBlock) (operations nexts : w32 → List w32)
    (cursor : Nat) (initial : WorkState)
    (step : SolverStep s o kind changing blocks operations nexts cursor initial)
    (h : reachableQueue initial) (hrefs : ∀ ref ∈ nexts step.ref, queueIndex ref < initial.inputs.length) :
    reachableQueue (step.next s o kind changing blocks operations nexts cursor initial) := by
  exact edgePrefix_reachable (initial.dequeue step.ref) (nexts step.ref) kind changing step.result
    step.targetMap (nexts step.ref).length h hrefs

theorem SolverExecution.reachable (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (blocks : List pdg.FlowBlock) (operations nexts : w32 → List w32)
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

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
