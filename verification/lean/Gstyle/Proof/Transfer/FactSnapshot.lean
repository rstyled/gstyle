module
public import Gstyle.Proof.Transfer.QuerySnapshot
public import Gstyle.Proof.Transfer.Facts
public import Gstyle.Proof.Transfer.Operator
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

def snapshotFact [FfiSyntax] (s : QuerySnapshot) (ref : w32) (name : GoString) : GoString :=
  if ref = W32 0 then go!"" else
  match s.nodes[(uint.Z ref - 1).toNat]? with
  | none => go!""
  | some node =>
    if node.Attributes' = W32 0 then go!"" else
    match s.attributes[(uint.Z node.Attributes' - 1).toNat]? with
    | none => go!""
    | some a => extensionValue s.strings name (pdg.rowContents s.extensionRows a.Extensions')

def snapshotRecord [FfiSyntax] (s : QuerySnapshot) (ref : w32) : pdg.AttributeSet :=
  if ref = W32 0 then zero_val pdg.AttributeSet else
  match s.nodes[(uint.Z ref - 1).toNat]? with
  | none => zero_val pdg.AttributeSet
  | some node =>
    if node.Attributes' = W32 0 then zero_val pdg.AttributeSet else
    s.attributes[(uint.Z node.Attributes' - 1).toNat]?.getD (zero_val pdg.AttributeSet)

def decodedFact [FfiSyntax] (s : QuerySnapshot) (fact : pdg.Fact) : GoString :=
  if fact.Status' = 1#8 then pdg.tableText s.strings fact.Value' else go!""

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

theorem wp_fact_snapshot (receiver : Loc) (s : QuerySnapshot) (ref : w32) (name : GoString)
    (hvalid : validSymbolRef s ref)
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ querySnapshotOwn receiver s }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"fact" #receiver)) (Val #ref)) (Val #name))
    {{ RET #(snapshotFact s ref name); querySnapshotOwn receiver s }} := by
  wp_start_folded as #Hsnapshot
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, Hgraph, Hfields, Hrows, Hkinds, Hunstable, Htracked, Hnodes, Hrecords,
    Hreferences, Haccesses, Hextensions, HrefRows, HaccessRows, HextensionRows, Htable, Hstrings⟩
  by_cases hz : ref = W32 0
  · subst ref
    simp only [snapshotFact, ite_true]
    wp_apply wp_fact_zero receiver s.view name DFrac.discard $$ [Hview] with Hview
    · iframe Hview
    wp_end
    all_goals iexact Hsnapshot
  · have href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.nodes.length := hvalid.resolve_left hz
    have hni : (uint.Z ref - 1).toNat < s.nodes.length := by omega
    let node := s.nodes[(uint.Z ref - 1).toNat]'hni
    have hnode : s.nodes[(uint.Z ref - 1).toNat]? = some node := List.getElem?_eq_getElem hni
    simp only [snapshotFact, hz, ite_false, hnode]
    by_cases ha : 0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ s.attributes.length
    · have hai : (uint.Z node.Attributes' - 1).toNat < s.attributes.length := by omega
      let a := s.attributes[(uint.Z node.Attributes' - 1).toNat]'hai
      have hrecord : s.attributes[(uint.Z node.Attributes' - 1).toNat]? = some a := List.getElem?_eq_getElem hai
      have ha0 : node.Attributes' ≠ W32 0 := by obtain ⟨_, _⟩ := ha; word
      rw [ite_eq_right ha0, hrecord]
      ihave Hvalues := pdg.readonlyRows_at_ref s.references s.referenceRows a.Outcomes' $$ HrefRows
      ihave Hentries := pdg.readonlyRows_at_ref s.extensions s.extensionRows a.Extensions' $$ HextensionRows
      wp_apply wp_fact_present receiver s.view s.graph s.nodes node s.attributes a s.references s.accesses s.extensions
        (pdg.rowContents s.referenceRows a.Outcomes') s.table s.strings (pdg.rowContents s.extensionRows a.Extensions')
        ref name DFrac.discard DFrac.discard DFrac.discard DFrac.discard href hnode ha hrecord (halloc a (List.mem_of_getElem? hrecord))
        $$ [Hview Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues Htable Hstrings Hentries]
        with ⟨Hview, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Htable, Hstrings, Hentries⟩
      · iframe Hview Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues Htable Hstrings Hentries
      wp_end
      all_goals iexact Hsnapshot
    · have hresult : (if node.Attributes' = W32 0 then go!"" else
          match s.attributes[(uint.Z node.Attributes' - 1).toNat]? with
          | none => go!""
          | some a => extensionValue s.strings name (pdg.rowContents s.extensionRows a.Extensions')) = go!"" := by
        by_cases ha0 : node.Attributes' = W32 0
        · rw [ite_eq_left ha0]
        · rw [ite_eq_right ha0]
          have hpos : 0 < uint.Z node.Attributes' := by word
          have hnone : s.attributes[(uint.Z node.Attributes' - 1).toNat]? = none := List.getElem?_eq_none (by omega)
          rw [hnone]
      rw [hresult]
      wp_apply wp_fact_absent receiver s.view s.graph s.nodes node s.attributes ref name DFrac.discard DFrac.discard href hnode ha
        $$ [Hview Hgraph Hnodes Hrecords] with ⟨Hview, Hgraph, Hnodes, Hrecords⟩
      · iframe Hview Hgraph Hnodes Hrecords
      wp_end
      all_goals iexact Hsnapshot

theorem wp_attrs_snapshot_absent (receiver : Loc) (s : QuerySnapshot) (ref : w32) (node : pdg.Node)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.nodes.length)
    (hnode : s.nodes[(uint.Z ref - 1).toNat]? = some node)
    (ha : ¬ (0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ s.attributes.length)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ querySnapshotOwn receiver s }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"attrs" #receiver)) (Val #ref))
    {{ RET #(zero_val pdg.Attributes); querySnapshotOwn receiver s }} := by
  wp_start_folded as #Hsnapshot
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, Hgraph, _, _, _, _, _, Hnodes, Hrecords, _, _, _, _, _, _, _, _⟩
  wp_apply wp_attrs_absent receiver s.view s.graph s.nodes node s.attributes ref DFrac.discard DFrac.discard href hnode ha
    $$ [Hview Hgraph Hnodes Hrecords] with ⟨Hview, Hgraph, Hnodes, Hrecords⟩
  · iframe Hview Hgraph Hnodes Hrecords
  wp_end
  all_goals iexact Hsnapshot

theorem wp_attrs_snapshot_full (receiver : Loc) (s : QuerySnapshot) (ref : w32)
    (hvalid : validSymbolRef s ref)
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ querySnapshotOwn receiver s }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"attrs" #receiver)) (Val #ref))
    {{ (attrs : pdg.Attributes), RET #attrs; querySnapshotOwn receiver s ∗
      ⌜attrs = pdg.expandedAttributes (snapshotRecord s ref) s.references s.accesses s.extensions attrs.Outcomes'⌝ }} := by
  wp_start_folded as #Hsnapshot
  by_cases hz : ref = W32 0
  · subst ref
    simp only [snapshotRecord, ite_true]
    wp_apply wp_attrs_zero receiver
    wp_end
    all_goals iframe Hsnapshot
    all_goals ipureintro; rfl
  · have href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.nodes.length := hvalid.resolve_left hz
    have hni : (uint.Z ref - 1).toNat < s.nodes.length := by omega
    let node := s.nodes[(uint.Z ref - 1).toNat]'hni
    have hnode : s.nodes[(uint.Z ref - 1).toNat]? = some node := List.getElem?_eq_getElem hni
    by_cases ha : 0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ s.attributes.length
    · have hai : (uint.Z node.Attributes' - 1).toNat < s.attributes.length := by omega
      let a := s.attributes[(uint.Z node.Attributes' - 1).toNat]'hai
      have hrecord : s.attributes[(uint.Z node.Attributes' - 1).toNat]? = some a := List.getElem?_eq_getElem hai
      have ha0 : node.Attributes' ≠ W32 0 := by obtain ⟨_, _⟩ := ha; word
      simp only [snapshotRecord, hz, ite_false, hnode, ha0, hrecord, Option.getD_some]
      ihave #Hctx := Hsnapshot
      iunfold querySnapshotOwn at Hctx
      icases Hctx with ⟨Hview, Hgraph, _, _, _, _, _, Hnodes, Hrecords,
        Hreferences, Haccesses, Hextensions, HrefRows, _, _, _, _⟩
      ihave Hvalues := pdg.readonlyRows_at_ref s.references s.referenceRows a.Outcomes' $$ HrefRows
      wp_apply wp_attrs_present receiver s.view s.graph s.nodes node s.attributes a s.references s.accesses s.extensions
        (pdg.rowContents s.referenceRows a.Outcomes') ref DFrac.discard DFrac.discard href hnode ha hrecord (halloc a (List.mem_of_getElem? hrecord))
        $$ [Hview Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues]
        with %out ⟨Hview, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Hout, %Hempty⟩
      · iframe Hview Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues
      wp_end
      all_goals iframe Hsnapshot
      all_goals ipureintro; rfl
    · have hm : snapshotRecord s ref = zero_val pdg.AttributeSet := by
        simp only [snapshotRecord, hz, ite_false, hnode]
        by_cases ha0 : node.Attributes' = W32 0
        · rw [ite_eq_left ha0]
        · rw [ite_eq_right ha0]
          have hpos : 0 < uint.Z node.Attributes' := by word
          have hnone : s.attributes[(uint.Z node.Attributes' - 1).toNat]? = none := List.getElem?_eq_none (by omega)
          rw [hnone]
          rfl
      rw [hm]
      wp_apply wp_attrs_snapshot_absent receiver s ref node href hnode ha $$ [Hsnapshot] with Hsnapshot
      · iframe Hsnapshot
      wp_end
      all_goals iframe Hsnapshot
      all_goals ipureintro; rfl

theorem wp_attrs_snapshot (receiver : Loc) (s : QuerySnapshot) (ref : w32)
    (hvalid : validSymbolRef s ref)
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ querySnapshotOwn receiver s }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"attrs" #receiver)) (Val #ref))
    {{ (attrs : pdg.Attributes), RET #attrs; querySnapshotOwn receiver s ∗
      ⌜attrs.Operator' = (snapshotRecord s ref).Operator' ∧ attrs.Callee' = (snapshotRecord s ref).Callee'⌝ }} := by
  wp_start_folded as #Hsnapshot
  wp_apply wp_attrs_snapshot_full receiver s ref hvalid halloc
      $$ [Hsnapshot] with %attrs ⟨Hsnapshot, %hattrs⟩
  · iframe Hsnapshot
  iapply HΦ
  iframe Hsnapshot
  ipureintro
  exact ⟨congrArg pdg.Attributes.Operator' hattrs, congrArg pdg.Attributes.Callee' hattrs⟩

theorem wp_operatorText_snapshot (receiver : Loc) (s : QuerySnapshot) (fact : pdg.Fact) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ querySnapshotOwn receiver s }}
      (App (App (Val (@! operatorText)) (Val #s.graph.Tables')) (Val #fact))
    {{ RET #(decodedFact s fact); querySnapshotOwn receiver s }} := by
  wp_start_folded as #Hsnapshot
  unfold decodedFact
  by_cases hs : fact.Status' = 1#8
  · rw [ite_eq_left hs]
    ihave #Hctx := Hsnapshot
    iunfold querySnapshotOwn at Hctx
    icases Hctx with ⟨_, _, _, _, _, _, _, _, _, _, _, _, _, _, _, Htable, Hstrings⟩
    wp_apply wp_operatorText_known s.graph.Tables' fact s.table s.strings DFrac.discard hs
      $$ [Htable Hstrings] with ⟨Htable, Hstrings⟩
    · iframe Htable Hstrings
    wp_end
    all_goals iexact Hsnapshot
  · rw [ite_eq_right hs]
    wp_apply wp_operatorText_unavailable s.graph.Tables' fact hs
    wp_end
    all_goals iexact Hsnapshot

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
