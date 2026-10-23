module
public import Gstyle.Proof.Worklist.SolveModel
public import Gstyle.Proof.Worklist.SolveSyntax
@[expose] public section
noncomputable section
set_option maxHeartbeats 0
set_option maxRecDepth 100000
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
variable [FuncUnfold fieldIdentity [] fieldIdentity.impl]
variable [FuncUnfold singleValueWrite [] singleValueWrite.impl]
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




/-- Read-only ownership for the operation and successor slices of every block. -/
def solverRowsOwn (blocks : List pdg.FlowBlock) (operations nexts : w32 → List w32) : IProp GF :=
  iprop(□ ∀ (ref : w32) (block : pdg.FlowBlock), ⌜blocks[queueIndex ref]? = some block⌝ -∗
    block.Operations' ↦*□ operations ref ∗ block.Successors' ↦*□ nexts ref)

instance solverRowsOwn_persistent (blocks : List pdg.FlowBlock) (operations nexts : w32 → List w32) :
    Persistent (solverRowsOwn (GF := GF) blocks operations nexts) := by
  unfold solverRowsOwn
  infer_instance

/-- Execute the actual selected solve cursor tail, including cursor initialization.
    This is the post-constructor loop boundary, not a theorem about newGraphView. -/
theorem wp_solve_cursor (receiver view graphLocal cursor : Loc) (s : QuerySnapshot)
    (flow : pdg.NormalFlow) (blocks : List pdg.FlowBlock) (operations nexts : w32 → List w32)
    (inputs flags queue : GoSlice) (pointers : List Loc) (initial final : WorkState)
    (o : AssumeOperators) (kind : w32 → GoString) (changing : w32 → Bool)
    (execution : SolverExecution s o kind changing blocks operations nexts 0 initial final)
    (hinvariant : scalarWorkInvariant s kind changing 0 initial)
    (hshape : initial.flags.length = initial.inputs.length)
    (hblocks : initial.flags.length = blocks.length)
    (agreement : metadataAgreement s kind changing)
    (hk : ∀ key, kind key ≠ go!"\"pointer\"")
    (hrefs : ∀ ref block, blocks[queueIndex ref]? = some block →
      ∀ next ∈ nexts ref, 0 < uint.Z next ∧ uint.Z next ≤ blocks.length)
    (hbound : final.queue.length < 2^63 - 1)
    (wf : ValueSnapshotValid s) (hparse : parseInt64Contract (GF := GF))
    (huparse : parseEmptyUint32Contract (GF := GF)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗
       isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
       isPkgInit (PROP := IProp GF) pkg_id.strconv ∗
       (querySnapshotOwn view s ∗ s.graph.ValueFlow' ↦□ flow ∗ flow.Blocks' ↦*□ blocks ∗
        solverRowsOwn blocks operations nexts ∗ assumeOperatorsOwn o ∗
        graphLocal ↦□ s.view.graph' ∗ cursor ↦ (W64 0) ∗
        localRootsOwn (GF := GF) s.view.fieldRoots' ∅ ∗
        workStateOwn receiver view inputs flags queue pointers initial) }}
      (solveCursorRun receiver graphLocal cursor)
    {{ RET executeVal; ∃ (out : GoSlice) (locations : List Loc),
        workStateOwn receiver view inputs flags out locations final ∗
        localRootsOwn (GF := GF) s.view.fieldRoots' ∅ ∗
        cursor ↦ (W64 final.queue.length) ∗
        ⌜scalarWorkInvariant s kind changing final.queue.length final ∧ inputsExtend kind changing initial final⌝ }} := by
  have Hextends := SolverExecution.inputs_extend s o kind changing blocks operations nexts 0 initial final
    execution (hshape.symm.trans hblocks) (by
      intro ref block hblock next hn
      have hp := hrefs ref block hblock next hn
      unfold queueIndex
      omega)
  wp_start_folded as ⟨#Hsnapshot, #Hflow, #Hblocks, #Hrows, #Hoperators, #Hg, Hcursor, Hroots, Hwork⟩
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, Hgraph, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _⟩
  ihave %Hblen := ownSlice_len $$ Hblocks
  unfold solveCursorRun solveCursorInitializer solveCursorBody solveBody solve.impl
  dsimp only [cursorSite, Option.or, Option.getD]
  try wp_auto
  ihave HI : (∃ (cv : w64) (q : GoSlice) (locations : List Loc) (current : WorkState),
      "Hcursor" ∷ cursor ↦ cv ∗
      "Hwork" ∷ workStateOwn receiver view inputs flags q locations current ∗
      "Hroots" ∷ localRootsOwn (GF := GF) s.view.fieldRoots' ∅ ∗
      "%Hbounds" ∷ ⌜0 ≤ sint.Z cv⌝ ∗
      "%Hinv" ∷ ⌜scalarWorkInvariant s kind changing (sint.nat cv) current⌝ ∗
      "%Hshape" ∷ ⌜current.flags.length = current.inputs.length ∧ current.flags.length = blocks.length⌝ ∗
      "%Hremaining" ∷ ⌜SolverExecution s o kind changing blocks operations nexts (sint.nat cv) current final⌝ : IProp GF)
      $$ [Hcursor Hwork Hroots]
  · iexists (W64 0), queue, pointers, initial
    iframe Hcursor Hwork Hroots
    ipureintro
    exact ⟨by decide, hinvariant, ⟨hshape, hblocks⟩, execution⟩
  wp_for HI
  iunfold workStateOwn at Hwork
  icases Hwork with ⟨Ha, Hinputs, Hmaps, Hflags, Hqueue, Hcap⟩
  ihave %Hqlen := ownSlice_len $$ Hqueue
  ihave %Hflen := ownSlice_len $$ Hflags
  ihave %Hilen := ownSlice_len $$ Hinputs
  ihave %Hplen := inputStatesOwn_length locations current.inputs $$ Hmaps
  try wp_auto
  simp only [go.intoVal_inj.eq_iff, decide_eq_true_eq, Bool.false_eq_true, ↓reduceIte]
  generalize Hindex : sint.nat cv = index at Hinv Hremaining
  by_cases Hif : sint.Z cv < sint.Z q.len
  · simp only [Hif, ite_true, _root_.decide_true]
    cases Hremaining with
    | done here state exhausted =>
      have hlen := exhausted
      have hpos := Hinv.1.position
      word
    | step here state final currentStep rest =>
      have Href := currentStep.reference s o kind changing blocks operations nexts index current Hinv.1
      have Hrefblocks : 0 < uint.Z currentStep.ref ∧ uint.Z currentStep.ref ≤ blocks.length := by
        simpa only [queueRefValid, Hshape.2] using Href
      have Hidx : queueIndex currentStep.ref < current.inputs.length := by
        rw [← Hshape.1]
        exact queueIndex_lt current.flags currentStep.ref Href
      have Hptridx : queueIndex currentStep.ref < locations.length := by omega
      have Hblock := currentStep.blockAt
      have Hsource := currentStep.input_valid s o kind changing blocks operations nexts
        index current Hinv.2.1
      have Hscalar := currentStep.input_scalar s o kind changing blocks operations nexts
        index current Hinv.2.2
      ihave #Hselected := Hrows
      iunfold solverRowsOwn at Hselected
      ihave ⟨#Hops, #Hnexts⟩ := Hselected $$ %currentStep.ref %currentStep.block %Hblock
      try wp_auto
      rw [ite_eq_left (by constructor <;> word : 0 ≤ sint.Z cv ∧ sint.Z cv < sint.Z q.len)]
      wp_apply wp_load_slice_index q (sint.Z cv) current.queue (DFrac.own 1) currentStep.ref
          (by word) $$ [Hqueue] with Hqueue
      · iframe Hqueue; ipureintro; simpa only [← Hindex] using currentStep.head
      try wp_auto
      have Hi : sint.Z (W64 (uint.Z (currentStep.ref - W32 1))) = uint.Z currentStep.ref - 1 := by
        obtain ⟨_, _⟩ := Href; word
      rw [Hi]
      rw [ite_eq_left (by obtain ⟨_, _⟩ := Href; constructor <;> word :
        0 ≤ uint.Z currentStep.ref - 1 ∧ uint.Z currentStep.ref - 1 < sint.Z flags.len)]
      wp_apply wp_store_slice_index flags (uint.Z currentStep.ref - 1) current.flags false
          $$ [Hflags] with Hflags
      · iframe Hflags; ipureintro; obtain ⟨_, _⟩ := Href; constructor <;> omega
      try wp_auto
      list_elem locations (queueIndex currentStep.ref) as source
      have Hlookup : current.inputs[queueIndex currentStep.ref]? = some (some currentStep.sourceMap) := by
        have h := currentStep.inputAt
        unfold WorkState.input at h
        cases hv : current.inputs[queueIndex currentStep.ref]? with
        | none => simp only [hv, Option.getD_none] at h; cases h
        | some value => simpa only [hv, Option.getD_some] using congrArg some h
      ihave ⟨Hsource, Hrestore⟩ := inputStatesOwn_acc locations current.inputs
        (queueIndex currentStep.ref) source (some currentStep.sourceMap) Hsource_lookup Hlookup $$ Hmaps
      iunfold ownsBranch at Hsource
      rw [Hi]
      rw [ite_eq_left (by unfold queueIndex at Hidx; constructor <;> word :
        0 ≤ uint.Z currentStep.ref - 1 ∧ uint.Z currentStep.ref - 1 < sint.Z inputs.len)]
      wp_apply wp_load_slice_index inputs (uint.Z currentStep.ref - 1) locations (DFrac.own 1) source
          (by unfold queueIndex at Hptridx; omega) $$ [Hinputs] with Hinputs
      · iframe Hinputs; ipureintro; exact Hsource_lookup
      wp_apply wp_copyState source currentStep.sourceMap $$ [Hsource] with %copied ⟨Hsource, Hcopy⟩
      · iframe Hsource
      ihave Howned : (ownsBranch (some currentStep.sourceMap) source : IProp GF) $$ [Hsource]
      · unfold ownsBranch; iexact Hsource
      ihave Hmaps := Hrestore $$ %source %(some currentStep.sourceMap) Howned
      rw [list_set_lookup_self locations (queueIndex currentStep.ref) source Hsource_lookup,
        list_set_lookup_self current.inputs (queueIndex currentStep.ref) (some currentStep.sourceMap) Hlookup]
      try wp_auto
      rw [Hi]
      rw [ite_eq_left (by constructor <;> word :
        0 ≤ uint.Z currentStep.ref - 1 ∧ uint.Z currentStep.ref - 1 < sint.Z flow.Blocks'.len)]
      wp_apply wp_load_slice_index flow.Blocks' (uint.Z currentStep.ref - 1) blocks DFrac.discard currentStep.block
          (by unfold queueIndex at Hblock; obtain ⟨_, _⟩ := Hrefblocks; omega) $$ [Hblocks] with Hblocks
      · iframe Hblocks; ipureintro; exact Hblock
      try wp_auto
      wp_apply wp_block_scalar receiver
        ({ view' := view, inputs' := inputs, queued' := flags, queue' := q } : analysis)
        s currentStep.block (operations currentStep.ref) copied currentStep.sourceMap currentStep.targetMap
        currentStep.transfer Hscalar wf hparse huparse $$ [Ha Hsnapshot Hops Hroots Hcopy]
        with ⟨Ha, Hsnapshot, Hroots, Hcopy, %Htargetscalar⟩
      · iframe Ha Hsnapshot Hops Hroots Hcopy
      try wp_auto
      let dequeued := current.dequeue currentStep.ref
      have Hd := WorkState.dequeue_invariant s kind changing index current currentStep.ref Hinv currentStep.head
      have Htarget := currentStep.transfer.valid s kind changing (operations currentStep.ref)
        currentStep.sourceMap currentStep.targetMap agreement Hsource
      have Hnextvalid : ∀ next ∈ nexts currentStep.ref, queueRefValid dequeued.flags next := by
        intro next hn
        simpa only [dequeued, WorkState.dequeue, queueRefValid, List.length_set, Hshape.2]
          using hrefs currentStep.ref currentStep.block Hblock next hn
      ihave Hwork : (workStateOwn receiver view inputs flags q locations dequeued : IProp GF)
          $$ [Ha Hinputs Hmaps Hflags Hqueue Hcap]
      · unfold workStateOwn
        dsimp only [dequeued, WorkState.dequeue, queueIndex]
        iframe Ha Hinputs Hmaps Hflags Hqueue Hcap
      wp_apply wp_successors_invariant receiver view copied s flow blocks currentStep.block (nexts currentStep.ref)
        currentStep.ref currentStep.targetMap inputs flags q locations dequeued o kind changing currentStep.result
        currentStep.decision wf hparse Hrefblocks Hblock Hnextvalid
        (by simpa only [dequeued, WorkState.dequeue, List.length_set] using Hshape.1)
        Hd.2.1 Htarget (index + 1) Hd.1
        $$ [Hsnapshot Hflow Hblocks Hnexts Hoperators Hwork Hcopy]
        with ⟨%out, %newLocations, Hwork, Hcopy, %HnextInvariant⟩
      · iframe Hsnapshot Hflow Hblocks Hnexts Hoperators Hwork Hcopy
      have HnextFull := currentStep.next_invariant s o kind changing blocks operations nexts
        index current Hinv agreement hk (by
          intro next hn
          simpa only [queueRefValid, Hshape.2] using hrefs currentStep.ref currentStep.block Hblock next hn)
      have Hnewshape := currentStep.next_shape s o kind changing blocks operations nexts index current
      have Hlenbound := SolverExecution.queue_length s o kind changing blocks operations nexts
        index current final (.step _ _ _ currentStep rest)
      wp_for_post
      iframe
      iexists (cv + W64 1), out, newLocations,
        (currentStep.next s o kind changing blocks operations nexts index current)
      dsimp only [SolverStep.next, dequeued]
      iframe Hwork Hroots Hcursor
      ipureintro
      have Hnat : sint.nat (cv + W64 1) = index + 1 := by word
      exact ⟨by word, by simpa only [Hnat, SolverStep.next] using HnextFull,
        ⟨by simpa only [SolverStep.next] using (Hnewshape.2.trans Hshape.1).trans Hnewshape.1.symm,
          by simpa only [SolverStep.next] using Hnewshape.2.trans Hshape.2⟩,
        by simpa only [Hnat, SolverStep.next] using rest⟩
  · simp only [Hif, ite_false, _root_.decide_false]
    cases Hremaining with
    | done here state exhausted =>
      have Hcv : cv = W64 final.queue.length := by word
      subst cv
      try wp_auto
      simp only [ite_true]
      iapply HΦ
      all_goals iexists q, locations
      all_goals iunfold workStateOwn
      all_goals iframe Ha Hinputs Hmaps Hflags Hqueue Hcap Hroots Hcursor
      all_goals ipureintro; exact ⟨by simpa only [exhausted] using Hinv, Hextends⟩
    | step here state final currentStep rest =>
      have Hhead := (List.getElem?_eq_some_iff.mp currentStep.head).choose
      word


/-- The finite trace comes from the acyclic budget and primitive execution data.
    No success certificate or child correctness theorem is a premise. The source,
    constructor and machine resource obligations remain explicit. -/
theorem wp_solve_cursor_ranked (receiver view graphLocal cursor : Loc) (s : QuerySnapshot)
    (flow : pdg.NormalFlow) (blocks : List pdg.FlowBlock) (operations nexts : w32 → List w32)
    (inputs flags queue : GoSlice) (pointers : List Loc) (initial : WorkState)
    (o : AssumeOperators) (kind : w32 → GoString) (changing : w32 → Bool)
    (weight : w32 → Nat) (budget : ∀ ref, queueWeight weight (nexts ref) < weight ref)
    (complete : SolverDataComplete s o kind changing blocks operations nexts)
    (hinvariant : scalarWorkInvariant s kind changing 0 initial)
    (hshape : initial.flags.length = initial.inputs.length)
    (hblocks : initial.flags.length = blocks.length) (hreachable : reachableQueue initial)
    (agreement : metadataAgreement s kind changing)
    (hk : ∀ key, kind key ≠ go!"\"pointer\"")
    (hrefs : ∀ ref block, blocks[queueIndex ref]? = some block →
      ∀ next ∈ nexts ref, 0 < uint.Z next ∧ uint.Z next ≤ blocks.length)
    (resource : ∀ final, SolverExecution s o kind changing blocks operations nexts 0 initial final →
      final.queue.length < 2^63 - 1)
    (wf : ValueSnapshotValid s) (hparse : parseInt64Contract (GF := GF))
    (huparse : parseEmptyUint32Contract (GF := GF)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗
       isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
       isPkgInit (PROP := IProp GF) pkg_id.strconv ∗
       (querySnapshotOwn view s ∗ s.graph.ValueFlow' ↦□ flow ∗ flow.Blocks' ↦*□ blocks ∗
        solverRowsOwn blocks operations nexts ∗ assumeOperatorsOwn o ∗
        graphLocal ↦□ s.view.graph' ∗ cursor ↦ (W64 0) ∗
        localRootsOwn (GF := GF) s.view.fieldRoots' ∅ ∗
        workStateOwn receiver view inputs flags queue pointers initial) }}
      (solveCursorRun receiver graphLocal cursor)
    {{ RET executeVal; ∃ (final : WorkState) (out : GoSlice) (locations : List Loc),
        workStateOwn receiver view inputs flags out locations final ∗
        localRootsOwn (GF := GF) s.view.fieldRoots' ∅ ∗
        cursor ↦ (W64 final.queue.length) ∗
        ⌜scalarWorkInvariant s kind changing final.queue.length final ∧ inputsExtend kind changing initial final ∧ reachableQueue final⌝ }} := by
  obtain ⟨final, execution⟩ := solverExecution_exists s o kind changing blocks operations nexts weight budget
    complete agreement hk hrefs 0 initial hinvariant hshape hblocks hreachable
  have hfinal := SolverExecution.reachable s o kind changing blocks operations nexts 0 initial final
    execution hreachable (hshape.symm.trans hblocks) (by
      intro ref block hblock next hn
      have hvalid := hrefs ref block hblock next hn
      unfold queueIndex
      omega)
  wp_start_folded as Hpre
  wp_apply wp_solve_cursor receiver view graphLocal cursor s flow blocks operations nexts
    inputs flags queue pointers initial final o kind changing execution hinvariant hshape hblocks
    agreement hk hrefs (resource final execution) wf hparse huparse $$ [Hpre]
    with ⟨%out, %locations, Hwork, Hroots, Hcursor, %Hmeaning⟩
  · iexact Hpre
  iapply HΦ
  iexists final, out, locations
  iframe Hwork Hroots Hcursor
  ipureintro
  exact ⟨Hmeaning.1, Hmeaning.2, hfinal⟩

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
