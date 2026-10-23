module
public import Gstyle.Proof.Worklist.SolveModel
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
theorem SolverStep.pending_decreases (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (blocks : List pdg.FlowBlock) (operations nexts : w32 → List w32)
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
theorem solverTransition_wellFounded (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (blocks : List pdg.FlowBlock) (operations nexts : w32 → List w32)
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
theorem solverExecution_exists (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (blocks : List pdg.FlowBlock) (operations nexts : w32 → List w32)
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
