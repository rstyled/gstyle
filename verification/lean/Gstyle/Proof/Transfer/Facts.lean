module
public import Gstyle.Proof.Transfer.Extension
public import Gstyle.Proof.Pdg.Attributes
public import Gstyle.Proof.StateMap.Layout
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

theorem wp_attrs_zero (receiver : Loc) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"attrs" #receiver)) (Val #(W32 0)))
    {{ RET #(zero_val pdg.Attributes); True }} := by
  wp_start
  try wp_auto
  wp_end

theorem wp_attrs_absent (receiver : Loc) (view : graphView) (graph : pdg.Graph)
    (nodes : List pdg.Node) (node : pdg.Node) (records : List pdg.AttributeSet)
    (ref : w32) (vd gd : DFrac)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ nodes.length)
    (hnode : nodes[(uint.Z ref - 1).toNat]? = some node)
    (ha : ¬ (0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ records.length)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{vd} view ∗ view.graph' ↦{gd} graph ∗
       graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records) }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"attrs" #receiver)) (Val #ref))
    {{ RET #(zero_val pdg.Attributes); receiver ↦{vd} view ∗ view.graph' ↦{gd} graph ∗
       graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records }} := by
  wp_start as ⟨Hview, Hgraph, Hnodes, Hrecords⟩
  ihave %Hlen := ownSlice_len $$ Hnodes
  try wp_auto
  wp_if_destruct
  · exfalso; obtain ⟨_, _⟩ := href; word
  · have Hi : sint.Z (W64 (uint.Z (ref - W32 1))) = uint.Z ref - 1 := by obtain ⟨_, _⟩ := href; word
    rw [Hi]
    rw [ite_eq_left (by obtain ⟨_, _⟩ := href; constructor <;> word : 0 ≤ uint.Z ref - 1 ∧ uint.Z ref - 1 < sint.Z graph.Nodes'.len)]
    wp_apply wp_load_slice_index graph.Nodes' (uint.Z ref - 1) nodes gd node (by obtain ⟨_, _⟩ := href; word) $$ [Hnodes] with Hnodes
    · iframe; ipureintro; exact hnode
    try wp_auto
    wp_apply pdg.wp_NodeAttributes_absent view.graph' graph node records gd ha $$ [Hgraph Hrecords] with ⟨Hgraph, Hrecords⟩
    · iframe
    wp_end

theorem wp_attrs_present (receiver : Loc) (view : graphView) (graph : pdg.Graph)
    (nodes : List pdg.Node) (node : pdg.Node) (records : List pdg.AttributeSet) (a : pdg.AttributeSet)
    (references accesses extensions : List GoSlice) (values : List w32)
    (ref : w32) (vd gd : DFrac)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ nodes.length)
    (hnode : nodes[(uint.Z ref - 1).toNat]? = some node)
    (ha : 0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ records.length)
    (hrecord : records[(uint.Z node.Attributes' - 1).toNat]? = some a) (halloc : values.length ≤ 2^46) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{vd} view ∗ view.graph' ↦{gd} graph ∗
       graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records ∗
       graph.AttributeTables'.References' ↦*{gd} references ∗ graph.AttributeTables'.Accesses' ↦*{gd} accesses ∗
       graph.AttributeTables'.Extensions' ↦*{gd} extensions ∗ pdg.listRow references a.Outcomes' ↦*{gd} values) }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"attrs" #receiver)) (Val #ref))
    {{ (out : GoSlice), RET #(pdg.expandedAttributes a references accesses extensions out);
       receiver ↦{vd} view ∗ view.graph' ↦{gd} graph ∗ graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records ∗
       graph.AttributeTables'.References' ↦*{gd} references ∗ graph.AttributeTables'.Accesses' ↦*{gd} accesses ∗
       graph.AttributeTables'.Extensions' ↦*{gd} extensions ∗ pdg.listRow references a.Outcomes' ↦*{gd} values ∗
       out ↦* values ∗ ⌜values = [] → out = slice.nil⌝ }} := by
  wp_start as ⟨Hview, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues⟩
  ihave %Hlen := ownSlice_len $$ Hnodes
  try wp_auto
  wp_if_destruct
  · exfalso; obtain ⟨_, _⟩ := href; word
  · have Hi : sint.Z (W64 (uint.Z (ref - W32 1))) = uint.Z ref - 1 := by obtain ⟨_, _⟩ := href; word
    rw [Hi]
    rw [ite_eq_left (by obtain ⟨_, _⟩ := href; constructor <;> word : 0 ≤ uint.Z ref - 1 ∧ uint.Z ref - 1 < sint.Z graph.Nodes'.len)]
    wp_apply wp_load_slice_index graph.Nodes' (uint.Z ref - 1) nodes gd node (by obtain ⟨_, _⟩ := href; word) $$ [Hnodes] with Hnodes
    · iframe; ipureintro; exact hnode
    try wp_auto
    wp_apply pdg.wp_NodeAttributes_present view.graph' graph node records a references accesses extensions values gd ha hrecord halloc $$ [Hgraph Hrecords Hreferences Haccesses Hextensions Hvalues] with %out ⟨Hgraph, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Hout, %Hempty⟩
    · iframe
    wp_end

variable [pdg.Tables.TypeAssumptions] [pdg.Extension.TypeAssumptions]
variable [FuncUnfold extension [] extension.impl]

theorem wp_fact_zero (receiver : Loc) (view : graphView) (name : GoString) (vd : DFrac) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ receiver ↦{vd} view }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"fact" #receiver)) (Val #(W32 0))) (Val #name))
    {{ RET #(go!""); receiver ↦{vd} view }} := by
  wp_start as Hview
  try wp_auto
  wp_apply wp_attrs_zero receiver
  try wp_auto
  wp_apply wp_extension_empty view.graph' slice.nil name (DFrac.own 1) $$ [] with Hempty
  · iframe; iapply ownSlice_nil
  wp_end

theorem wp_fact_absent (receiver : Loc) (view : graphView) (graph : pdg.Graph)
    (nodes : List pdg.Node) (node : pdg.Node) (records : List pdg.AttributeSet)
    (ref : w32) (name : GoString) (vd gd : DFrac)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ nodes.length)
    (hnode : nodes[(uint.Z ref - 1).toNat]? = some node)
    (ha : ¬ (0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ records.length)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{vd} view ∗ view.graph' ↦{gd} graph ∗
       graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"fact" #receiver)) (Val #ref)) (Val #name))
    {{ RET #(go!""); receiver ↦{vd} view ∗ view.graph' ↦{gd} graph ∗
       graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records }} := by
  wp_start as ⟨Hview, Hgraph, Hnodes, Hrecords⟩
  try wp_auto
  wp_apply wp_attrs_absent receiver view graph nodes node records ref vd gd href hnode ha $$ [Hview Hgraph Hnodes Hrecords] with ⟨Hview, Hgraph, Hnodes, Hrecords⟩
  · iframe
  try wp_auto
  wp_apply wp_extension_empty view.graph' slice.nil name (DFrac.own 1) $$ [] with Hempty
  · iframe; iapply ownSlice_nil
  wp_end

theorem wp_fact_present (receiver : Loc) (view : graphView) (graph : pdg.Graph)
    (nodes : List pdg.Node) (node : pdg.Node) (records : List pdg.AttributeSet) (a : pdg.AttributeSet)
    (references accesses extensions : List GoSlice) (values : List w32)
    (table : pdg.Tables) (strings : List GoString) (entries : List pdg.Extension)
    (ref : w32) (name : GoString) (vd gd td ed : DFrac)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ nodes.length)
    (hnode : nodes[(uint.Z ref - 1).toNat]? = some node)
    (ha : 0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ records.length)
    (hrecord : records[(uint.Z node.Attributes' - 1).toNat]? = some a) (halloc : values.length ≤ 2^46) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{vd} view ∗ view.graph' ↦{gd} graph ∗
       graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records ∗
       graph.AttributeTables'.References' ↦*{gd} references ∗ graph.AttributeTables'.Accesses' ↦*{gd} accesses ∗
       graph.AttributeTables'.Extensions' ↦*{gd} extensions ∗ pdg.listRow references a.Outcomes' ↦*{gd} values ∗
       graph.Tables' ↦{td} table ∗ table.Strings' ↦*{td} strings ∗ pdg.listRow extensions a.Extensions' ↦*{ed} entries) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"fact" #receiver)) (Val #ref)) (Val #name))
    {{ RET #(extensionValue strings name entries);
       receiver ↦{vd} view ∗ view.graph' ↦{gd} graph ∗ graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records ∗
       graph.AttributeTables'.References' ↦*{gd} references ∗ graph.AttributeTables'.Accesses' ↦*{gd} accesses ∗
       graph.AttributeTables'.Extensions' ↦*{gd} extensions ∗ pdg.listRow references a.Outcomes' ↦*{gd} values ∗
       graph.Tables' ↦{td} table ∗ table.Strings' ↦*{td} strings ∗ pdg.listRow extensions a.Extensions' ↦*{ed} entries }} := by
  wp_start as ⟨Hview, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Htable, Hstrings, Hentries⟩
  try wp_auto
  wp_apply wp_attrs_present receiver view graph nodes node records a references accesses extensions values ref vd gd href hnode ha hrecord halloc $$ [Hview Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues] with %out ⟨Hview, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Hout, %Hempty⟩
  · iframe
  try wp_auto
  wp_apply wp_extension view.graph' graph table strings (pdg.listRow extensions a.Extensions') entries name gd td ed $$ [Hgraph Htable Hstrings Hentries] with ⟨Hgraph, Htable, Hstrings, Hentries⟩
  · iframe
  wp_end

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
