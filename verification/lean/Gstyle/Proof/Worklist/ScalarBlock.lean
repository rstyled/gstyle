module
public import Gstyle.Proof.Worklist.ScalarTransfer
public import Gstyle.Proof.Worklist.FlowLayout
public import Gstyle.Proof.Worklist.Layout
public import Gstyle.Proof.Transfer.Assignment
public import Gstyle.Proof.Transfer.LocalInvalidation
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





variable [analysis.TypeAssumptions]
variable {dq : DFrac}
variable [FuncUnfold fieldIdentity [] fieldIdentity.impl]
variable [FuncUnfold singleValueWrite [] singleValueWrite.impl]


/-- Producer data for one admitted signed assignment; no execution or soundness
    callback is stored here. Source lowering must establish these fields. -/
structure ScalarWriteSite (s : QuerySnapshot) (ref : w32) where
  child : w32
  node : pdg.Node
  row : GoArray w32 2
  write : pdg.Access
  sf : Nat
  href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.nodes.length
  hnode : s.nodes[(uint.Z ref - 1).toNat]? = some node
  hrow : s.operands[(uint.Z ref - 1).toNat]? = some row
  hchild : row.arr[0]? = some child
  hwrites : pdg.rowContents s.accessRows (snapshotRecord s ref).Writes' = [write]
  hselect : selectsRhs node.Category'
    (pdg.tableText s.strings (snapshotRecord s ref).WriteForm') (W64 1) = true
  hk : 0 < uint.Z write.Symbol' ∧ uint.Z write.Symbol' ≤ s.kinds.length
  hkind : s.kinds[(uint.Z write.Symbol' - 1).toNat]? = some (go!"\"signed\"")
  ht : 0 < uint.Z write.Symbol' ∧ uint.Z write.Symbol' ≤ s.tracked.length
  htracked : s.tracked[(uint.Z write.Symbol' - 1).toNat]? = some true
  hstep : snapshotFact s ref (go!"signed-step64") = go!""
  hrange : snapshotFact s ref (go!"range-index") = go!""

inductive ScalarStep (s : QuerySnapshot) : w32 → StateData → StateData → Prop where
  | noWrites (ref : w32) (m : StateData)
      (hvalid : validSymbolRef s ref)
      (hwrites : pdg.rowContents s.accessRows (snapshotRecord s ref).Writes' = []) :
      ScalarStep s ref m m
  | write (ref : w32) (m : StateData) (site : ScalarWriteSite s ref) (raw : Value)
      (evaluation : ValueEvaluation s m site.sf site.child raw)
      (hidentity : raw.Identity' = W64 0) :
      ScalarStep s ref m (<[site.write.Symbol' := { raw with Guarded' := false }]>m)

inductive ScalarExecution (s : QuerySnapshot) : List w32 → StateData → StateData → Prop where
  | nil (m : StateData) : ScalarExecution s [] m m
  | cons (ref : w32) (tail : List w32) (m middle out : StateData)
      (step : ScalarStep s ref m middle) (rest : ScalarExecution s tail middle out) :
      ScalarExecution s (ref :: tail) m out

theorem scalarState_insert (m : StateData) (symbol : w32) (raw : Value)
    (hm : scalarState m) (hi : raw.Identity' = W64 0) :
    scalarState (<[symbol := { raw with Guarded' := false }]>m) := by
  intro key value hv
  by_cases he : symbol = key
  · subst key
    simp only [GMap.lookup_insert] at hv
    cases hv
    exact hi
  · simp only [GMap.lookup_insert_eq_iff, he, Ne.symm he] at hv
    exact hm key value hv


/-- Clearing pointer-guard metadata keeps both numeric bounds unchanged. -/
theorem scalar_store_sound (kind : w32 → GoString) (changing : w32 → Bool)
    (m : StateData) (env : w32 → Int) (symbol : w32) (raw : Value) (x : Int)
    (hm : stateDenotes kind changing m env) (hstable : changing symbol = false)
    (hx : interval.contains (bounds raw) x) :
    stateDenotes kind changing (<[symbol := { raw with Guarded' := false }]>m)
      (fun key => if key = symbol then x else env key) := by
  intro key
  by_cases he : key = symbol
  · subst key
    simpa only [bindingValue, hstable, Bool.false_eq_true, ite_false,
      GMap.lookup_insert, Option.getD_some, ite_true, bounds] using hx
  · have hne : symbol ≠ key := Ne.symm he
    simpa only [bindingValue, GMap.lookup_insert_eq_iff, he, hne, ite_false] using hm key

theorem scalar_store_valid (s : QuerySnapshot) (kind : w32 → GoString) (changing : w32 → Bool)
    (m : StateData) (symbol : w32) (raw : Value)
    (hm : validBranchState s.kinds s.unstable kind changing m)
    (hk : 0 < uint.Z symbol ∧ uint.Z symbol ≤ s.kinds.length)
    (hkind : s.kinds[(uint.Z symbol - 1).toNat]? = some (kind symbol))
    (hchanging : s.unstable[(uint.Z symbol - 1).toNat]? = some (changing symbol)) :
    validBranchState s.kinds s.unstable kind changing
      (<[symbol := { raw with Guarded' := false }]>m) := by
  intro key hpresent
  by_cases he : symbol = key
  · subst key
    exact ⟨hk, hkind, hchanging⟩
  · apply hm key
    simpa only [GMap.lookup_insert_eq_iff, he, Ne.symm he, ite_false] using hpresent

theorem ScalarStep.preserves_scalar (s : QuerySnapshot) (ref : w32) (m n : StateData)
    (step : ScalarStep s ref m n) (hm : scalarState m) : scalarState n := by
  cases step with
  | noWrites => exact hm
  | write site raw evaluation hi => exact scalarState_insert m site.write.Symbol' raw hm hi

theorem wp_transfer_scalar_step (receiver : Loc) (a : analysis) (s : QuerySnapshot)
    (ref : w32) (state : Loc) (m n : StateData) (step : ScalarStep s ref m n)
    (hm : scalarState m) (wf : ValueSnapshotValid s)
    (hparse : parseInt64Contract (GF := GF)) (huparse : parseEmptyUint32Contract (GF := GF)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.strconv ∗
       (receiver ↦{dq} a ∗ querySnapshotOwn a.view' s ∗
        localRootsOwn (GF := GF) s.view.fieldRoots' ∅ ∗ state ↦$ m) }}
      (App (App (Val #(methods (go.PointerType analysis.ty) go!"transfer" #receiver))
        (Val #ref)) (Val #state))
    {{ RET #(); receiver ↦{dq} a ∗ querySnapshotOwn a.view' s ∗
        localRootsOwn (GF := GF) s.view.fieldRoots' ∅ ∗ state ↦$ n ∗ ⌜scalarState n⌝ }} := by
  have hn := step.preserves_scalar s ref m n hm
  cases step with
  | noWrites hv hw =>
    wp_start_folded as Hpre
    wp_apply wp_transfer_no_writes receiver a s ref state m hm hv wf.symbols.allocation hw
        $$ [Hpre] with ⟨Ha, Hsnapshot, Hroots, Hstate⟩
    · iframe Hpre
    wp_end
    all_goals iframe Ha Hsnapshot Hroots Hstate
    all_goals ipureintro; exact hn
  | write site raw evaluation hi =>
    wp_start_folded as Hpre
    wp_apply wp_transfer_scalar_write receiver a s ref site.child site.node site.row site.write
      state m raw site.sf hm wf hparse huparse site.href site.hnode site.hrow site.hchild
      site.hwrites site.hselect site.hk site.hkind site.ht site.htracked site.hstep site.hrange evaluation
      $$ [Hpre] with ⟨Ha, Hsnapshot, Hroots, Hstate⟩
    · iframe Hpre
    wp_end
    all_goals iframe Ha Hsnapshot Hroots Hstate
    all_goals ipureintro; exact hn

variable [pdg.FlowBlock.TypeAssumptions]

/-- The solver's actual block loop, with nil report, executes every scalar step.
    The list is arbitrary finite; no syntax-depth or operation-count bound is used. -/
theorem wp_block_scalar (receiver : Loc) (a : analysis) (s : QuerySnapshot)
    (block : pdg.FlowBlock) (operations : List w32) (state : Loc) (m n : StateData)
    (execution : ScalarExecution s operations m n) (hm : scalarState m)
    (wf : ValueSnapshotValid s) (hparse : parseInt64Contract (GF := GF))
    (huparse : parseEmptyUint32Contract (GF := GF)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.strconv ∗
       (receiver ↦{dq} a ∗ querySnapshotOwn a.view' s ∗ block.Operations' ↦*□ operations ∗
        localRootsOwn (GF := GF) s.view.fieldRoots' ∅ ∗ state ↦$ m) }}
      (App (App (App (Val #(methods (go.PointerType analysis.ty) go!"block" #receiver))
        (Val #block)) (Val #state)) (Val #null))
    {{ RET #(); receiver ↦{dq} a ∗ querySnapshotOwn a.view' s ∗
        localRootsOwn (GF := GF) s.view.fieldRoots' ∅ ∗ state ↦$ n ∗ ⌜scalarState n⌝ }} := by
  wp_start as ⟨Ha, #Hsnapshot, #Hoperations, Hroots, Hstate⟩
  ihave %Hlen := ownSlice_len $$ Hoperations
  try wp_auto
  ihave HI : (∃ (iv : w64) (rv : w32) (current : StateData),
      "i" ∷ i_ptr ↦ iv ∗ "ref" ∷ ref_ptr ↦ rv ∗
      "a" ∷ a_ptr ↦ receiver ∗ "state" ∷ state_ptr ↦ state ∗
      "report" ∷ report_ptr ↦ null ∗ "block" ∷ block_ptr ↦ block ∗
      "Ha" ∷ receiver ↦{dq} a ∗
      "Hroots" ∷ localRootsOwn (GF := GF) s.view.fieldRoots' ∅ ∗
      "Hstate" ∷ state ↦$ current ∗
      "%Hbounds" ∷ ⌜0 ≤ sint.Z iv ∧ sint.Z iv ≤ sint.Z block.Operations'.len⌝ ∗
      "%Hremaining" ∷ ⌜ScalarExecution s (operations.drop (sint.nat iv)) current n⌝ ∗
      "%Hscalar" ∷ ⌜scalarState current⌝ : IProp GF)
      $$ [i ref a state report block Ha Hroots Hstate]
  · iexists (W64 0), (zero_val w32), m
    simp only [show sint.nat (W64 0) = 0 from rfl, List.drop_zero]
    iframe i ref a state report block Ha Hroots Hstate
    ipureintro
    exact ⟨by constructor <;> word, execution, hm⟩
  wp_for HI
  simp only [go.intoVal_inj.eq_iff, decide_eq_true_eq, Bool.false_eq_true, ↓reduceIte]
  by_cases Hif : sint.Z iv < sint.Z block.Operations'.len
  · simp only [Hif, ite_true, _root_.decide_true]
    have hi : sint.nat iv < operations.length := by word
    list_elem operations (sint.nat iv) as operation
    have Hdrop : operations.drop (sint.nat iv) = operation :: operations.drop (sint.nat iv + 1) := by
      rw [List.drop_eq_getElem?_toList_append, Hoperation_lookup]
      rfl
    rw [Hdrop] at Hremaining
    cases Hremaining with
    | cons _ _ _ middle _ step rest =>
      try wp_auto
      rw [ite_eq_left (by constructor <;> word :
        0 ≤ sint.Z iv ∧ sint.Z iv < sint.Z block.Operations'.len)]
      wp_apply wp_load_slice_index block.Operations' (sint.Z iv) operations DFrac.discard operation
          (by word) $$ [Hoperations] with Hoperations
      · iframe Hoperations; ipureintro; exact Hoperation_lookup
      try wp_auto
      wp_apply wp_transfer_scalar_step receiver a s operation state current middle step Hscalar wf hparse huparse
          $$ [Ha Hsnapshot Hroots Hstate] with ⟨Ha, Hsnapshot, Hroots, Hstate, %Hnextscalar⟩
      · iframe Ha Hsnapshot Hroots Hstate
      try wp_auto
      wp_for_post
      iframe
      iexists (iv + W64 1), operation, middle
      iframe
      ipureintro
      have hn : sint.nat (iv + W64 1) = sint.nat iv + 1 := by word
      exact ⟨by constructor <;> word, by simpa only [hn] using rest, Hnextscalar⟩
  · simp only [Hif, ite_false, _root_.decide_false]
    have he : sint.nat iv = operations.length := by word
    rw [he, List.drop_length] at Hremaining
    cases Hremaining
    wp_end
    all_goals iframe Ha Hsnapshot Hroots Hstate
    all_goals ipureintro; exact Hscalar

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
