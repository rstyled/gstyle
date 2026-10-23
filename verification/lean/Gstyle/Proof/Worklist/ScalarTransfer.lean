module
public import Gstyle.Proof.Worklist.ScalarGuards
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
variable [FuncUnfold fieldIdentity [] fieldIdentity.impl]
variable [FuncUnfold singleValueWrite [] singleValueWrite.impl]

/-- Scalar values carry no pointer identity; field-root metadata is separate. -/
def scalarState (m : StateData) : Prop :=
  ∀ key value, m !! key = some value → value.Identity' = W64 0

theorem wp_invalidateFields_scalar (receiver : Loc) (s : QuerySnapshot)
    (ref : w32) (state : Loc) (m : StateData) (hm : scalarState m) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗
       (querySnapshotOwn receiver s ∗ localRootsOwn (GF := GF) s.view.fieldRoots' ∅ ∗ state ↦$ m) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"invalidateFields" #receiver))
        (Val #ref)) (Val #state))
    {{ RET #(); querySnapshotOwn receiver s ∗ localRootsOwn (GF := GF) s.view.fieldRoots' ∅ ∗ state ↦$ m }} := by
  wp_start_folded as ⟨#Hsnapshot, Hroots, Hstate⟩
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, Hgraph, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _⟩
  have Hlocal : ∀ key value, m !! key = some value →
      ((∅ : GMap w32 w32) !! key).getD (W32 0) = W32 0 ∧
      localIdentity value.Identity' s.graph.Symbols'.len := by
    intro key value hv
    exact ⟨by simp, Or.inl (hm key value hv)⟩
  wp_apply wp_invalidateFields_local receiver s.view s.graph ∅ state m ref Hlocal
      $$ [Hview Hgraph Hroots Hstate] with ⟨Hview, Hgraph, Hroots, Hstate⟩
  · iframe Hview Hgraph Hroots Hstate
  wp_end
  all_goals iframe Hsnapshot Hroots Hstate

/-- Actual early return, including field invalidation; no helper outcome is assumed. -/
theorem wp_transfer_no_writes (receiver : Loc) (a : analysis) (s : QuerySnapshot)
    (ref : w32) (state : Loc) (m : StateData) (hm : scalarState m)
    (hvalid : validSymbolRef s ref)
    (halloc : ∀ b ∈ s.attributes, (pdg.rowContents s.referenceRows b.Outcomes').length ≤ 2^46)
    (hwrites : pdg.rowContents s.accessRows (snapshotRecord s ref).Writes' = []) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗
       (receiver ↦□ a ∗ querySnapshotOwn a.view' s ∗
        localRootsOwn (GF := GF) s.view.fieldRoots' ∅ ∗ state ↦$ m) }}
      (App (App (Val #(methods (go.PointerType analysis.ty) go!"transfer" #receiver))
        (Val #ref)) (Val #state))
    {{ RET #(); receiver ↦□ a ∗ querySnapshotOwn a.view' s ∗
        localRootsOwn (GF := GF) s.view.fieldRoots' ∅ ∗ state ↦$ m }} := by
  wp_start as ⟨#Ha, #Hsnapshot, Hroots, Hstate⟩
  try wp_auto
  wp_apply wp_attrs_snapshot_full a.view' s ref hvalid halloc
      $$ [Hsnapshot] with %attrs ⟨Hsnapshot, %hattrs⟩
  · iframe Hsnapshot
  have hws : attrs.Writes' = pdg.listRow s.accesses (snapshotRecord s ref).Writes' :=
    congrArg pdg.Attributes.Writes' hattrs
  ihave #Hrows := querySnapshotOwn_rows a.view' s (snapshotRecord s ref).Writes' $$ Hsnapshot
  icases Hrows with ⟨_, Hwrites, _⟩
  ihave %Hlen := ownSlice_len $$ Hwrites
  rw [hwrites] at Hlen
  try wp_auto
  wp_apply wp_invalidateFields_scalar a.view' s ref state m hm
      $$ [Hsnapshot Hroots Hstate] with ⟨Hsnapshot, Hroots, Hstate⟩
  · iframe Hsnapshot Hroots Hstate
  try wp_auto
  rw [hws]
  have Hzero : (pdg.listRow s.accesses (snapshotRecord s ref).Writes').len = W64 0 := by
    obtain ⟨Hnat, Hpositive⟩ := Hlen
    simp only [List.length_nil] at Hnat
    word
  rw [Hzero]
  try wp_auto
  wp_end
  all_goals iframe Ha Hsnapshot Hroots Hstate

/-- Real single scalar assignment: evaluate the selected RHS, clear Guarded,
    preserve scalar helper guards, and store exactly the written binding. -/
theorem wp_transfer_scalar_write (receiver : Loc) (a : analysis) (s : QuerySnapshot)
    (ref child : w32) (node : pdg.Node) (row : GoArray w32 2) (write : pdg.Access)
    (state : Loc) (m : StateData) (raw : Value) (sf : Nat)
    (hm : scalarState m) (wf : ValueSnapshotValid s)
    (hparse : parseInt64Contract (GF := GF)) (huparse : parseEmptyUint32Contract (GF := GF))
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.nodes.length)
    (hnode : s.nodes[(uint.Z ref - 1).toNat]? = some node)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hchild : row.arr[0]? = some child)
    (hwrites : pdg.rowContents s.accessRows (snapshotRecord s ref).Writes' = [write])
    (hselect : selectsRhs node.Category'
      (pdg.tableText s.strings (snapshotRecord s ref).WriteForm') (W64 1) = true)
    (hk : 0 < uint.Z write.Symbol' ∧ uint.Z write.Symbol' ≤ s.kinds.length)
    (hkind : s.kinds[(uint.Z write.Symbol' - 1).toNat]? = some (go!"\"signed\""))
    (ht : 0 < uint.Z write.Symbol' ∧ uint.Z write.Symbol' ≤ s.tracked.length)
    (htracked : s.tracked[(uint.Z write.Symbol' - 1).toNat]? = some true)
    (hstep : snapshotFact s ref (go!"signed-step64") = go!"")
    (hrange : snapshotFact s ref (go!"range-index") = go!"")
    (evaluation : ValueEvaluation s m sf child raw) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.strconv ∗
       (receiver ↦□ a ∗ querySnapshotOwn a.view' s ∗
        localRootsOwn (GF := GF) s.view.fieldRoots' ∅ ∗ state ↦$ m) }}
      (App (App (Val #(methods (go.PointerType analysis.ty) go!"transfer" #receiver))
        (Val #ref)) (Val #state))
    {{ RET #(); receiver ↦□ a ∗ querySnapshotOwn a.view' s ∗
        localRootsOwn (GF := GF) s.view.fieldRoots' ∅ ∗
        state ↦$ (<[write.Symbol' := { raw with Guarded' := false }]>m) }} := by
  wp_start as ⟨#Ha, #Hsnapshot, Hroots, Hstate⟩
  try wp_auto
  wp_apply wp_attrs_snapshot_full a.view' s ref (Or.inr href) wf.symbols.allocation
      $$ [Hsnapshot] with %attrs ⟨Hsnapshot, %hattrs⟩
  · iframe Hsnapshot
  have hws : attrs.Writes' = pdg.listRow s.accesses (snapshotRecord s ref).Writes' :=
    congrArg pdg.Attributes.Writes' hattrs
  have Hform : attrs.WriteForm' = (snapshotRecord s ref).WriteForm' :=
    congrArg pdg.Attributes.WriteForm' hattrs
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, Hgraph, _, Hoperands, Hkinds, _, Htracked, Hnodes,
    _, _, _, _, _, HaccessRows, _, Htable, Hstrings⟩
  ihave #Hwrites := pdg.readonlyRows_at_ref s.accesses s.accessRows
    (snapshotRecord s ref).Writes' $$ HaccessRows
  ihave %Hlen := ownSlice_len $$ Hwrites
  rw [hwrites] at Hlen
  have Hlength : attrs.Writes'.len = W64 1 := by
    rw [hws]
    obtain ⟨Hnat, Hpositive⟩ := Hlen
    simp only [List.length_cons, List.length_nil] at Hnat
    word
  try wp_auto
  wp_apply wp_invalidateFields_scalar a.view' s ref state m hm
      $$ [Hsnapshot Hroots Hstate] with ⟨Hsnapshot, Hroots, Hstate⟩
  · iframe Hsnapshot Hroots Hstate
  try wp_auto
  rw [Hlength]
  try wp_auto
  rw [Hform]
  wp_apply pdg.wp_Text s.graph.Tables' s.table s.strings DFrac.discard
      (snapshotRecord s ref).WriteForm' $$ [Htable Hstrings] with ⟨Htable, Hstrings⟩
  · iframe Htable Hstrings
  try wp_auto
  ihave %Hnodeslen := ownSlice_len $$ Hnodes
  have Hi : sint.Z (W64 (uint.Z (ref - W32 1))) = uint.Z ref - 1 := by
    obtain ⟨_, _⟩ := href; word
  rw [Hi]
  rw [ite_eq_left (by obtain ⟨_, _⟩ := href; constructor <;> word :
    0 ≤ uint.Z ref - 1 ∧ uint.Z ref - 1 < sint.Z s.graph.Nodes'.len)]
  icases ownSlice_elem_acc (uint.Z ref - 1) node s.graph.Nodes' DFrac.discard s.nodes
      (by obtain ⟨_, _⟩ := href; omega) hnode $$ Hnodes with ⟨Hnode, HrestoreNodes⟩
  try wp_auto
  rw [Hlength]
  wp_apply wp_singleValueWrite node.Category'
    (pdg.tableText s.strings (snapshotRecord s ref).WriteForm') (W64 1)
  rw [hselect]
  try wp_auto
  have Ho : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length := by
    rw [wf.symbols.operands]; exact href
  wp_apply wp_operand_readonly a.view' s.view s.operands ref (W64 0) row child Ho
      (by decide) hrow hchild DFrac.discard DFrac.discard $$ [Hview Hoperands]
      with ⟨Hview, Hoperands⟩
  · iframe Hview Hoperands
  wp_apply wp_value_evaluation a.view' s state m sf child raw wf.symbols wf.unstable hparse evaluation
      $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
  · iframe Hsnapshot Hstate
  try wp_auto

  let cleared : Value := { raw with Guarded' := false }
  let final : StateData := <[write.Symbol' := cleared]>m
  ihave HI : (∃ (iv : w64) (wv : pdg.Access),
      "i" ∷ i_ptr ↦ iv ∗ "write" ∷ write_ptr ↦ wv ∗
      "a" ∷ a_ptr ↦ receiver ∗ "state" ∷ state_ptr ↦ state ∗ "ref" ∷ ref_ptr ↦ ref ∗
      "value" ∷ value_ptr ↦ cleared ∗
      "Hroots" ∷ localRootsOwn (GF := GF) s.view.fieldRoots' ∅ ∗
      "Hstate" ∷ state ↦$ (if iv = W64 0 then m else final) ∗
      "%Hbounds" ∷ ⌜0 ≤ sint.Z iv ∧ sint.Z iv ≤ 1⌝ : IProp GF)
      $$ [i write a state ref value Hroots Hstate]
  · iexists (W64 0), (zero_val pdg.Access)
    simp only [ite_true]
    iframe i write a state ref value Hroots Hstate
    ipureintro
    constructor <;> word
  wp_for HI
  simp only [go.intoVal_inj.eq_iff, decide_eq_true_eq, Bool.false_eq_true, ↓reduceIte]
  by_cases Hif : sint.Z iv < sint.Z attrs.Writes'.len
  · simp only [Hif, ite_true, _root_.decide_true]
    have hz : iv = W64 0 := by rw [Hlength] at Hif; word
    subst iv
    try wp_auto
    rw [ite_eq_left (by rw [Hlength]; constructor <;> decide :
      0 ≤ sint.Z (W64 0) ∧ sint.Z (W64 0) < sint.Z attrs.Writes'.len)]
    rw [hws]
    try wp_auto
    wp_apply wp_load_slice_index (t := pdg.Access.ty)
        (pdg.listRow s.accesses (snapshotRecord s ref).Writes') (sint.Z (W64 0))
        (pdg.rowContents s.accessRows (snapshotRecord s ref).Writes') DFrac.discard write
        (by simp [hwrites]) $$ [Hwrites] with Hwrites
    · iframe Hwrites; ipureintro; simp [hwrites]
    try wp_auto
    wp_apply wp_trackedBinding a.view' s.view s.tracked write.Symbol' true ht htracked DFrac.discard
        $$ [Hview Htracked] with ⟨Hview, Htracked⟩
    · iframe Hview Htracked
    try wp_auto
    wp_apply wp_incrementValue_scalar a.view' s ref write.Symbol' cleared state
        (Or.inr href) wf.symbols.allocation hstep $$ [Hsnapshot] with Hsnapshot
    · iframe Hsnapshot
    try wp_auto
    wp_apply wp_interfaceAssignment_scalar a.view' s ref write.Symbol' cleared hk hkind
        $$ [Hsnapshot] with Hsnapshot
    · iframe Hsnapshot
    try wp_auto
    simp only [ite_true]
    wp_apply wp_assign_nonpointer a.view' s.view s.kinds state m ref write.Symbol'
        (go!"\"signed\"") cleared hk hkind (by decide) DFrac.discard
        $$ [Hview Hkinds Hstate] with ⟨Hview, Hkinds, Hstate⟩
    · iframe Hview Hkinds Hstate
    simp only [assignedValue, show (go!"\"signed\"" : GoString) ≠ go!"\"unsigned\"" by decide,
      _root_.false_and, ite_false]
    try wp_auto
    wp_apply wp_rangeWrite_scalar receiver a s ref write.Symbol' state final
        (Or.inr href) wf.symbols.allocation hrange huparse
        $$ [Ha Hsnapshot Hstate] with ⟨Ha, Hsnapshot, Hstate⟩
    · iframe Ha Hsnapshot Hstate
    try wp_auto
    wp_for_post
    iframe
    iexists (W64 1), write
    simp only [show (W64 1) ≠ W64 0 by decide, ite_false]
    iframe
    ipureintro
    constructor <;> word
  · simp only [Hif, ite_false, _root_.decide_false]
    have hn : iv ≠ W64 0 := by rw [Hlength] at Hif; word
    simp only [hn, ite_false]
    wp_end
    all_goals iframe Ha Hsnapshot Hroots Hstate


end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
