module
public import Gstyle.Proof.Transfer.FactSnapshot
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
variable [pdg.Tables.TypeAssumptions] [pdg.Extension.TypeAssumptions]

/-- Readonly graph resources, independent of the constructor's mutable receiver and slices. -/
def constructorGraphOwn (s : QuerySnapshot) : IProp GF := iprop(
  s.view.graph' ↦□ s.graph ∗
  s.graph.Nodes' ↦*□ s.nodes ∗ s.graph.Attributes' ↦*□ s.attributes ∗
  s.graph.AttributeTables'.References' ↦*□ s.references ∗
  s.graph.AttributeTables'.Accesses' ↦*□ s.accesses ∗
  s.graph.AttributeTables'.Extensions' ↦*□ s.extensions ∗
  pdg.readonlyRows s.references s.referenceRows ∗ pdg.readonlyRows s.accesses s.accessRows ∗
  pdg.readonlyRows s.extensions s.extensionRows ∗
  s.graph.Tables' ↦□ s.table ∗ s.table.Strings' ↦*□ s.strings)

instance constructorGraphOwn_persistent (s : QuerySnapshot) :
    Persistent (constructorGraphOwn (GF := GF) s) := by
  unfold constructorGraphOwn
  infer_instance

variable [FuncUnfold pdg.listAt [pdg.Ref.ty] (pdg.listAt.impl pdg.Ref.ty)]
variable [FuncUnfold pdg.listAt [pdg.Access.ty] (pdg.listAt.impl pdg.Access.ty)]
variable [FuncUnfold pdg.listAt [pdg.Extension.ty] (pdg.listAt.impl pdg.Extension.ty)]
variable [FuncUnfold extension [] extension.impl]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

/-- Read a snapshot fact while retaining arbitrary receiver ownership. -/
theorem wp_graphView_fact_raw (receiver : Loc) (view : graphView) (s : QuerySnapshot)
    (ref : w32) (name : GoString) (vd : DFrac)
    (hgraph : view.graph' = s.view.graph') (hvalid : validSymbolRef s ref)
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ receiver ↦{vd} view ∗ constructorGraphOwn s }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"fact" #receiver)) (Val #ref)) (Val #name))
    {{ RET #(snapshotFact s ref name); receiver ↦{vd} view ∗ constructorGraphOwn s }} := by
  by_cases hz : ref = W32 0
  · subst ref
    simp only [snapshotFact, ite_true]
    wp_start_folded as ⟨Hview, #Hsnapshot⟩
    wp_apply wp_fact_zero receiver view name vd $$ [Hview] with Hview
    · iframe Hview
    wp_end
    all_goals iframe Hview Hsnapshot
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
      wp_start_folded as ⟨Hview, #Hsnapshot⟩
      ihave #Hctx := Hsnapshot
      iunfold constructorGraphOwn at Hctx
      icases Hctx with ⟨Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions,
        HrefRows, _, HextensionRows, Htable, Hstrings⟩
      ihave Hvalues := pdg.readonlyRows_at_ref s.references s.referenceRows a.Outcomes' $$ HrefRows
      ihave Hentries := pdg.readonlyRows_at_ref s.extensions s.extensionRows a.Extensions' $$ HextensionRows
      wp_apply wp_fact_present receiver view s.graph s.nodes node s.attributes a s.references s.accesses s.extensions
        (pdg.rowContents s.referenceRows a.Outcomes') s.table s.strings (pdg.rowContents s.extensionRows a.Extensions')
        ref name vd DFrac.discard DFrac.discard DFrac.discard href hnode ha hrecord (halloc a (List.mem_of_getElem? hrecord))
        $$ [Hview Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues Htable Hstrings Hentries]
        with ⟨Hview, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Htable, Hstrings, Hentries⟩
      · rw [hgraph]
        iframe Hview Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues Htable Hstrings Hentries
      wp_end
      all_goals iframe Hview Hsnapshot
    · have hzero : node.Attributes' = W32 0 ∨ s.attributes[(uint.Z node.Attributes' - 1).toNat]? = none := by
        by_cases ha0 : node.Attributes' = W32 0
        · exact Or.inl ha0
        · right
          have hpos : 0 < uint.Z node.Attributes' := by word
          exact List.getElem?_eq_none (by omega)
      rcases hzero with ha0 | hnone
      all_goals first | simp only [ha0, ite_true] | simp only [hnone, ite_self]
      all_goals
        wp_start_folded as ⟨Hview, #Hsnapshot⟩
        ihave #Hctx := Hsnapshot
        iunfold constructorGraphOwn at Hctx
        icases Hctx with ⟨Hgraph, Hnodes, Hrecords, _, _, _, _, _, _, _, _⟩
        wp_apply wp_fact_absent receiver view s.graph s.nodes node s.attributes ref name vd DFrac.discard href hnode ha
          $$ [Hview Hgraph Hnodes Hrecords] with ⟨Hview, Hgraph, Hnodes, Hrecords⟩
        · rw [hgraph]
          iframe Hview Hgraph Hnodes Hrecords
        wp_end
        all_goals iframe Hview Hsnapshot

omit [pdg.Tables.TypeAssumptions] [pdg.Extension.TypeAssumptions]
  [FuncUnfold extension [] extension.impl] in
/-- Read expanded attributes while retaining arbitrary receiver ownership. -/
theorem wp_graphView_attrs_raw (receiver : Loc) (view : graphView) (s : QuerySnapshot)
    (ref : w32) (vd : DFrac)
    (hgraph : view.graph' = s.view.graph') (hvalid : validSymbolRef s ref)
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ receiver ↦{vd} view ∗ constructorGraphOwn s }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"attrs" #receiver)) (Val #ref))
    {{ (attrs : pdg.Attributes), RET #attrs; receiver ↦{vd} view ∗ constructorGraphOwn s ∗
      ⌜attrs = pdg.expandedAttributes (snapshotRecord s ref) s.references s.accesses s.extensions attrs.Outcomes'⌝ }} := by
  wp_start_folded as ⟨Hview, #Hsnapshot⟩
  by_cases hz : ref = W32 0
  · subst ref
    isimp only [snapshotRecord, ite_true] at HΦ
    wp_apply wp_attrs_zero receiver
    wp_end
    all_goals iframe Hview Hsnapshot
    all_goals ipureintro; rfl
  · have href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.nodes.length := hvalid.resolve_left hz
    have hni : (uint.Z ref - 1).toNat < s.nodes.length := by omega
    let node := s.nodes[(uint.Z ref - 1).toNat]'hni
    have hnode : s.nodes[(uint.Z ref - 1).toNat]? = some node := List.getElem?_eq_getElem hni
    ihave #Hctx := Hsnapshot
    iunfold constructorGraphOwn at Hctx
    icases Hctx with ⟨Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions,
      HrefRows, _, _, _, _⟩
    by_cases ha : 0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ s.attributes.length
    · have hai : (uint.Z node.Attributes' - 1).toNat < s.attributes.length := by omega
      let a := s.attributes[(uint.Z node.Attributes' - 1).toNat]'hai
      have hrecord : s.attributes[(uint.Z node.Attributes' - 1).toNat]? = some a := List.getElem?_eq_getElem hai
      have ha0 : node.Attributes' ≠ W32 0 := by obtain ⟨_, _⟩ := ha; word
      isimp only [snapshotRecord, hz, ite_false, hnode, ha0, hrecord, Option.getD_some] at HΦ
      ihave Hvalues := pdg.readonlyRows_at_ref s.references s.referenceRows a.Outcomes' $$ HrefRows
      wp_apply wp_attrs_present receiver view s.graph s.nodes node s.attributes a s.references s.accesses s.extensions
        (pdg.rowContents s.referenceRows a.Outcomes') ref vd DFrac.discard href hnode ha hrecord
        (halloc a (List.mem_of_getElem? hrecord))
        $$ [Hview Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues]
        with %out ⟨Hview, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Hout, %Hempty⟩
      · rw [hgraph]
        iframe Hview Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues
      wp_end
      all_goals iframe Hview Hsnapshot
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
      isimp only [hm] at HΦ
      wp_apply wp_attrs_absent receiver view s.graph s.nodes node s.attributes ref vd DFrac.discard href hnode ha
        $$ [Hview Hgraph Hnodes Hrecords] with ⟨Hview, Hgraph, Hnodes, Hrecords⟩
      · rw [hgraph]
        iframe Hview Hgraph Hnodes Hrecords
      wp_end
      all_goals iframe Hview Hsnapshot
      all_goals ipureintro; rfl

omit [graphView.TypeAssumptions] [pdg.Tables.TypeAssumptions] [pdg.Extension.TypeAssumptions]
  [FuncUnfold extension [] extension.impl] in
/-- Read a known graph node directly, preserving the constructor's readonly graph resources. -/
theorem wp_constructorNodeAttributes_raw (s : QuerySnapshot) (ref : w32) (node : pdg.Node)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.nodes.length)
    (hnode : s.nodes[(uint.Z ref - 1).toNat]? = some node)
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ constructorGraphOwn s }}
      (App (Val #(methods (go.PointerType pdg.Graph.ty) go!"NodeAttributes" #s.view.graph')) (Val #node))
    {{ (attrs : pdg.Attributes), RET #attrs; constructorGraphOwn s ∗
      ⌜attrs = pdg.expandedAttributes (snapshotRecord s ref) s.references s.accesses s.extensions attrs.Outcomes'⌝ }} := by
  wp_start_folded as #Hsnapshot
  have hz : ref ≠ W32 0 := by obtain ⟨_, _⟩ := href; word
  ihave #Hctx := Hsnapshot
  iunfold constructorGraphOwn at Hctx
  icases Hctx with ⟨Hgraph, _, Hrecords, Hreferences, Haccesses, Hextensions,
    HrefRows, _, _, _, _⟩
  by_cases ha : 0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ s.attributes.length
  · have hai : (uint.Z node.Attributes' - 1).toNat < s.attributes.length := by omega
    let a := s.attributes[(uint.Z node.Attributes' - 1).toNat]'hai
    have hrecord : s.attributes[(uint.Z node.Attributes' - 1).toNat]? = some a := List.getElem?_eq_getElem hai
    have ha0 : node.Attributes' ≠ W32 0 := by obtain ⟨_, _⟩ := ha; word
    isimp only [snapshotRecord, hz, ite_false, hnode, ha0, hrecord, Option.getD_some] at HΦ
    ihave Hvalues := pdg.readonlyRows_at_ref s.references s.referenceRows a.Outcomes' $$ HrefRows
    wp_apply pdg.wp_NodeAttributes_present s.view.graph' s.graph node s.attributes a
      s.references s.accesses s.extensions (pdg.rowContents s.referenceRows a.Outcomes')
      DFrac.discard ha hrecord (halloc a (List.mem_of_getElem? hrecord))
      $$ [Hgraph Hrecords Hreferences Haccesses Hextensions Hvalues]
      with %out ⟨Hgraph, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Hout, %Hempty⟩
    · iframe Hgraph Hrecords Hreferences Haccesses Hextensions Hvalues
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
    isimp only [hm] at HΦ
    wp_apply pdg.wp_NodeAttributes_absent s.view.graph' s.graph node s.attributes DFrac.discard ha
      $$ [Hgraph Hrecords] with ⟨Hgraph, Hrecords⟩
    · iframe Hgraph Hrecords
    wp_end
    all_goals iframe Hsnapshot
    all_goals ipureintro; rfl

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
