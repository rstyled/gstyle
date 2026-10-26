module
public import Gstyle.Proof.Worklist.ConstructorTrackingSymbols
public import Gstyle.Proof.Worklist.ConstructorTrackingDefinitions
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
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
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

theorem wp_trackingOperator_raw (s : QuerySnapshot) (fact : pdg.Fact) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ constructorGraphOwn s }}
      (App (App (Val (@! operatorText)) (Val #s.graph.Tables')) (Val #fact))
    {{ RET #(decodedFact s fact); constructorGraphOwn s }} := by
  wp_start_folded as #Hsnapshot
  unfold decodedFact
  by_cases hs : fact.Status' = 1#8
  · rw [ite_eq_left hs]
    ihave #Hctx := Hsnapshot
    iunfold constructorGraphOwn at Hctx
    icases Hctx with ⟨_, _, _, _, _, _, _, _, _, Htable, Hstrings⟩
    wp_apply wp_operatorText_known s.graph.Tables' fact s.table s.strings DFrac.discard hs
      $$ [Htable Hstrings] with ⟨Htable, Hstrings⟩
    · iframe Htable Hstrings
    wp_end
    all_goals iexact Hsnapshot
  · rw [ite_eq_right hs]
    wp_apply wp_operatorText_unavailable s.graph.Tables' fact hs
    wp_end
    all_goals iexact Hsnapshot

/-- Execute the real inverse-map lookup and all comparison operand appends.
The map is borrowed as supplied; comparison entries are never removed or masked. -/
theorem wp_predicateDefinitions_scalar (receiver : Loc) (view : graphView) (s : QuerySnapshot)
    (definitions : GoSlice) (initial : List GoSlice) (ref : w32)
    (row : GoArray w32 2) (inverseLoc : Loc) (inverseMap : GMap GoString GoString)
    (vd fd kd rd : DFrac) (hgraph : view.graph' = s.view.graph')
    (hdata : ScalarTrackingSymbols s) (hvalid : validSymbolRef s ref)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row) (harity : row.arr.length = 2)
    (hchildren : ∀ child ∈ row.arr, validSymbolRef s child)
    (hcount : initial.length = s.kinds.length) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ receiver ↦{vd} view ∗ constructorGraphOwn s ∗
      view.fields' ↦*{fd} s.fields ∗ view.kinds' ↦*{kd} s.kinds ∗ view.operands' ↦*{rd} s.operands ∗
      definitions ↦* initial ∗ trackingDefinitionRowsOwn initial ∗
      globalAddr inverse ↦□ inverseLoc ∗ inverseLoc ↦${DFrac.discard} inverseMap }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"predicateDefinitions" #receiver))
        (Val #ref)) (Val #definitions))
    {{ RET #(); receiver ↦{vd} view ∗ constructorGraphOwn s ∗
      view.fields' ↦*{fd} s.fields ∗ view.kinds' ↦*{kd} s.kinds ∗ view.operands' ↦*{rd} s.operands ∗
      ∃ out : List GoSlice, definitions ↦* out ∗ trackingDefinitionRowsOwn out ∗
        ⌜out.length = initial.length⌝ }} := by
  wp_start as ⟨Hview, #Hsnapshot, Hfields, Hkinds, Hoperands, Hdefinitions, Hrows, #HinvAddr, #HinvMap⟩
  ihave #Hctx := Hsnapshot
  iunfold constructorGraphOwn at Hctx
  icases Hctx with ⟨Hgraph, _, _, _, _, _, _, _, _, _, _⟩
  ihave %Holen := ownSlice_len $$ Hoperands
  ihave %Hdlen := ownSlice_len $$ Hdefinitions
  try wp_auto
  rw [hgraph]
  try wp_auto
  wp_apply wp_graphView_attrs_raw receiver view s ref vd hgraph hvalid hdata.outcomeBound
    $$ [Hview Hsnapshot] with %attrs ⟨Hview, Hsnapshot, %hattrs⟩
  · iframe Hview Hsnapshot
  try wp_auto
  have hop : attrs.Operator' = (snapshotRecord s ref).Operator' := congrArg pdg.Attributes.Operator' hattrs
  rw [hop]
  wp_apply wp_trackingOperator_raw s (snapshotRecord s ref).Operator'
    $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  try wp_auto
  wp_apply wp_map_lookup2 go.string go.string inverseLoc inverseMap
    (decodedFact s (snapshotRecord s ref).Operator') DFrac.discard $$ [HinvMap] with HinvMap
  · iframe HinvMap
  try wp_auto
  cases hc : (inverseMap !! decodedFact s (snapshotRecord s ref).Operator').isSome with
  | false =>
    try wp_auto
    wp_end
    all_goals iframe Hview Hsnapshot Hfields Hkinds Hoperands
    all_goals iexists initial; iframe Hdefinitions Hrows; ipureintro; rfl
  | true =>
    try wp_auto
    have Hi : sint.Z (W64 (uint.Z (ref - W32 1))) = uint.Z ref - 1 := by
      obtain ⟨_, _⟩ := href; word
    rw [Hi]
    rw [ite_eq_left (by obtain ⟨_, _⟩ := href; constructor <;> word :
      0 ≤ uint.Z ref - 1 ∧ uint.Z ref - 1 < sint.Z view.operands'.len)]
    wp_apply wp_load_slice_index view.operands' (uint.Z ref - 1) s.operands rd row
      (by omega) $$ [Hoperands] with Hoperands
    · iframe; ipureintro; exact hrow
    try wp_auto
    ihave HI : (∃ (iv : w64) (ov : w32) (rows : List GoSlice),
        "i" ∷ i_ptr ↦ iv ∗ "operand" ∷ operand_ptr ↦ ov ∗
        "v" ∷ v_ptr ↦ receiver ∗ "ref" ∷ ref_ptr ↦ ref ∗
        "definitions" ∷ definitions_ptr ↦ definitions ∗
        "Hview" ∷ receiver ↦{vd} view ∗
        "Hfields" ∷ view.fields' ↦*{fd} s.fields ∗ "Hkinds" ∷ view.kinds' ↦*{kd} s.kinds ∗
        "Hoperands" ∷ view.operands' ↦*{rd} s.operands ∗
        "Hdefinitions" ∷ definitions ↦* rows ∗ "Hrows" ∷ trackingDefinitionRowsOwn rows ∗
        "%Hbounds" ∷ ⌜0 ≤ sint.Z iv ∧ sint.Z iv ≤ 2⌝ ∗
        "%Hlength" ∷ ⌜rows.length = initial.length⌝ : IProp GF)
        $$ [i operand v ref definitions Hview Hfields Hkinds Hoperands Hdefinitions Hrows]
    · iexists (W64 0), (zero_val w32), initial
      iframe
      ipureintro; exact ⟨by decide, rfl⟩
    wp_for HI
    wp_if_destruct
    · have hi : sint.nat iv < row.arr.length := by rw [harity]; word
      list_elem row.arr (sint.nat iv) as child
      rw [ite_eq_left Hbounds.1, Hchild_lookup]
      try wp_auto
      wp_apply wp_trackingSymbol_scalar receiver view s child vd fd kd hgraph hdata
        (hchildren child (List.mem_of_getElem? Hchild_lookup))
        $$ [Hview Hsnapshot Hfields Hkinds] with %symbol ⟨Hview, Hsnapshot, Hfields, Hkinds, %Hsymbol⟩
      · iframe Hview Hsnapshot Hfields Hkinds
      try wp_auto
      by_cases hz : symbol = W32 0
      · rw [decide_eq_true hz]
        try wp_auto
        wp_for_post
        iframe
        iexists (iv + W64 1), child, rows
        iframe
        ipureintro; exact ⟨by constructor <;> word, Hlength⟩
      · rw [decide_eq_false hz]
        try wp_auto
        have hb : 0 < uint.Z symbol ∧ uint.Z symbol ≤ rows.length := by
          have hp := Hsymbol.resolve_left hz
          rw [Hlength, hcount]
          exact hp
        have Hsi : sint.Z (W64 (uint.Z (symbol - W32 1))) = uint.Z symbol - 1 := by
          obtain ⟨_, _⟩ := hb; word
        rw [Hsi]
        rw [ite_eq_left (by obtain ⟨_, _⟩ := hb; constructor <;> word :
          0 ≤ uint.Z symbol - 1 ∧ uint.Z symbol - 1 < sint.Z definitions.len)]
        have hj : (uint.Z symbol - 1).toNat < rows.length := by omega
        let current := rows[(uint.Z symbol - 1).toNat]'hj
        have hcurrent : rows[(uint.Z symbol - 1).toNat]? = some current := List.getElem?_eq_getElem hj
        wp_apply wp_load_slice_index definitions (uint.Z symbol - 1) rows (DFrac.own 1) current
          (by omega) $$ [Hdefinitions] with Hdefinitions
        · iframe; ipureintro; exact hcurrent
        icases trackingDefinitionRowsOwn_acc rows (uint.Z symbol - 1).toNat current hcurrent
          $$ Hrows with ⟨Hcurrent, Hrestore⟩
        iunfold trackingDefinitionOwn at Hcurrent
        icases Hcurrent with ⟨%values, Hvalues, Hcap⟩
        try wp_auto
        wp_apply wp_slice_literal (V := w32) [ref]
        isplitr
        · ipureintro; rfl
        iintro %one ⟨Hone, -⟩
        try wp_auto
        wp_apply wp_slice_append $$ [Hvalues Hcap Hone] with %next ⟨Hvalues, Hcap, -⟩
        · iframe
        try wp_auto
        rw [Hsi]
        rw [ite_eq_left (by obtain ⟨_, _⟩ := hb; constructor <;> word :
          0 ≤ uint.Z symbol - 1 ∧ uint.Z symbol - 1 < sint.Z definitions.len)]
        wp_apply wp_store_slice_index definitions (uint.Z symbol - 1) rows next
          $$ [Hdefinitions] with Hdefinitions
        · iframe; ipureintro; constructor <;> omega
        ihave Hnext : (trackingDefinitionOwn next : IProp GF) $$ [Hvalues Hcap]
        · unfold trackingDefinitionOwn; iexists (values ++ [ref]); iframe
        ihave Hrows := Hrestore $$ %next Hnext
        wp_for_post
        iframe
        iexists (iv + W64 1), child, (rows.set (uint.Z symbol - 1).toNat next)
        iframe
        ipureintro
        exact ⟨by constructor <;> word, (List.length_set ..).trans Hlength⟩
    · wp_end
      all_goals iframe Hview Hsnapshot Hfields Hkinds Hoperands
      all_goals iexists rows; iframe Hdefinitions Hrows; ipureintro; exact Hlength

end execution
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
