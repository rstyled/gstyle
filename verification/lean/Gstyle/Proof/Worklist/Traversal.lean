module
public import Gstyle.Proof.Worklist.Decision
public import Gstyle.Proof.Worklist.FlowLayout
public import Gstyle.Proof.Worklist.Layout
public import Gstyle.Proof.Worklist.StepModel
public import Gstyle.Proof.StateMap.Equality
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
local instance workTraversalDataDecidable : DecidableEq StateData := Classical.typeDecidableEq _

def workStateOwn (receiver view : Loc) (inputs flags queue : GoSlice)
    (pointers : List Loc) (model : WorkState) : IProp GF := iprop(
  receiver ↦ ({ view' := view, inputs' := inputs, queued' := flags, queue' := queue } : analysis) ∗
  inputs ↦* pointers ∗ inputStatesOwn pointers model.inputs ∗ flags ↦* model.flags ∗
  queue ↦* model.queue ∗ ownSliceCap w32 queue (DFrac.own 1))

/-- Traverse the real successor array in order, refining copies and joining changed inputs. -/
theorem wp_successors (receiver view state : Loc) (s : QuerySnapshot)
    (flow : pdg.NormalFlow) (blocks : List pdg.FlowBlock) (block : pdg.FlowBlock)
    (nexts : List w32) (ref : w32) (m : StateData)
    (inputs flags queue : GoSlice) (pointers : List Loc) (initial : WorkState)
    (o : AssumeOperators) (kind : w32 → GoString) (changing : w32 → Bool)
    (result : Bool → StateData → Option StateData)
    (decision : DecisionEvidence s o kind changing block.Condition' result)
    (wf : ValueSnapshotValid s) (hparse : parseInt64Contract (GF := GF))
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ blocks.length)
    (hblock : blocks[(uint.Z ref - 1).toNat]? = some block)
    (hnexts : ∀ next ∈ nexts, queueRefValid initial.flags next)
    (hshape : initial.flags.length = initial.inputs.length)
    (hmodels : ∀ p ∈ initial.inputs, validBranchOption s.kinds s.unstable kind changing p)
    (hm : validBranchState s.kinds s.unstable kind changing m) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗
       isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
       (querySnapshotOwn view s ∗ s.graph.ValueFlow' ↦□ flow ∗ flow.Blocks' ↦*□ blocks ∗
        block.Successors' ↦*□ nexts ∗ assumeOperatorsOwn o ∗
        workStateOwn receiver view inputs flags queue pointers initial ∗ state ↦$ m) }}
      (App (App (Val #(methods (go.PointerType analysis.ty) go!"successors" #receiver))
        (Val #ref)) (Val #state))
    {{ RET #(); ∃ (out : GoSlice) (locations : List Loc),
        workStateOwn receiver view inputs flags out locations
          (edgePrefix initial nexts kind changing result m nexts.length) ∗ state ↦$ m }} := by
  wp_start as ⟨#Hsnapshot, #Hflow, #Hblocks, #Hnexts, #Hoperators, Hwork, Hstate⟩
  iunfold workStateOwn at Hwork
  icases Hwork with ⟨Ha, Hinputs, Hmaps, Hflags, Hqueue, Hcap⟩
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, Hgraph, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _⟩
  ihave %Hblen := ownSlice_len $$ Hblocks
  ihave %Hnlen := ownSlice_len $$ Hnexts
  ihave %Hplen := inputStatesOwn_length pointers initial.inputs $$ Hmaps
  ihave %Hilen := ownSlice_len $$ Hinputs
  try wp_auto
  have Hi : sint.Z (W64 (uint.Z (ref - W32 1))) = uint.Z ref - 1 := by
    obtain ⟨_, _⟩ := href
    word
  rw [Hi]
  rw [ite_eq_left (by obtain ⟨_, _⟩ := href; constructor <;> word :
    0 ≤ uint.Z ref - 1 ∧ uint.Z ref - 1 < sint.Z flow.Blocks'.len)]
  wp_apply wp_load_slice_index flow.Blocks' (uint.Z ref - 1) blocks DFrac.discard block (by omega)
      $$ [Hblocks] with Hblocks
  · iframe Hblocks; ipureintro; exact hblock
  try wp_auto
  ihave Hwork : (workStateOwn receiver view inputs flags queue pointers initial : IProp GF)
      $$ [Ha Hinputs Hmaps Hflags Hqueue Hcap]
  · unfold workStateOwn
    iframe Ha Hinputs Hmaps Hflags Hqueue Hcap
  ihave HI : (∃ (iv ov : w64) (nv : w32) (q : GoSlice) (locations : List Loc),
      "i" ∷ i_ptr ↦ iv ∗ "outcome" ∷ outcome_ptr ↦ ov ∗ "next" ∷ next_ptr ↦ nv ∗
      "a" ∷ a_ptr ↦ receiver ∗ "state" ∷ state_ptr ↦ state ∗ "block" ∷ block_ptr ↦ block ∗
      "Hwork" ∷ workStateOwn receiver view inputs flags q locations
        (edgePrefix initial nexts kind changing result m (sint.nat iv)) ∗
      "Hstate" ∷ state ↦$ m ∗
      "%Hbounds" ∷ ⌜0 ≤ sint.Z iv ∧ sint.Z iv ≤ sint.Z block.Successors'.len⌝ ∗
      "%Hvalid" ∷ ⌜validWorkInputs s kind changing
        (edgePrefix initial nexts kind changing result m (sint.nat iv))⌝ : IProp GF)
      $$ [i outcome next a state block Hwork Hstate]
  · iexists (W64 0), (zero_val w64), (zero_val w32), queue, pointers
    simp only [show sint.nat (W64 0) = 0 from rfl, edgePrefix]
    iframe Hwork Hstate i outcome next a state block
    ipureintro
    exact ⟨by constructor <;> word, hmodels⟩
  wp_for HI
  simp only [go.intoVal_inj.eq_iff, decide_eq_true_eq, Bool.false_eq_true, ↓reduceIte]
  by_cases Hif : sint.Z iv < sint.Z block.Successors'.len
  · simp only [Hif, ite_true, decide_true]
    have hi : sint.nat iv < nexts.length := by word
    list_elem nexts (sint.nat iv) as target
    try wp_auto
    rw [ite_eq_left (by constructor <;> word :
      0 ≤ sint.Z iv ∧ sint.Z iv < sint.Z block.Successors'.len)]
    wp_apply wp_load_slice_index block.Successors' (sint.Z iv) nexts DFrac.discard target (by word)
        $$ [Hnexts] with Hnexts
    · iframe Hnexts; ipureintro; exact Htarget_lookup
    iunfold workStateOwn at Hwork
    icases Hwork with ⟨Ha, Hinputs, Hmaps, Hflags, Hqueue, Hcap⟩
    try wp_auto
    wp_apply wp_copyState state m $$ [Hstate] with %branch ⟨Hstate, Hbranch⟩
    · iframe Hstate
    try wp_auto
    have heq : iv = W64 0 ↔ sint.nat iv = 0 := by word
    simp only [heq]
    wp_apply wp_decision view s o kind changing block.Condition' result decision wf hparse
        (decide (sint.nat iv = 0)) branch m hm $$ [Hsnapshot Hoperators Hbranch]
        with %out ⟨Hsnapshot, Hoperators, Hbranch, %hbranchvalid⟩
    · iframe Hsnapshot Hoperators Hbranch
    cases hb : result (decide (sint.nat iv = 0)) m with
    | none =>
      iunfold ownsBranch at Hbranch
      icases Hbranch with %hbranch
      subst out
      try wp_auto
      ihave Hwork : (workStateOwn receiver view inputs flags q locations
          (edgePrefix initial nexts kind changing result m (sint.nat iv)) : IProp GF)
          $$ [Ha Hinputs Hmaps Hflags Hqueue Hcap]
      · unfold workStateOwn
        iframe Ha Hinputs Hmaps Hflags Hqueue Hcap
      wp_for_post
      iframe
      iexists (iv + W64 1), iv, target, q, locations
      have hn : sint.nat (iv + W64 1) = sint.nat iv + 1 := by word
      rw [hn, edgePrefix_step initial nexts kind changing result m _ target Htarget_lookup]
      rw [hb]
      simp only [WorkState.edge]
      iframe
      ipureintro
      exact ⟨by constructor <;> word, Hvalid⟩
    | some n =>
      have Hnvalid : validBranchState s.kinds s.unstable kind changing n := by
        simpa only [hb, validBranchOption] using hbranchvalid
      iunfold ownsBranch at Hbranch
      ihave %Hnon := ownMap_not_nil out n (DFrac.own 1) $$ Hbranch
      try wp_auto
      rw [decide_eq_false Hnon]
      try wp_auto
      let current := edgePrefix initial nexts kind changing result m (sint.nat iv)
      have Htargetref := hnexts target (List.mem_of_getElem? Htarget_lookup)
      have Hpositive : 0 < uint.Z target := Htargetref.1
      have Hmodelshape := edgePrefix_shape initial nexts kind changing result m (sint.nat iv)
      ihave %Hlocations := inputStatesOwn_length locations current.inputs $$ Hmaps
      ihave %Hinputlen := ownSlice_len $$ Hinputs
      have Htargetindex : queueIndex target < current.inputs.length := by
        have h := Hmodelshape.1
        unfold queueIndex
        unfold queueRefValid at Htargetref
        dsimp only [current] at *
        omega
      have Hpointerindex : queueIndex target < locations.length := by omega
      list_elem locations (queueIndex target) as previous
      list_elem current.inputs (queueIndex target) as previousModel
      have Hslotvalid := Hvalid previousModel (List.mem_of_getElem? HpreviousModel_lookup)
      ihave ⟨Hprevious, Hrestore⟩ := inputStatesOwn_acc locations current.inputs
        (queueIndex target) previous previousModel Hprevious_lookup HpreviousModel_lookup $$ Hmaps
      have Hindex : sint.Z (W64 (uint.Z (target - W32 1))) = uint.Z target - 1 := by
        unfold queueRefValid at Htargetref
        obtain ⟨_, _⟩ := Htargetref
        word
      rw [Hindex]
      rw [ite_eq_left (by unfold queueIndex at Hpointerindex; constructor <;> word :
        0 ≤ uint.Z target - 1 ∧ uint.Z target - 1 < sint.Z inputs.len)]
      wp_apply wp_load_slice_index inputs (uint.Z target - 1) locations (DFrac.own 1) previous (by
          unfold queueIndex at Hpointerindex; omega) $$ [Hinputs] with Hinputs
      · iframe Hinputs; ipureintro; exact Hprevious_lookup
      wp_apply wp_join_input view s kind changing previous out previousModel n wf.unstable Hslotvalid Hnvalid
          $$ [Hsnapshot Hprevious Hbranch] with %joined ⟨Hsnapshot, Hprevious, Hbranch, Hjoined⟩
      · iframe Hsnapshot Hprevious Hbranch
      try wp_auto
      rw [Hindex]
      rw [ite_eq_left (by unfold queueIndex at Hpointerindex; constructor <;> word :
        0 ≤ uint.Z target - 1 ∧ uint.Z target - 1 < sint.Z inputs.len)]
      wp_apply wp_load_slice_index inputs (uint.Z target - 1) locations (DFrac.own 1) previous (by omega)
          $$ [Hinputs] with Hinputs
      · iframe Hinputs; ipureintro; exact Hprevious_lookup
      wp_apply wp_equal_input previous joined previousModel (joinedInput kind changing previousModel n)
          $$ [Hprevious Hjoined] with ⟨Hprevious, Hjoined, %Hnil⟩
      · iframe Hprevious Hjoined
      try wp_auto
      have HpreviousValue : current.inputs[queueIndex target]?.getD none = previousModel := by
        rw [HpreviousModel_lookup]; rfl
      by_cases Hsame : previousModel = some (joinedInput kind changing previousModel n)
      · rw [inputEqual_of_same previousModel (joinedInput kind changing previousModel n) Hsame]
        try wp_auto
        rw [Hindex]
        rw [ite_eq_left (by unfold queueIndex at Hpointerindex; constructor <;> word :
          0 ≤ uint.Z target - 1 ∧ uint.Z target - 1 < sint.Z inputs.len)]
        wp_apply wp_load_slice_index inputs (uint.Z target - 1) locations (DFrac.own 1) previous (by omega)
            $$ [Hinputs] with Hinputs
        · iframe Hinputs; ipureintro; exact Hprevious_lookup
        have HpreviousNon : previous ≠ map.nil := by
          intro he
          have hn := Hnil.mp he
          rw [hn] at Hsame
          cases Hsame
        try wp_auto
        rw [decide_eq_false HpreviousNon]
        try wp_auto
        ihave Hmaps := Hrestore $$ %previous %previousModel Hprevious
        rw [list_set_lookup_self locations (queueIndex target) previous Hprevious_lookup,
            list_set_lookup_self current.inputs (queueIndex target) previousModel HpreviousModel_lookup]
        ihave Hwork : (workStateOwn receiver view inputs flags q locations current : IProp GF)
            $$ [Ha Hinputs Hmaps Hflags Hqueue Hcap]
        · unfold workStateOwn
          iframe Ha Hinputs Hmaps Hflags Hqueue Hcap
        have Hstable : current.edge kind changing target (some n) = current := by
          simp only [WorkState.edge, HpreviousValue]
          rw [if_pos Hsame]
        wp_for_post
        iframe
        iexists (iv + W64 1), iv, target, q, locations
        have hn : sint.nat (iv + W64 1) = sint.nat iv + 1 := by word
        rw [hn, edgePrefix_step initial nexts kind changing result m _ target Htarget_lookup, hb, Hstable]
        iframe
        ipureintro
        exact ⟨by constructor <;> word, Hvalid⟩
      · cases Hequal : inputEqual previousModel (joinedInput kind changing previousModel n)
        all_goals try wp_auto
        all_goals try
          have Hnone := inputEqual_changed_nil previousModel (joinedInput kind changing previousModel n) Hequal Hsame
          have HpreviousNil := Hnil.mpr Hnone
          rw [Hindex]
          rw [ite_eq_left (by unfold queueIndex at Hpointerindex; constructor <;> word :
            0 ≤ uint.Z target - 1 ∧ uint.Z target - 1 < sint.Z inputs.len)]
          wp_apply wp_load_slice_index inputs (uint.Z target - 1) locations (DFrac.own 1) previous (by omega)
              $$ [Hinputs] with Hinputs
          · iframe Hinputs; ipureintro; exact Hprevious_lookup
          try wp_auto
          rw [decide_eq_true HpreviousNil]
          try wp_auto
        all_goals
          rw [Hindex]
          rw [ite_eq_left (by unfold queueIndex at Hpointerindex; constructor <;> word :
            0 ≤ uint.Z target - 1 ∧ uint.Z target - 1 < sint.Z inputs.len)]
          wp_apply wp_store_slice_index inputs (uint.Z target - 1) locations joined $$ [Hinputs] with Hinputs
          · iframe Hinputs; ipureintro; unfold queueIndex at Hpointerindex; constructor <;> omega
          ihave Howned : (ownsBranch (some (joinedInput kind changing previousModel n)) joined : IProp GF) $$ [Hjoined]
          · unfold ownsBranch
            iexact Hjoined
          ihave Hmaps := Hrestore $$ %joined %(some (joinedInput kind changing previousModel n)) Howned
          have Hflagref : queueRefValid current.flags target := by
            unfold queueRefValid
            rw [show current.flags.length = initial.flags.length from Hmodelshape.2]
            exact Htargetref
          have Hflagindex := queueIndex_lt current.flags target Hflagref
          list_elem current.flags (queueIndex target) as present
          try wp_auto
          wp_apply wp_enqueue receiver
              ({ view' := view, inputs' := inputs, queued' := flags, queue' := q } : analysis)
              current.flags current.queue target present Hflagref Hpresent_lookup
              $$ [Ha Hflags Hqueue Hcap] with ⟨%newQueue, Ha, Hflags, Hqueue, Hcap⟩
          · iframe Ha Hflags Hqueue Hcap
          have Hupdated : current.edge kind changing target (some n) =
              { inputs := current.inputs.set (queueIndex target) (some (joinedInput kind changing previousModel n)),
                flags := if present then current.flags else current.flags.set (queueIndex target) true,
                queue := if present then current.queue else current.queue ++ [target] } := by
            simp only [WorkState.edge, HpreviousValue, if_neg Hsame, WorkState.enqueue, Hpresent_lookup, Option.getD_some]
            cases present <;> rfl
          ihave Hwork : (workStateOwn receiver view inputs flags newQueue (locations.set (queueIndex target) joined)
              (current.edge kind changing target (some n)) : IProp GF)
              $$ [Ha Hinputs Hmaps Hflags Hqueue Hcap]
          · rw [Hupdated]
            unfold workStateOwn queueIndex
            iframe Ha Hinputs Hmaps Hflags Hqueue Hcap
          have HnewValid := WorkState.edge_valid s current kind changing target (some n) Hvalid Hnvalid
          try wp_auto
          wp_for_post
          iframe
          iexists (iv + W64 1), iv, target, newQueue, (locations.set (queueIndex target) joined)
          have hn : sint.nat (iv + W64 1) = sint.nat iv + 1 := by word
          rw [hn, edgePrefix_step initial nexts kind changing result m _ target Htarget_lookup, hb]
          iframe
          ipureintro
          exact ⟨by constructor <;> word, HnewValid⟩
  · simp only [Hif, ite_false, decide_false]
    have hi : sint.nat iv = nexts.length := by word
    rw [hi]
    wp_end
    all_goals iexists q, locations
    all_goals iframe Hwork Hstate

theorem wp_successors_invariant (receiver view state : Loc) (s : QuerySnapshot)
    (flow : pdg.NormalFlow) (blocks : List pdg.FlowBlock) (block : pdg.FlowBlock)
    (nexts : List w32) (ref : w32) (m : StateData)
    (inputs flags queue : GoSlice) (pointers : List Loc) (initial : WorkState)
    (o : AssumeOperators) (kind : w32 → GoString) (changing : w32 → Bool)
    (result : Bool → StateData → Option StateData)
    (decision : DecisionEvidence s o kind changing block.Condition' result)
    (wf : ValueSnapshotValid s) (hparse : parseInt64Contract (GF := GF))
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ blocks.length)
    (hblock : blocks[(uint.Z ref - 1).toNat]? = some block)
    (hnexts : ∀ next ∈ nexts, queueRefValid initial.flags next)
    (hshape : initial.flags.length = initial.inputs.length)
    (hmodels : ∀ p ∈ initial.inputs, validBranchOption s.kinds s.unstable kind changing p)
    (hm : validBranchState s.kinds s.unstable kind changing m)
    (cursor : Nat) (hqueue : QueueInvariant initial.flags cursor initial.queue) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗
       isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
       (querySnapshotOwn view s ∗ s.graph.ValueFlow' ↦□ flow ∗ flow.Blocks' ↦*□ blocks ∗
        block.Successors' ↦*□ nexts ∗ assumeOperatorsOwn o ∗
        workStateOwn receiver view inputs flags queue pointers initial ∗ state ↦$ m) }}
      (App (App (Val #(methods (go.PointerType analysis.ty) go!"successors" #receiver))
        (Val #ref)) (Val #state))
    {{ RET #(); ∃ (out : GoSlice) (locations : List Loc),
        workStateOwn receiver view inputs flags out locations
          (edgePrefix initial nexts kind changing result m nexts.length) ∗ state ↦$ m ∗
        ⌜QueueInvariant (edgePrefix initial nexts kind changing result m nexts.length).flags cursor
            (edgePrefix initial nexts kind changing result m nexts.length).queue ∧
          validWorkInputs s kind changing (edgePrefix initial nexts kind changing result m nexts.length)⌝ }} := by
  have hq := edgePrefix_queue_invariant initial nexts kind changing result m nexts.length cursor hqueue hnexts
  have hr : ∀ truth, validBranchOption s.kinds s.unstable kind changing (result truth m) :=
    fun truth => DecisionEvidence.valid s o kind changing block.Condition' result decision truth m hm
  have hv := edgePrefix_valid s initial nexts kind changing result m nexts.length hmodels hr
  wp_start_folded as Hinputs
  wp_apply wp_successors receiver view state s flow blocks block nexts ref m inputs flags queue pointers initial
      o kind changing result decision wf hparse href hblock hnexts hshape hmodels hm
      $$ [Hinputs] with ⟨%out, %locations, Hwork, Hstate⟩
  · iexact Hinputs
  iapply HΦ
  iexists out, locations
  iframe Hwork Hstate
  ipureintro
  exact ⟨hq, hv⟩

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
