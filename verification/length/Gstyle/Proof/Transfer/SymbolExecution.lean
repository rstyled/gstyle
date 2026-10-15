module
public import Gstyle.Proof.Transfer.SymbolModel
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


theorem wp_symbol_snapshot_zero (receiver : Loc) (s : QuerySnapshot) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ querySnapshotOwn receiver s }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"symbol" #receiver)) (Val #(W32 0)))
    {{ RET #(W32 0); querySnapshotOwn receiver s }} := by
  wp_start_folded as Hsnapshot
  wp_apply wp_symbol_zero receiver
  wp_end

theorem wp_symbol_snapshot_cached (receiver : Loc) (s : QuerySnapshot) (ref value : w32)
    (hf : 0 < uint.Z ref ∧ uint.Z ref ≤ s.fields.length)
    (hcache : s.fields[(uint.Z ref - 1).toNat]? = some value) (hv : value ≠ W32 0) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ querySnapshotOwn receiver s }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"symbol" #receiver)) (Val #ref))
    {{ RET #value; querySnapshotOwn receiver s }} := by
  wp_start_folded as #Hsnapshot
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, _, Hfields, _, _, _, _, _, _, _, _, _, _, _, _, _, _⟩
  wp_apply wp_symbol_cached receiver s.view s.fields ref value DFrac.discard DFrac.discard hf hcache hv
    $$ [Hview Hfields] with ⟨Hview, Hfields⟩
  · iframe Hview Hfields
  wp_end
  all_goals iexact Hsnapshot

theorem wp_symbol_snapshot_absent (receiver : Loc) (s : QuerySnapshot) (ref : w32) (node : pdg.Node)
    (hf : 0 < uint.Z ref ∧ uint.Z ref ≤ s.fields.length)
    (hfield : s.fields[(uint.Z ref - 1).toNat]? = some (W32 0))
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.nodes.length)
    (hnode : s.nodes[(uint.Z ref - 1).toNat]? = some node)
    (ha : ¬ (0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ s.attributes.length)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ querySnapshotOwn receiver s }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"symbol" #receiver)) (Val #ref))
    {{ RET #(W32 0); querySnapshotOwn receiver s }} := by
  wp_start_folded as #Hsnapshot
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, Hgraph, Hfields, _, _, _, _, Hnodes, Hrecords, _, _, _, _, _, _, _, _⟩
  wp_apply wp_symbol_absent_attributes receiver s.view s.graph s.fields s.nodes node s.attributes ref
    DFrac.discard DFrac.discard DFrac.discard hf hfield href hnode ha
    $$ [Hview Hfields Hgraph Hnodes Hrecords] with ⟨Hview, Hfields, Hgraph, Hnodes, Hrecords⟩
  · iframe Hview Hfields Hgraph Hnodes Hrecords
  wp_end
  all_goals iexact Hsnapshot

theorem wp_symbol_snapshot_direct (receiver : Loc) (s : QuerySnapshot) (ref : w32) (node : pdg.Node) (a : pdg.AttributeSet)
    (hf : 0 < uint.Z ref ∧ uint.Z ref ≤ s.fields.length)
    (hfield : s.fields[(uint.Z ref - 1).toNat]? = some (W32 0))
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.nodes.length)
    (hnode : s.nodes[(uint.Z ref - 1).toNat]? = some node)
    (ha : 0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ s.attributes.length)
    (hrecord : s.attributes[(uint.Z node.Attributes' - 1).toNat]? = some a)
    (halloc : (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46)
    (entry : pdg.Access) (hreads : pdg.rowContents s.accessRows a.Reads' = [entry])
    (hcategory : node.Category' = W8 1) (hlocation : entry.Location' = W32 0) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ querySnapshotOwn receiver s }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"symbol" #receiver)) (Val #ref))
    {{ RET #(boundedReadSymbol (sint.Z s.view.kinds'.len) entry.Symbol'); querySnapshotOwn receiver s }} := by
  wp_start_folded as #Hsnapshot
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, Hgraph, Hfields, Hrows, Hkinds, Hunstable, Htracked, Hnodes, Hrecords,
    Hreferences, Haccesses, Hextensions, HrefRows, HaccessRows, HextensionRows, Htable, Hstrings⟩
  ihave Hvalues := pdg.readonlyRows_at_ref s.references s.referenceRows a.Outcomes' $$ HrefRows
  ihave Hreads := pdg.readonlyRows_at_ref s.accesses s.accessRows a.Reads' $$ HaccessRows
  rw [hreads]
  wp_apply wp_symbol_direct receiver s.view s.graph s.fields s.nodes node s.attributes a s.references s.accesses s.extensions (pdg.rowContents s.referenceRows a.Outcomes') entry ref DFrac.discard DFrac.discard DFrac.discard DFrac.discard
    hf hfield href hnode ha hrecord halloc hcategory hlocation
    $$ [Hview Hfields Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues Hreads] with ⟨Hview, Hfields, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Hreads⟩
  · iframe Hview Hfields Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues Hreads
  wp_end
  all_goals iexact Hsnapshot

theorem wp_symbol_snapshot_unavailable_operator (receiver : Loc) (s : QuerySnapshot) (ref : w32) (node : pdg.Node) (a : pdg.AttributeSet)
    (hf : 0 < uint.Z ref ∧ uint.Z ref ≤ s.fields.length)
    (hfield : s.fields[(uint.Z ref - 1).toNat]? = some (W32 0))
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.nodes.length)
    (hnode : s.nodes[(uint.Z ref - 1).toNat]? = some node)
    (ha : 0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ s.attributes.length)
    (hrecord : s.attributes[(uint.Z node.Attributes' - 1).toNat]? = some a)
    (halloc : (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46)
    (hnotdirect : directReadValue node (pdg.rowContents s.accessRows a.Reads') = false)
    (hstatus : a.Operator'.Status' ≠ 1#8)
:
    {{ isPkgInit (PROP := IProp GF) pkg ∗ querySnapshotOwn receiver s }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"symbol" #receiver)) (Val #ref))
    {{ RET #(W32 0); querySnapshotOwn receiver s }} := by
  wp_start_folded as #Hsnapshot
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, Hgraph, Hfields, Hrows, Hkinds, Hunstable, Htracked, Hnodes, Hrecords,
    Hreferences, Haccesses, Hextensions, HrefRows, HaccessRows, HextensionRows, Htable, Hstrings⟩
  ihave Hvalues := pdg.readonlyRows_at_ref s.references s.referenceRows a.Outcomes' $$ HrefRows
  ihave Hreads := pdg.readonlyRows_at_ref s.accesses s.accessRows a.Reads' $$ HaccessRows
  wp_apply wp_symbol_unavailable_operator receiver s.view s.graph s.fields s.nodes node s.attributes a s.references s.accesses s.extensions (pdg.rowContents s.referenceRows a.Outcomes') (pdg.rowContents s.accessRows a.Reads') ref DFrac.discard DFrac.discard DFrac.discard DFrac.discard hf hfield href hnode ha hrecord halloc hnotdirect hstatus
    $$ [Hview Hfields Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues Hreads] with ⟨Hview, Hfields, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Hreads⟩
  · iframe Hview Hfields Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues Hreads
  wp_end
  all_goals iexact Hsnapshot

theorem wp_symbol_snapshot_known_other (receiver : Loc) (s : QuerySnapshot) (ref : w32) (node : pdg.Node) (a : pdg.AttributeSet)
    (hf : 0 < uint.Z ref ∧ uint.Z ref ≤ s.fields.length)
    (hfield : s.fields[(uint.Z ref - 1).toNat]? = some (W32 0))
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.nodes.length)
    (hnode : s.nodes[(uint.Z ref - 1).toNat]? = some node)
    (ha : 0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ s.attributes.length)
    (hrecord : s.attributes[(uint.Z node.Attributes' - 1).toNat]? = some a)
    (halloc : (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46)
    (hnotdirect : directReadValue node (pdg.rowContents s.accessRows a.Reads') = false)
    (hstatus : a.Operator'.Status' = 1#8)
    (hop : pdg.tableText s.strings a.Operator'.Value' ≠ go!"parentheses")
:
    {{ isPkgInit (PROP := IProp GF) pkg ∗ querySnapshotOwn receiver s }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"symbol" #receiver)) (Val #ref))
    {{ RET #(W32 0); querySnapshotOwn receiver s }} := by
  wp_start_folded as #Hsnapshot
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, Hgraph, Hfields, Hrows, Hkinds, Hunstable, Htracked, Hnodes, Hrecords,
    Hreferences, Haccesses, Hextensions, HrefRows, HaccessRows, HextensionRows, Htable, Hstrings⟩
  ihave Hvalues := pdg.readonlyRows_at_ref s.references s.referenceRows a.Outcomes' $$ HrefRows
  ihave Hreads := pdg.readonlyRows_at_ref s.accesses s.accessRows a.Reads' $$ HaccessRows
  wp_apply wp_symbol_known_other receiver s.view s.graph s.fields s.nodes node s.attributes a s.references s.accesses s.extensions (pdg.rowContents s.referenceRows a.Outcomes') (pdg.rowContents s.accessRows a.Reads') s.table s.strings ref DFrac.discard DFrac.discard DFrac.discard DFrac.discard DFrac.discard hf hfield href hnode ha hrecord halloc hnotdirect hstatus hop
    $$ [Hview Hfields Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues Hreads Htable Hstrings] with ⟨Hview, Hfields, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Hreads, Htable, Hstrings⟩
  · iframe Hview Hfields Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues Hreads Htable Hstrings
  wp_end
  all_goals iexact Hsnapshot

theorem wp_symbol_snapshot_parentheses (receiver : Loc) (s : QuerySnapshot)
    (ref child result : w32) (node : pdg.Node) (a : pdg.AttributeSet) (row : GoArray w32 2)
    (hf : 0 < uint.Z ref ∧ uint.Z ref ≤ s.fields.length)
    (hfield : s.fields[(uint.Z ref - 1).toNat]? = some (W32 0))
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.nodes.length)
    (hnode : s.nodes[(uint.Z ref - 1).toNat]? = some node)
    (ha : 0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ s.attributes.length)
    (hrecord : s.attributes[(uint.Z node.Attributes' - 1).toNat]? = some a)
    (halloc : (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46)
    (hnotdirect : directReadValue node (pdg.rowContents s.accessRows a.Reads') = false)
    (hstatus : a.Operator'.Status' = 1#8)
    (hop : pdg.tableText s.strings a.Operator'.Value' = go!"parentheses")
    (hoperand : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row) (hchildref : row.arr[0]? = some child)
    (hchild :
      {{ isPkgInit (PROP := IProp GF) pkg ∗ querySnapshotOwn receiver s }}
        (App (Val #(methods (go.PointerType graphView.ty) go!"symbol" #receiver)) (Val #child))
      {{ RET #result; querySnapshotOwn receiver s }}) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ querySnapshotOwn receiver s }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"symbol" #receiver)) (Val #ref))
    {{ RET #result; querySnapshotOwn receiver s }} := by
  wp_start as #Hsnapshot
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, Hgraph, Hfields, Hrows, Hkinds, Hunstable, Htracked, Hnodes, Hrecords,
    Hreferences, Haccesses, Hextensions, HrefRows, HaccessRows, HextensionRows, Htable, Hstrings⟩
  ihave Hvalues := pdg.readonlyRows_at_ref s.references s.referenceRows a.Outcomes' $$ HrefRows
  ihave Hreads := pdg.readonlyRows_at_ref s.accesses s.accessRows a.Reads' $$ HaccessRows
  ihave %Hflen := ownSlice_len $$ Hfields
  ihave %Hnlen := ownSlice_len $$ Hnodes
  try wp_auto
  wp_if_destruct
  · exfalso; obtain ⟨_, _⟩ := href; word
  · have Hi : sint.Z (W64 (uint.Z (ref - W32 1))) = uint.Z ref - 1 := by obtain ⟨_, _⟩ := href; word
    rw [Hi]
    have hfb : 0 ≤ uint.Z ref - 1 ∧ uint.Z ref - 1 < sint.Z s.view.fields'.len := by obtain ⟨_, _⟩ := hf; word
    rw [ite_eq_left hfb]
    wp_apply wp_load_slice_index s.view.fields' (uint.Z ref - 1) s.fields DFrac.discard (W32 0) (by omega) $$ [Hfields] with Hfields
    · iframe Hfields; ipureintro; exact hfield
    try wp_auto
    wp_apply wp_attrs_present receiver s.view s.graph s.nodes node s.attributes a s.references s.accesses s.extensions (pdg.rowContents s.referenceRows a.Outcomes') ref DFrac.discard DFrac.discard href hnode ha hrecord halloc
      $$ [Hview Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues]
      with %out ⟨Hview, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Hout, %Hempty⟩
    · iframe Hview Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues
    try wp_auto
    rw [Hi]
    have hnb : 0 ≤ uint.Z ref - 1 ∧ uint.Z ref - 1 < sint.Z s.graph.Nodes'.len := by obtain ⟨_, _⟩ := href; word
    rw [ite_eq_left hnb]
    wp_apply wp_load_slice_index s.graph.Nodes' (uint.Z ref - 1) s.nodes DFrac.discard node (by omega) $$ [Hnodes] with Hnodes
    · iframe Hnodes; ipureintro; exact hnode
    try wp_auto
    wp_apply wp_directBindingRead node (pdg.expandedAttributes a s.references s.accesses s.extensions out) (pdg.rowContents s.accessRows a.Reads') DFrac.discard
      $$ [Hreads] with Hreads
    · simp only [pdg.expandedAttributes]
      iframe Hreads
    rw [hnotdirect]
    try wp_auto
    wp_apply wp_operatorText_known s.graph.Tables' a.Operator' s.table s.strings DFrac.discard hstatus $$ [Htable Hstrings] with ⟨Htable, Hstrings⟩
    · iframe Htable Hstrings
    try wp_auto
    rw [hop]
    try wp_auto
    wp_apply wp_operand_readonly receiver s.view s.operands ref (W64 0) row child hoperand (by decide) hrow hchildref DFrac.discard DFrac.discard
      $$ [Hview Hrows] with ⟨Hview, Hrows⟩
    · iframe Hview Hrows
    try wp_auto
    wp_apply hchild $$ [Hsnapshot] with HchildSnapshot
    · iframe Hsnapshot
    try wp_auto
    wp_end

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
