module
public import Gstyle.Proof.Worklist.SetupSyntax
public import Gstyle.Proof.Worklist.Traversal
public import Gstyle.Proof.Worklist.Initial
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
variable [pdg.Access.TypeAssumptions] [pdg.Tables.TypeAssumptions] [pdg.Extension.TypeAssumptions]
variable [Value.TypeAssumptions] [State.TypeAssumptions]
variable [analysis.TypeAssumptions] [pdg.NormalFlow.TypeAssumptions] [pdg.FlowBlock.TypeAssumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard
attribute [local goose_wp_simp] setupContext.eq_1 setupContext.eq_2 setupContext_terminal cursorHead cursorStart

theorem inputStatesOwn_nil (count : Nat) :
    ⊢ inputStatesOwn (GF := GF) (List.replicate count map.nil) (List.replicate count none) := by
  induction count with
  | zero => simp only [List.replicate_zero, inputStatesOwn]; ipureintro; trivial
  | succ count ih =>
    simp only [List.replicate_succ, inputStatesOwn, ownsBranch]
    isplitr
    · ipureintro; trivial
    · exact ih

theorem inputStatesOwn_seed (count index : Nat) (state : Loc) (hi : index < count) :
    state ↦$ (∅ : StateData) ⊢ inputStatesOwn (GF := GF)
      ((List.replicate count map.nil).set index state)
      ((List.replicate count none).set index (some ∅)) := by
  iintro Hstate
  ihave Hnil := inputStatesOwn_nil (GF := GF) count
  ihave ⟨_, Hrestore⟩ := inputStatesOwn_acc
    (List.replicate count map.nil) (List.replicate count none) index map.nil none
    (by simp [hi]) (by simp [hi]) $$ Hnil
  iapply Hrestore $$ %state %(some ∅) [Hstate]
  iunfold ownsBranch
  iexact Hstate

/-- The actual allocation, entry store and enqueue produce these exact containers.
    Snapshot construction is composed separately; no initial ownership is assumed. -/
theorem wp_solve_setup_raw (graphLocal viewLocal graph view : Loc)
    (g : pdg.Graph) (flow : pdg.NormalFlow) (blocks : List pdg.FlowBlock)
    (entry : w32) (hentry : flow.Entry' = entry)
    (href : 0 < uint.Z entry ∧ uint.Z entry ≤ blocks.length)
    (halloc : blocks.length ≤ 2^45) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ graphLocal ↦ graph ∗ viewLocal ↦ view ∗
       (graph ↦□ g ∗ g.ValueFlow' ↦□ flow ∗ flow.Blocks' ↦*□ blocks) }}
      (solveSetupRawRun graphLocal viewLocal)
    {{ (receiver : Loc), RET (returnVal (PairV #receiver #interface.nil));
       graphLocal ↦ graph ∗ viewLocal ↦ view ∗
       ∃ (inputs flags queue : GoSlice) (pointers : List Loc),
         workStateOwn receiver view inputs flags queue pointers (initialWorkState blocks.length entry) }} := by
  wp_start_folded as ⟨Hg, Hview, #Hgraph, #Hflow, #Hblocks⟩
  isimp only [initialWorkState] at HΦ
  ihave %Hblen := ownSlice_len $$ Hblocks
  unfold solveSetupRawRun solveSetupBody solveAnalysisBody solveBody solve.impl
  dsimp only [analysisSite, Option.or, Option.getD]
  wp_expr_simp
  try wp_auto
  wp_apply wp_slice_make2 (V := Loc) flow.Blocks'.len (by
      intro _; simp only [go.typeSize_loc]; word) $$ [] with %inputs ⟨Hinputs, HinputCap⟩
  · ipureintro; exact Hblen.2
  try wp_auto
  wp_apply wp_slice_make2 (V := Bool) flow.Blocks'.len (by
      intro _; simp only [go.typeSize_bool]; word) $$ [] with %flags ⟨Hflags, HflagCap⟩
  · ipureintro; exact Hblen.2
  try wp_auto
  wp_apply wp_map_make1 (K := w32) (V := Value) pdg.Ref.ty Value.ty with %entryState HentryMap
  try wp_auto
  ihave %Hilen := ownSlice_len $$ Hinputs
  simp only [List.length_replicate] at Hilen
  have Hi : sint.Z (W64 (uint.Z (entry - W32 1))) = uint.Z entry - 1 := by
    obtain ⟨_, _⟩ := href; word
  rw [hentry, Hi]
  rw [ite_eq_left (by obtain ⟨_, _⟩ := href; constructor <;> word :
    0 ≤ uint.Z entry - 1 ∧ uint.Z entry - 1 < sint.Z inputs.len)]
  wp_apply wp_store_slice_index inputs (uint.Z entry - 1)
      (List.replicate (sint.nat flow.Blocks'.len) (zero_val Loc)) entryState $$ [Hinputs] with Hinputs
  · iframe Hinputs; ipureintro; simp only [List.length_replicate]
    obtain ⟨_, _⟩ := href; constructor <;> omega
  try wp_auto
  rw [hentry]
  have Hcount : sint.nat flow.Blocks'.len = blocks.length := Hblen.1.symm
  isimp only [Hcount] at Hinputs Hflags
  ihave Hqueue := ownSlice_nil (GF := GF) (V := w32)
  ihave Hcap := ownSliceCap_nil (GF := GF) (V := w32)
  wp_apply wp_enqueue a_ptr
      ({view' := view, inputs' := inputs, queued' := flags, queue' := slice.nil} : analysis)
      (List.replicate blocks.length false) [] entry false
      (by simpa using href)
      (by simpa only [unqueuedWorkState, queueIndex] using unqueuedWorkState_entry_flag blocks.length entry href)
      $$ [a Hflags Hqueue Hcap] with ⟨%queue, Ha, Hflags, Hqueue, Hcap⟩
  · iframe a Hflags Hqueue Hcap
  try wp_auto
  iapply HΦ
  iframe Hg Hview
  iexists inputs, flags, queue, ((List.replicate blocks.length map.nil).set (queueIndex entry) entryState)
  iunfold workStateOwn
  ihave Hmaps := inputStatesOwn_seed (GF := GF) blocks.length (queueIndex entry) entryState
      (by unfold queueIndex; omega) $$ HentryMap
  simp only [queueIndex]
  have Hnil : zero_val Loc = map.nil := by rfl
  isimp only [Hnil] at Hinputs
  isimp only [List.nil_append] at Hqueue
  iframe Ha Hinputs Hmaps Hflags Hqueue Hcap

theorem wp_solve_setup_containers (graphLocal viewLocal graph view : Loc)
    (g : pdg.Graph) (flow : pdg.NormalFlow) (blocks : List pdg.FlowBlock)
    (entry : w32) (hentry : flow.Entry' = entry)
    (href : 0 < uint.Z entry ∧ uint.Z entry ≤ blocks.length)
    (halloc : blocks.length ≤ 2^45) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ graphLocal ↦ graph ∗ viewLocal ↦ view ∗
       (graph ↦□ g ∗ g.ValueFlow' ↦□ flow ∗ flow.Blocks' ↦*□ blocks) }}
      (solveSetupRun graphLocal viewLocal)
    {{ (receiver : Loc), RET (PairV #receiver #interface.nil);
       graphLocal ↦ graph ∗ viewLocal ↦ view ∗
       ∃ (inputs flags queue : GoSlice) (pointers : List Loc),
         workStateOwn receiver view inputs flags queue pointers (initialWorkState blocks.length entry) }} := by
  wp_start_folded as ⟨Hg, Hview, #Hgraph, #Hflow, #Hblocks⟩
  unfold solveSetupRun
  wp_apply wp_solve_setup_raw graphLocal viewLocal graph view g flow blocks entry hentry href halloc
      $$ [Hg Hview] with %receiver ⟨Hg, Hview, %inputs, %flags, %queue, %pointers, Hwork⟩
  · iframe Hg Hview Hgraph Hflow Hblocks
  try wp_auto
  iapply HΦ
  iframe Hg Hview
  iexists inputs, flags, queue, pointers
  iexact Hwork

/-- Empty-map validity, queue reachability and the scalar invariant are conclusions. -/
theorem wp_solve_setup_invariant (graphLocal viewLocal graph view : Loc)
    (g : pdg.Graph) (flow : pdg.NormalFlow) (blocks : List pdg.FlowBlock)
    (s : QuerySnapshot) (kind : w32 → GoString) (changing : w32 → Bool)
    (entry : w32) (hentry : flow.Entry' = entry)
    (href : 0 < uint.Z entry ∧ uint.Z entry ≤ blocks.length)
    (halloc : blocks.length ≤ 2^45) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ graphLocal ↦ graph ∗ viewLocal ↦ view ∗
       (graph ↦□ g ∗ g.ValueFlow' ↦□ flow ∗ flow.Blocks' ↦*□ blocks) }}
      (solveSetupRun graphLocal viewLocal)
    {{ (receiver : Loc), RET (PairV #receiver #interface.nil);
       graphLocal ↦ graph ∗ viewLocal ↦ view ∗
       ∃ (inputs flags queue : GoSlice) (pointers : List Loc),
         workStateOwn receiver view inputs flags queue pointers (initialWorkState blocks.length entry) ∗
         ⌜scalarWorkInvariant s kind changing 0 (initialWorkState blocks.length entry) ∧
           (initialWorkState blocks.length entry).inputs.length = blocks.length ∧
           (initialWorkState blocks.length entry).flags.length = blocks.length ∧
           reachableQueue (initialWorkState blocks.length entry)⌝ }} := by
  wp_start_folded as ⟨Hg, Hview, #Hgraph, #Hflow, #Hblocks⟩
  wp_apply wp_solve_setup_containers graphLocal viewLocal graph view g flow blocks entry hentry href halloc
      $$ [Hg Hview] with %receiver ⟨Hg, Hview, %inputs, %flags, %queue, %pointers, Hwork⟩
  · iframe Hg Hview Hgraph Hflow Hblocks
  iapply HΦ
  iframe Hg Hview
  iexists inputs, flags, queue, pointers
  iframe Hwork
  ipureintro
  exact ⟨initialWorkState_scalarWorkInvariant s kind changing blocks.length entry href,
    (initialWorkState_shape blocks.length entry).1, (initialWorkState_shape blocks.length entry).2,
    initialWorkState_reachableQueue blocks.length entry href⟩

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
