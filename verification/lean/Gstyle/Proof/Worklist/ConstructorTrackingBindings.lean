module
public import Gstyle.Proof.Worklist.ConstructorTrackingRoots
public import Gstyle.Proof.Worklist.ConstructorTrackingPredicate
public import Gstyle.Proof.Worklist.ConstructorTrackingWrites
@[expose] public section
noncomputable section
set_option maxRecDepth 100000
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

/-- Finite scalar graph data. These are representation and bounds requirements,
with no assumed execution contract for constructor children. -/
structure ScalarTrackingData [FfiSyntax] (s : QuerySnapshot) : Prop where
  roots : ScalarTrackingRoots s
  symbols : ScalarTrackingSymbols s
  nodeBound : s.nodes.length < 2^32
  kindBound : s.kinds.length ≤ 2^43
  writeBounds : ∀ ref, validSymbolRef s ref →
    ∀ write ∈ pdg.rowContents s.accessRows (snapshotRecord s ref).Writes',
      trackingSymbolBound s.kinds.length write.Symbol'

section execution
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
variable [FuncUnfold pdg.listAt [pdg.Ref.ty] (pdg.listAt.impl pdg.Ref.ty)]
variable [FuncUnfold pdg.listAt [pdg.Access.ty] (pdg.listAt.impl pdg.Access.ty)]
variable [FuncUnfold pdg.listAt [pdg.Extension.ty] (pdg.listAt.impl pdg.Extension.ty)]
variable [FuncUnfold directBindingRead [] directBindingRead.impl]
variable [FuncUnfold operatorText [] operatorText.impl]
variable [FuncUnfold extension [] extension.impl]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

/-- Execute the complete production scalar tracker, including all node helpers,
definition writes and the final empty-queue guard. The receiver and all borrowed
view arrays retain their original values and permissions. -/
theorem wp_trackBindings_scalar (receiver : Loc) (view : graphView) (s : QuerySnapshot)
    (inverseLoc : Loc) (inverseMap : GMap GoString GoString) (vd fd kd rd : DFrac)
    (hgraph : view.graph' = s.view.graph') (hdata : ScalarTrackingData s) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ receiver ↦{vd} view ∗ constructorGraphOwn s ∗
      view.fields' ↦*{fd} s.fields ∗ view.kinds' ↦*{kd} s.kinds ∗ view.operands' ↦*{rd} s.operands ∗
      globalAddr inverse ↦□ inverseLoc ∗ inverseLoc ↦${DFrac.discard} inverseMap }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"trackBindings" #receiver)) (Val #()))
    {{ RET #(); receiver ↦{vd} view ∗ constructorGraphOwn s ∗
      view.fields' ↦*{fd} s.fields ∗ view.kinds' ↦*{kd} s.kinds ∗ view.operands' ↦*{rd} s.operands }} := by
  wp_start as ⟨Hview, #Hsnapshot, Hfields, Hkinds, Hoperands, #HinvAddr, #HinvMap⟩
  ihave #Hctx := Hsnapshot
  iunfold constructorGraphOwn at Hctx
  icases Hctx with ⟨Hgraph, Hnodes, _, _, _, _, _, HaccessRows, _, _, _⟩
  ihave %Hklen := ownSlice_len $$ Hkinds
  ihave %Hnlen := ownSlice_len $$ Hnodes
  try wp_auto
  have hkmax : 0 < typeSize GoSlice → sint.Z view.kinds'.len ≤ 2^48 / typeSize GoSlice := by
    intro _; simp only [go.typeSize_slice]; have h := hdata.kindBound; word
  wp_apply wp_slice_make2 (V := GoSlice) view.kinds'.len hkmax $$ []
    with %definitionsSlice ⟨Hdefinitions, -⟩
  · ipureintro; exact Hklen.2
  have hkl : sint.nat view.kinds'.len = s.kinds.length := by word
  rw [hkl]
  try wp_auto
  rw [hgraph]
  try wp_auto
  have hnmax : 0 < typeSize Bool → sint.Z s.graph.Nodes'.len ≤ 2^48 / typeSize Bool := by
    intro _; simp only [go.typeSize_bool]; have h := hdata.nodeBound; word
  wp_apply wp_slice_make2 (V := Bool) s.graph.Nodes'.len hnmax $$ []
    with %visitedSlice ⟨Hvisited, -⟩
  · ipureintro; exact Hnlen.2
  try wp_auto
  rw [hgraph]
  try wp_auto
  rename_i body_i_ptr
  irename : (body_i_ptr ↦ zero_val w64 : IProp GF) => body_i
  ihave HI : (∃ (iv bv : w64) (nv : pdg.Node) (rows : List GoSlice),
      "i" ∷ i_ptr ↦ iv ∗ "body_i" ∷ body_i_ptr ↦ bv ∗ "node" ∷ node_ptr ↦ nv ∗
      "v" ∷ v_ptr ↦ receiver ∗ "definitions" ∷ definitions_ptr ↦ definitionsSlice ∗
      "queue" ∷ queue_ptr ↦ (slice.nil : GoSlice) ∗ "visited" ∷ visited_ptr ↦ visitedSlice ∗
      "Hview" ∷ receiver ↦{vd} view ∗ "Hfields" ∷ view.fields' ↦*{fd} s.fields ∗
      "Hkinds" ∷ view.kinds' ↦*{kd} s.kinds ∗ "Hoperands" ∷ view.operands' ↦*{rd} s.operands ∗
      "Hdefinitions" ∷ definitionsSlice ↦* rows ∗ "Hrows" ∷ trackingDefinitionRowsOwn rows ∗
      "%Hbounds" ∷ ⌜0 ≤ sint.Z iv ∧ sint.Z iv ≤ sint.Z s.graph.Nodes'.len⌝ ∗
      "%Hlength" ∷ ⌜rows.length = s.kinds.length⌝ : IProp GF)
      $$ [i body_i node v definitions queue visited Hview Hfields Hkinds Hoperands Hdefinitions]
  · iexists (W64 0), (W64 0), (zero_val pdg.Node), (List.replicate s.kinds.length slice.nil)
    iframe
    isplitr
    · iapply trackingDefinitionRowsOwn_nil (GF := GF) s.kinds.length
    ipureintro
    exact ⟨by constructor <;> word, List.length_replicate ..⟩
  wp_for HI
  wp_if_destruct
  · try simp only [go.intoVal_inj.eq_iff, decide_eq_true_eq] at Hif
    have hi : sint.nat iv < s.nodes.length := by word
    list_elem s.nodes (sint.nat iv) as entry
    rw [ite_eq_left (by constructor <;> word :
      0 ≤ sint.Z iv ∧ sint.Z iv < sint.Z s.graph.Nodes'.len)]
    wp_apply wp_load_slice_index s.graph.Nodes' (sint.Z iv) s.nodes DFrac.discard entry
      (by word) $$ [Hnodes] with Hnodes
    · iframe Hnodes; ipureintro; exact Hentry_lookup
    try wp_auto
    let nodeRef : w32 := W32 (sint.Z (iv + W64 1))
    have href : 0 < uint.Z nodeRef ∧ uint.Z nodeRef ≤ s.nodes.length := by
      have hlimit := hdata.nodeBound; dsimp [nodeRef]; constructor <;> word
    have hindex : (uint.Z nodeRef - 1).toNat = sint.nat iv := by
      have hlimit := hdata.nodeBound; dsimp [nodeRef]; word
    have hnode : s.nodes[(uint.Z nodeRef - 1).toNat]? = some entry := by rw [hindex]; exact Hentry_lookup
    wp_apply wp_trackRoots_scalar_data receiver view s nodeRef visitedSlice queue_ptr vd rd
      hgraph hdata.roots href $$ [Hview Hsnapshot Hoperands] with ⟨Hview, #Hsnapshot, Hoperands⟩
    · iframe Hview Hsnapshot Hoperands
    try wp_auto
    wp_apply wp_trackCall_scalar receiver view s.graph s.nodes entry nodeRef visitedSlice queue_ptr
      vd DFrac.discard href hnode (hdata.roots.noCalls entry (List.mem_of_getElem? Hentry_lookup))
      $$ [Hview Hgraph Hnodes] with ⟨Hview, Hgraph, Hnodes⟩
    · rw [hgraph]; iframe Hview Hgraph Hnodes
    try wp_auto
    have hoi : (uint.Z nodeRef - 1).toNat < s.operands.length := by rw [hdata.roots.operandsLength]; omega
    let row := s.operands[(uint.Z nodeRef - 1).toNat]'hoi
    have hrow : s.operands[(uint.Z nodeRef - 1).toNat]? = some row := List.getElem?_eq_getElem hoi
    wp_apply wp_predicateDefinitions_scalar receiver view s definitionsSlice rows nodeRef row inverseLoc inverseMap
      vd fd kd rd hgraph hdata.symbols (Or.inr href)
      (by rw [hdata.roots.operandsLength]; exact href) hrow
      (hdata.roots.operandShape row (List.mem_of_getElem? hrow))
      (hdata.roots.operandRefs row (List.mem_of_getElem? hrow)) Hlength
      $$ [Hview Hsnapshot Hfields Hkinds Hoperands Hdefinitions Hrows HinvAddr HinvMap]
      with ⟨Hview, #Hsnapshot, Hfields, Hkinds, Hoperands, %predicateRows, Hdefinitions, Hrows, %HpredicateLength⟩
    · iframe Hview Hsnapshot Hfields Hkinds Hoperands Hdefinitions Hrows HinvAddr HinvMap
    try wp_auto
    rw [hgraph]
    wp_apply +noauto wp_constructorNodeAttributes_raw s nodeRef entry href hnode hdata.roots.outcomeBound
      $$ [Hsnapshot] with %attrs ⟨#Hsnapshot, %hattrs⟩
    · iframe Hsnapshot
    wp_pures
    have hwrites : attrs.Writes' = pdg.listRow s.accesses (snapshotRecord s nodeRef).Writes' :=
      congrArg pdg.Attributes.Writes' hattrs
    rw [hwrites]
    ihave Hwrites := pdg.readonlyRows_at_ref s.accesses s.accessRows (snapshotRecord s nodeRef).Writes' $$ HaccessRows
    ieval (change (_ : IProp GF) ⊢ WP (trackingWritesNormalizedRun definitions_ptr ref_ptr
      (pdg.listRow s.accesses (snapshotRecord s nodeRef).Writes')) {{ _ }})
    wp_apply wp_tracking_writes_normalized definitions_ptr ref_ptr definitionsSlice
      (pdg.listRow s.accesses (snapshotRecord s nodeRef).Writes') predicateRows
      (pdg.rowContents s.accessRows (snapshotRecord s nodeRef).Writes') nodeRef
      (by
        intro write hwrite
        have hw := hdata.writeBounds nodeRef (Or.inr href) write hwrite
        simpa only [trackingSymbolBound, HpredicateLength, Hlength] using hw)
      $$ [definitions ref Hdefinitions Hrows Hwrites]
      with ⟨definitions, ref, %out, Hdefinitions, Hrows, %HwriteLength⟩
    · iframe definitions ref Hdefinitions Hrows Hwrites
    wp_for_post
    iframe
    iexists (iv + W64 1), iv, entry, out
    iframe
    ipureintro
    exact ⟨by constructor <;> word, HwriteLength.trans (HpredicateLength.trans Hlength)⟩
  · try simp only [go.intoVal_inj.eq_iff, decide_eq_true_eq] at Hif
    try simp only [go.intoVal_inj.eq_iff, decide_eq_true_eq, decide_eq_false Hif, Bool.false_eq_true, ite_false, ite_true]
    try wp_auto
    wp_for
    try simp only [go.intoVal_inj.eq_iff, Bool.false_eq_true, _root_.decide_false, _root_.decide_true, ite_false, ite_true]
    try wp_auto
    wp_end
    all_goals iframe Hview Hsnapshot Hfields Hkinds Hoperands

end execution
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
