module
public import Gstyle.Proof.Transfer.BindingRead
public import Gstyle.Proof.Transfer.Operator
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

def boundedReadSymbol (limit : Int) (symbol : w32) : w32 :=
  if limit < uint.Z symbol then W32 0 else symbol

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

variable [pdg.Access.TypeAssumptions] [pdg.Tables.TypeAssumptions]
variable [FuncUnfold directBindingRead [] directBindingRead.impl]
variable [FuncUnfold operatorText [] operatorText.impl]

theorem wp_symbol_direct (receiver : Loc) (view : graphView) (graph : pdg.Graph)
    (fields : List w32) (nodes : List pdg.Node) (node : pdg.Node)
    (records : List pdg.AttributeSet) (a : pdg.AttributeSet)
    (references accesses extensions : List GoSlice) (values : List w32) (entry : pdg.Access)
    (ref : w32) (vd gd fd rd : DFrac)
    (hf : 0 < uint.Z ref ∧ uint.Z ref ≤ fields.length)
    (hfield : fields[(uint.Z ref - 1).toNat]? = some (W32 0))
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ nodes.length)
    (hnode : nodes[(uint.Z ref - 1).toNat]? = some node)
    (ha : 0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ records.length)
    (hrecord : records[(uint.Z node.Attributes' - 1).toNat]? = some a) (halloc : values.length ≤ 2^46)
    (hcategory : node.Category' = W8 1) (hlocation : entry.Location' = W32 0) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{vd} view ∗ view.fields' ↦*{fd} fields ∗
       view.graph' ↦{gd} graph ∗ graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records ∗
       graph.AttributeTables'.References' ↦*{gd} references ∗ graph.AttributeTables'.Accesses' ↦*{gd} accesses ∗
       graph.AttributeTables'.Extensions' ↦*{gd} extensions ∗ pdg.listRow references a.Outcomes' ↦*{gd} values ∗
       pdg.listRow accesses a.Reads' ↦*{rd} [entry]) }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"symbol" #receiver)) (Val #ref))
    {{ RET #(boundedReadSymbol (sint.Z view.kinds'.len) entry.Symbol');
       receiver ↦{vd} view ∗ view.fields' ↦*{fd} fields ∗ view.graph' ↦{gd} graph ∗
       graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records ∗
       graph.AttributeTables'.References' ↦*{gd} references ∗ graph.AttributeTables'.Accesses' ↦*{gd} accesses ∗
       graph.AttributeTables'.Extensions' ↦*{gd} extensions ∗ pdg.listRow references a.Outcomes' ↦*{gd} values ∗
       pdg.listRow accesses a.Reads' ↦*{rd} [entry] }} := by
  wp_start as ⟨Hview, Hfields, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Hreads⟩
  ihave %Hflen := ownSlice_len $$ Hfields
  ihave %Hnlen := ownSlice_len $$ Hnodes
  try wp_auto
  wp_if_destruct
  · exfalso; obtain ⟨_, _⟩ := href; word
  · have Hi : sint.Z (W64 (uint.Z (ref - W32 1))) = uint.Z ref - 1 := by obtain ⟨_, _⟩ := href; word
    rw [Hi]
    have hfb : 0 ≤ uint.Z ref - 1 ∧ uint.Z ref - 1 < sint.Z view.fields'.len := by obtain ⟨_, _⟩ := hf; word
    rw [ite_eq_left hfb]
    wp_apply wp_load_slice_index view.fields' (uint.Z ref - 1) fields fd (W32 0) (by omega) $$ [Hfields] with Hfields
    · iframe; ipureintro; exact hfield
    try wp_auto
    wp_apply wp_attrs_present receiver view graph nodes node records a references accesses extensions values ref vd gd href hnode ha hrecord halloc
      $$ [Hview Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues]
      with %out ⟨Hview, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Hout, %Hempty⟩
    · iframe
    try wp_auto
    rw [Hi]
    have hnb : 0 ≤ uint.Z ref - 1 ∧ uint.Z ref - 1 < sint.Z graph.Nodes'.len := by obtain ⟨_, _⟩ := href; word
    rw [ite_eq_left hnb]
    wp_apply wp_load_slice_index graph.Nodes' (uint.Z ref - 1) nodes gd node (by omega) $$ [Hnodes] with Hnodes
    · iframe; ipureintro; exact hnode
    try wp_auto
    wp_apply wp_directBindingRead_single node (pdg.expandedAttributes a references accesses extensions out) entry rd
      $$ [Hreads] with Hreads
    · iframe
    have hdirect : directReadValue node [entry] = true := by
      unfold directReadValue
      exact decide_eq_true ⟨hcategory, hlocation⟩
    rw [hdirect]
    try wp_auto
    ihave %Hrlen := ownSlice_len $$ Hreads
    have hrl : (pdg.listRow accesses a.Reads').len = W64 1 := by
      simp only [List.length_singleton] at Hrlen; word
    rw [show sint.Z (W64 0) = 0 from rfl]
    rw [ite_eq_left (by simp [hrl] : (0 : Int) ≤ 0 ∧ 0 < sint.Z (pdg.listRow accesses a.Reads').len)]
    icases ownSlice_elem_acc 0 entry (pdg.listRow accesses a.Reads') rd [entry] (by omega) (by rfl)
      $$ Hreads with ⟨Hentry, Hreads⟩
    try wp_auto
    ihave Hreads := Hreads $$ %entry [Hentry]
    · iexact Hentry
    rw [list_set_lookup_self [entry] (Int.toNat (0 : Int)) entry (by rfl)]
    have Hconv : sint.Z (W64 (uint.Z entry.Symbol')) = uint.Z entry.Symbol' := by word
    rw [Hconv]
    unfold boundedReadSymbol
    by_cases hl : sint.Z view.kinds'.len < uint.Z entry.Symbol'
    · rw [decide_eq_true hl, ite_eq_left hl]
      try wp_auto
      wp_end
    · rw [decide_eq_false hl, ite_eq_right hl]
      try wp_auto
      wp_end

theorem wp_symbol_unavailable_operator (receiver : Loc) (view : graphView) (graph : pdg.Graph)
    (fields : List w32) (nodes : List pdg.Node) (node : pdg.Node)
    (records : List pdg.AttributeSet) (a : pdg.AttributeSet)
    (references accesses extensions : List GoSlice) (values : List w32) (reads : List pdg.Access)
    (ref : w32) (vd gd fd rd : DFrac)
    (hf : 0 < uint.Z ref ∧ uint.Z ref ≤ fields.length)
    (hfield : fields[(uint.Z ref - 1).toNat]? = some (W32 0))
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ nodes.length)
    (hnode : nodes[(uint.Z ref - 1).toNat]? = some node)
    (ha : 0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ records.length)
    (hrecord : records[(uint.Z node.Attributes' - 1).toNat]? = some a) (halloc : values.length ≤ 2^46)
    (hnotdirect : directReadValue node reads = false) (hstatus : a.Operator'.Status' ≠ 1#8) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{vd} view ∗ view.fields' ↦*{fd} fields ∗
       view.graph' ↦{gd} graph ∗ graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records ∗
       graph.AttributeTables'.References' ↦*{gd} references ∗ graph.AttributeTables'.Accesses' ↦*{gd} accesses ∗
       graph.AttributeTables'.Extensions' ↦*{gd} extensions ∗ pdg.listRow references a.Outcomes' ↦*{gd} values ∗
       pdg.listRow accesses a.Reads' ↦*{rd} reads) }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"symbol" #receiver)) (Val #ref))
    {{ RET #(W32 0);
       receiver ↦{vd} view ∗ view.fields' ↦*{fd} fields ∗ view.graph' ↦{gd} graph ∗
       graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records ∗
       graph.AttributeTables'.References' ↦*{gd} references ∗ graph.AttributeTables'.Accesses' ↦*{gd} accesses ∗
       graph.AttributeTables'.Extensions' ↦*{gd} extensions ∗ pdg.listRow references a.Outcomes' ↦*{gd} values ∗
       pdg.listRow accesses a.Reads' ↦*{rd} reads }} := by
  wp_start as ⟨Hview, Hfields, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Hreads⟩
  ihave %Hflen := ownSlice_len $$ Hfields
  ihave %Hnlen := ownSlice_len $$ Hnodes
  try wp_auto
  wp_if_destruct
  · exfalso; obtain ⟨_, _⟩ := href; word
  · have Hi : sint.Z (W64 (uint.Z (ref - W32 1))) = uint.Z ref - 1 := by obtain ⟨_, _⟩ := href; word
    rw [Hi]
    have hfb : 0 ≤ uint.Z ref - 1 ∧ uint.Z ref - 1 < sint.Z view.fields'.len := by obtain ⟨_, _⟩ := hf; word
    rw [ite_eq_left hfb]
    wp_apply wp_load_slice_index view.fields' (uint.Z ref - 1) fields fd (W32 0) (by omega) $$ [Hfields] with Hfields
    · iframe; ipureintro; exact hfield
    try wp_auto
    wp_apply wp_attrs_present receiver view graph nodes node records a references accesses extensions values ref vd gd href hnode ha hrecord halloc
      $$ [Hview Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues]
      with %out ⟨Hview, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Hout, %Hempty⟩
    · iframe
    try wp_auto
    rw [Hi]
    have hnb : 0 ≤ uint.Z ref - 1 ∧ uint.Z ref - 1 < sint.Z graph.Nodes'.len := by obtain ⟨_, _⟩ := href; word
    rw [ite_eq_left hnb]
    wp_apply wp_load_slice_index graph.Nodes' (uint.Z ref - 1) nodes gd node (by omega) $$ [Hnodes] with Hnodes
    · iframe; ipureintro; exact hnode
    try wp_auto
    wp_apply wp_directBindingRead node (pdg.expandedAttributes a references accesses extensions out) reads rd
      $$ [Hreads] with Hreads
    · iframe
    rw [hnotdirect]
    try wp_auto
    wp_apply wp_operatorText_unavailable graph.Tables' a.Operator' hstatus
    try wp_auto
    wp_end

theorem wp_symbol_known_other (receiver : Loc) (view : graphView) (graph : pdg.Graph)
    (fields : List w32) (nodes : List pdg.Node) (node : pdg.Node)
    (records : List pdg.AttributeSet) (a : pdg.AttributeSet)
    (references accesses extensions : List GoSlice) (values : List w32) (reads : List pdg.Access)
    (table : pdg.Tables) (strings : List GoString)
    (ref : w32) (vd gd fd rd td : DFrac)
    (hf : 0 < uint.Z ref ∧ uint.Z ref ≤ fields.length)
    (hfield : fields[(uint.Z ref - 1).toNat]? = some (W32 0))
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ nodes.length)
    (hnode : nodes[(uint.Z ref - 1).toNat]? = some node)
    (ha : 0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ records.length)
    (hrecord : records[(uint.Z node.Attributes' - 1).toNat]? = some a) (halloc : values.length ≤ 2^46)
    (hnotdirect : directReadValue node reads = false) (hstatus : a.Operator'.Status' = 1#8)
    (hop : pdg.tableText strings a.Operator'.Value' ≠ go!"parentheses") :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{vd} view ∗ view.fields' ↦*{fd} fields ∗
       view.graph' ↦{gd} graph ∗ graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records ∗
       graph.AttributeTables'.References' ↦*{gd} references ∗ graph.AttributeTables'.Accesses' ↦*{gd} accesses ∗
       graph.AttributeTables'.Extensions' ↦*{gd} extensions ∗ pdg.listRow references a.Outcomes' ↦*{gd} values ∗
       pdg.listRow accesses a.Reads' ↦*{rd} reads ∗ graph.Tables' ↦{td} table ∗ table.Strings' ↦*{td} strings) }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"symbol" #receiver)) (Val #ref))
    {{ RET #(W32 0);
       receiver ↦{vd} view ∗ view.fields' ↦*{fd} fields ∗ view.graph' ↦{gd} graph ∗
       graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records ∗
       graph.AttributeTables'.References' ↦*{gd} references ∗ graph.AttributeTables'.Accesses' ↦*{gd} accesses ∗
       graph.AttributeTables'.Extensions' ↦*{gd} extensions ∗ pdg.listRow references a.Outcomes' ↦*{gd} values ∗
       pdg.listRow accesses a.Reads' ↦*{rd} reads ∗ graph.Tables' ↦{td} table ∗ table.Strings' ↦*{td} strings }} := by
  wp_start as ⟨Hview, Hfields, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Hreads, Htable, Hstrings⟩
  ihave %Hflen := ownSlice_len $$ Hfields
  ihave %Hnlen := ownSlice_len $$ Hnodes
  try wp_auto
  wp_if_destruct
  · exfalso; obtain ⟨_, _⟩ := href; word
  · have Hi : sint.Z (W64 (uint.Z (ref - W32 1))) = uint.Z ref - 1 := by obtain ⟨_, _⟩ := href; word
    rw [Hi]
    have hfb : 0 ≤ uint.Z ref - 1 ∧ uint.Z ref - 1 < sint.Z view.fields'.len := by obtain ⟨_, _⟩ := hf; word
    rw [ite_eq_left hfb]
    wp_apply wp_load_slice_index view.fields' (uint.Z ref - 1) fields fd (W32 0) (by omega) $$ [Hfields] with Hfields
    · iframe; ipureintro; exact hfield
    try wp_auto
    wp_apply wp_attrs_present receiver view graph nodes node records a references accesses extensions values ref vd gd href hnode ha hrecord halloc
      $$ [Hview Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues]
      with %out ⟨Hview, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Hout, %Hempty⟩
    · iframe
    try wp_auto
    rw [Hi]
    have hnb : 0 ≤ uint.Z ref - 1 ∧ uint.Z ref - 1 < sint.Z graph.Nodes'.len := by obtain ⟨_, _⟩ := href; word
    rw [ite_eq_left hnb]
    wp_apply wp_load_slice_index graph.Nodes' (uint.Z ref - 1) nodes gd node (by omega) $$ [Hnodes] with Hnodes
    · iframe; ipureintro; exact hnode
    try wp_auto
    wp_apply wp_directBindingRead node (pdg.expandedAttributes a references accesses extensions out) reads rd
      $$ [Hreads] with Hreads
    · iframe
    rw [hnotdirect]
    try wp_auto
    wp_apply wp_operatorText_known graph.Tables' a.Operator' table strings td hstatus $$ [Htable Hstrings] with ⟨Htable, Hstrings⟩
    · iframe
    try wp_auto
    rw [decide_eq_false hop]
    try wp_auto
    wp_end

theorem wp_symbol_absent_attributes (receiver : Loc) (view : graphView) (graph : pdg.Graph)
    (fields : List w32) (nodes : List pdg.Node) (node : pdg.Node) (records : List pdg.AttributeSet)
    (ref : w32) (vd gd fd : DFrac)
    (hf : 0 < uint.Z ref ∧ uint.Z ref ≤ fields.length)
    (hfield : fields[(uint.Z ref - 1).toNat]? = some (W32 0))
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ nodes.length)
    (hnode : nodes[(uint.Z ref - 1).toNat]? = some node)
    (ha : ¬ (0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ records.length)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{vd} view ∗ view.fields' ↦*{fd} fields ∗
       view.graph' ↦{gd} graph ∗ graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records) }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"symbol" #receiver)) (Val #ref))
    {{ RET #(W32 0); receiver ↦{vd} view ∗ view.fields' ↦*{fd} fields ∗ view.graph' ↦{gd} graph ∗
       graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records }} := by
  wp_start as ⟨Hview, Hfields, Hgraph, Hnodes, Hrecords⟩
  ihave %Hflen := ownSlice_len $$ Hfields
  ihave %Hnlen := ownSlice_len $$ Hnodes
  try wp_auto
  wp_if_destruct
  · exfalso; obtain ⟨_, _⟩ := href; word
  · have Hi : sint.Z (W64 (uint.Z (ref - W32 1))) = uint.Z ref - 1 := by obtain ⟨_, _⟩ := href; word
    rw [Hi]
    have hfb : 0 ≤ uint.Z ref - 1 ∧ uint.Z ref - 1 < sint.Z view.fields'.len := by obtain ⟨_, _⟩ := hf; word
    rw [ite_eq_left hfb]
    wp_apply wp_load_slice_index view.fields' (uint.Z ref - 1) fields fd (W32 0) (by omega) $$ [Hfields] with Hfields
    · iframe; ipureintro; exact hfield
    try wp_auto
    wp_apply wp_attrs_absent receiver view graph nodes node records ref vd gd href hnode ha
      $$ [Hview Hgraph Hnodes Hrecords] with ⟨Hview, Hgraph, Hnodes, Hrecords⟩
    · iframe
    try wp_auto
    rw [Hi]
    have hnb : 0 ≤ uint.Z ref - 1 ∧ uint.Z ref - 1 < sint.Z graph.Nodes'.len := by obtain ⟨_, _⟩ := href; word
    rw [ite_eq_left hnb]
    wp_apply wp_load_slice_index graph.Nodes' (uint.Z ref - 1) nodes gd node (by omega) $$ [Hnodes] with Hnodes
    · iframe; ipureintro; exact hnode
    try wp_auto
    wp_apply wp_directBindingRead_arity node (zero_val pdg.Attributes) (by change W64 0 ≠ W64 1; decide)
    try wp_auto
    wp_apply wp_operatorText_unavailable graph.Tables' (zero_val pdg.Fact) (by change (0#8) ≠ 1#8; decide)
    try wp_auto
    wp_end

theorem wp_symbol_parentheses (receiver : Loc) (view : graphView) (graph : pdg.Graph)
    (fields : List w32) (nodes : List pdg.Node) (node : pdg.Node)
    (records : List pdg.AttributeSet) (a : pdg.AttributeSet)
    (references accesses extensions : List GoSlice) (values : List w32) (reads : List pdg.Access)
    (table : pdg.Tables) (strings : List GoString)
    (rows : List (GoArray w32 2)) (row : GoArray w32 2) (child result : w32)
    (ref : w32) (vd gd fd rd td od : DFrac)
    (hf : 0 < uint.Z ref ∧ uint.Z ref ≤ fields.length)
    (hfield : fields[(uint.Z ref - 1).toNat]? = some (W32 0))
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ nodes.length)
    (hnode : nodes[(uint.Z ref - 1).toNat]? = some node)
    (ha : 0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ records.length)
    (hrecord : records[(uint.Z node.Attributes' - 1).toNat]? = some a) (halloc : values.length ≤ 2^46)
    (hnotdirect : directReadValue node reads = false) (hstatus : a.Operator'.Status' = 1#8)
    (hop : pdg.tableText strings a.Operator'.Value' = go!"parentheses")
    (hoperand : 0 < uint.Z ref ∧ uint.Z ref ≤ rows.length)
    (hrow : rows[(uint.Z ref - 1).toNat]? = some row) (hchildref : row.arr[0]? = some child)
    (hchild :
      {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{vd} view ∗ view.fields' ↦*{fd} fields ∗
       view.graph' ↦{gd} graph ∗ graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records ∗
       graph.AttributeTables'.References' ↦*{gd} references ∗ graph.AttributeTables'.Accesses' ↦*{gd} accesses ∗
       graph.AttributeTables'.Extensions' ↦*{gd} extensions ∗ pdg.listRow references a.Outcomes' ↦*{gd} values ∗
       pdg.listRow accesses a.Reads' ↦*{rd} reads ∗ graph.Tables' ↦{td} table ∗ table.Strings' ↦*{td} strings ∗ view.operands' ↦*{od} rows) }}
        (App (Val #(methods (go.PointerType graphView.ty) go!"symbol" #receiver)) (Val #child))
      {{ RET #result; receiver ↦{vd} view ∗ view.fields' ↦*{fd} fields ∗
       view.graph' ↦{gd} graph ∗ graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records ∗
       graph.AttributeTables'.References' ↦*{gd} references ∗ graph.AttributeTables'.Accesses' ↦*{gd} accesses ∗
       graph.AttributeTables'.Extensions' ↦*{gd} extensions ∗ pdg.listRow references a.Outcomes' ↦*{gd} values ∗
       pdg.listRow accesses a.Reads' ↦*{rd} reads ∗ graph.Tables' ↦{td} table ∗ table.Strings' ↦*{td} strings ∗ view.operands' ↦*{od} rows }}) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{vd} view ∗ view.fields' ↦*{fd} fields ∗
       view.graph' ↦{gd} graph ∗ graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records ∗
       graph.AttributeTables'.References' ↦*{gd} references ∗ graph.AttributeTables'.Accesses' ↦*{gd} accesses ∗
       graph.AttributeTables'.Extensions' ↦*{gd} extensions ∗ pdg.listRow references a.Outcomes' ↦*{gd} values ∗
       pdg.listRow accesses a.Reads' ↦*{rd} reads ∗ graph.Tables' ↦{td} table ∗ table.Strings' ↦*{td} strings ∗ view.operands' ↦*{od} rows) }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"symbol" #receiver)) (Val #ref))
    {{ RET #result;
       receiver ↦{vd} view ∗ view.fields' ↦*{fd} fields ∗ view.graph' ↦{gd} graph ∗
       graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records ∗
       graph.AttributeTables'.References' ↦*{gd} references ∗ graph.AttributeTables'.Accesses' ↦*{gd} accesses ∗
       graph.AttributeTables'.Extensions' ↦*{gd} extensions ∗ pdg.listRow references a.Outcomes' ↦*{gd} values ∗
       pdg.listRow accesses a.Reads' ↦*{rd} reads ∗ graph.Tables' ↦{td} table ∗ table.Strings' ↦*{td} strings ∗ view.operands' ↦*{od} rows }} := by
  wp_start as ⟨Hview, Hfields, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Hreads, Htable, Hstrings, Hrows⟩
  ihave %Hflen := ownSlice_len $$ Hfields
  ihave %Hnlen := ownSlice_len $$ Hnodes
  try wp_auto
  wp_if_destruct
  · exfalso; obtain ⟨_, _⟩ := href; word
  · have Hi : sint.Z (W64 (uint.Z (ref - W32 1))) = uint.Z ref - 1 := by obtain ⟨_, _⟩ := href; word
    rw [Hi]
    have hfb : 0 ≤ uint.Z ref - 1 ∧ uint.Z ref - 1 < sint.Z view.fields'.len := by obtain ⟨_, _⟩ := hf; word
    rw [ite_eq_left hfb]
    wp_apply wp_load_slice_index view.fields' (uint.Z ref - 1) fields fd (W32 0) (by omega) $$ [Hfields] with Hfields
    · iframe; ipureintro; exact hfield
    try wp_auto
    wp_apply wp_attrs_present receiver view graph nodes node records a references accesses extensions values ref vd gd href hnode ha hrecord halloc
      $$ [Hview Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues]
      with %out ⟨Hview, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Hout, %Hempty⟩
    · iframe
    try wp_auto
    rw [Hi]
    have hnb : 0 ≤ uint.Z ref - 1 ∧ uint.Z ref - 1 < sint.Z graph.Nodes'.len := by obtain ⟨_, _⟩ := href; word
    rw [ite_eq_left hnb]
    wp_apply wp_load_slice_index graph.Nodes' (uint.Z ref - 1) nodes gd node (by omega) $$ [Hnodes] with Hnodes
    · iframe; ipureintro; exact hnode
    try wp_auto
    wp_apply wp_directBindingRead node (pdg.expandedAttributes a references accesses extensions out) reads rd
      $$ [Hreads] with Hreads
    · iframe
    rw [hnotdirect]
    try wp_auto
    wp_apply wp_operatorText_known graph.Tables' a.Operator' table strings td hstatus $$ [Htable Hstrings] with ⟨Htable, Hstrings⟩
    · iframe
    try wp_auto
    rw [hop]
    try wp_auto
    wp_apply wp_operand_readonly receiver view rows ref (W64 0) row child hoperand (by decide) hrow hchildref vd od
      $$ [Hview Hrows] with ⟨Hview, Hrows⟩
    · iframe
    try wp_auto
    wp_apply hchild $$ [Hview Hfields Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues Hreads Htable Hstrings Hrows]
      with ⟨Hview, Hfields, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Hreads, Htable, Hstrings, Hrows⟩
    · iframe
    try wp_auto
    wp_end

theorem wp_symbol_parentheses_cached_child (receiver : Loc) (view : graphView) (graph : pdg.Graph)
    (fields : List w32) (nodes : List pdg.Node) (node : pdg.Node)
    (records : List pdg.AttributeSet) (a : pdg.AttributeSet)
    (references accesses extensions : List GoSlice) (values : List w32) (reads : List pdg.Access)
    (table : pdg.Tables) (strings : List GoString)
    (rows : List (GoArray w32 2)) (row : GoArray w32 2) (child result : w32)
    (ref : w32) (vd gd fd rd td od : DFrac)
    (hf : 0 < uint.Z ref ∧ uint.Z ref ≤ fields.length)
    (hfield : fields[(uint.Z ref - 1).toNat]? = some (W32 0))
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ nodes.length)
    (hnode : nodes[(uint.Z ref - 1).toNat]? = some node)
    (ha : 0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ records.length)
    (hrecord : records[(uint.Z node.Attributes' - 1).toNat]? = some a) (halloc : values.length ≤ 2^46)
    (hnotdirect : directReadValue node reads = false) (hstatus : a.Operator'.Status' = 1#8)
    (hop : pdg.tableText strings a.Operator'.Value' = go!"parentheses")
    (hoperand : 0 < uint.Z ref ∧ uint.Z ref ≤ rows.length)
    (hrow : rows[(uint.Z ref - 1).toNat]? = some row) (hchildref : row.arr[0]? = some child)
    (hchildfields : 0 < uint.Z child ∧ uint.Z child ≤ fields.length)
    (hcached : fields[(uint.Z child - 1).toNat]? = some result)
    (hresult : result ≠ W32 0) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{vd} view ∗ view.fields' ↦*{fd} fields ∗
       view.graph' ↦{gd} graph ∗ graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records ∗
       graph.AttributeTables'.References' ↦*{gd} references ∗ graph.AttributeTables'.Accesses' ↦*{gd} accesses ∗
       graph.AttributeTables'.Extensions' ↦*{gd} extensions ∗ pdg.listRow references a.Outcomes' ↦*{gd} values ∗
       pdg.listRow accesses a.Reads' ↦*{rd} reads ∗ graph.Tables' ↦{td} table ∗ table.Strings' ↦*{td} strings ∗ view.operands' ↦*{od} rows) }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"symbol" #receiver)) (Val #ref))
    {{ RET #result;
       receiver ↦{vd} view ∗ view.fields' ↦*{fd} fields ∗ view.graph' ↦{gd} graph ∗
       graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records ∗
       graph.AttributeTables'.References' ↦*{gd} references ∗ graph.AttributeTables'.Accesses' ↦*{gd} accesses ∗
       graph.AttributeTables'.Extensions' ↦*{gd} extensions ∗ pdg.listRow references a.Outcomes' ↦*{gd} values ∗
       pdg.listRow accesses a.Reads' ↦*{rd} reads ∗ graph.Tables' ↦{td} table ∗ table.Strings' ↦*{td} strings ∗ view.operands' ↦*{od} rows }} := by
  wp_start_folded as Hinputs
  wp_apply wp_symbol_parentheses receiver view graph fields nodes node records a references accesses extensions values reads table strings rows row child result ref vd gd fd rd td od hf hfield href hnode ha hrecord halloc hnotdirect hstatus hop hoperand hrow hchildref
    $$ [Hinputs] with Hinputs
  · wp_start_folded as ⟨Hview, Hfields, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Hreads, Htable, Hstrings, Hrows⟩
    wp_apply wp_symbol_cached receiver view fields child result vd fd hchildfields hcached hresult
      $$ [Hview Hfields] with ⟨Hview, Hfields⟩
    · iframe
    wp_end
  · iexact Hinputs
  wp_end

theorem wp_symbol_parentheses_zero_child (receiver : Loc) (view : graphView) (graph : pdg.Graph)
    (fields : List w32) (nodes : List pdg.Node) (node : pdg.Node)
    (records : List pdg.AttributeSet) (a : pdg.AttributeSet)
    (references accesses extensions : List GoSlice) (values : List w32) (reads : List pdg.Access)
    (table : pdg.Tables) (strings : List GoString)
    (rows : List (GoArray w32 2)) (row : GoArray w32 2) (child : w32)
    (ref : w32) (vd gd fd rd td od : DFrac)
    (hf : 0 < uint.Z ref ∧ uint.Z ref ≤ fields.length)
    (hfield : fields[(uint.Z ref - 1).toNat]? = some (W32 0))
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ nodes.length)
    (hnode : nodes[(uint.Z ref - 1).toNat]? = some node)
    (ha : 0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ records.length)
    (hrecord : records[(uint.Z node.Attributes' - 1).toNat]? = some a) (halloc : values.length ≤ 2^46)
    (hnotdirect : directReadValue node reads = false) (hstatus : a.Operator'.Status' = 1#8)
    (hop : pdg.tableText strings a.Operator'.Value' = go!"parentheses")
    (hoperand : 0 < uint.Z ref ∧ uint.Z ref ≤ rows.length)
    (hrow : rows[(uint.Z ref - 1).toNat]? = some row) (hchildref : row.arr[0]? = some child)
    (hzero : child = W32 0) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{vd} view ∗ view.fields' ↦*{fd} fields ∗
       view.graph' ↦{gd} graph ∗ graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records ∗
       graph.AttributeTables'.References' ↦*{gd} references ∗ graph.AttributeTables'.Accesses' ↦*{gd} accesses ∗
       graph.AttributeTables'.Extensions' ↦*{gd} extensions ∗ pdg.listRow references a.Outcomes' ↦*{gd} values ∗
       pdg.listRow accesses a.Reads' ↦*{rd} reads ∗ graph.Tables' ↦{td} table ∗ table.Strings' ↦*{td} strings ∗ view.operands' ↦*{od} rows) }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"symbol" #receiver)) (Val #ref))
    {{ RET #(W32 0);
       receiver ↦{vd} view ∗ view.fields' ↦*{fd} fields ∗ view.graph' ↦{gd} graph ∗
       graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records ∗
       graph.AttributeTables'.References' ↦*{gd} references ∗ graph.AttributeTables'.Accesses' ↦*{gd} accesses ∗
       graph.AttributeTables'.Extensions' ↦*{gd} extensions ∗ pdg.listRow references a.Outcomes' ↦*{gd} values ∗
       pdg.listRow accesses a.Reads' ↦*{rd} reads ∗ graph.Tables' ↦{td} table ∗ table.Strings' ↦*{td} strings ∗ view.operands' ↦*{od} rows }} := by
  subst child
  wp_start_folded as Hinputs
  wp_apply wp_symbol_parentheses receiver view graph fields nodes node records a references accesses extensions values reads table strings rows row (W32 0) (W32 0) ref vd gd fd rd td od hf hfield href hnode ha hrecord halloc hnotdirect hstatus hop hoperand hrow hchildref
    $$ [Hinputs] with Hinputs
  · wp_start_folded as Hinputs
    wp_apply wp_symbol_zero receiver
    wp_end
  · iexact Hinputs
  wp_end

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
