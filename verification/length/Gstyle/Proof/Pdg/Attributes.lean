module
public import Gstyle.Proof.Pdg.GraphLayout
public import Gstyle.Proof.Pdg.List
public import Gstyle.Proof.Pdg.Outcomes
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.pdg
def expandedAttributes [FfiSyntax] (a : AttributeSet) (references accesses extensions : List GoSlice)
    (outcomes : GoSlice) : Attributes :=
  { Symbols' := listRow references a.Symbols', Defines' := listRow references a.Defines',
    Reads' := listRow accesses a.Reads', Writes' := listRow accesses a.Writes',
    Operator' := a.Operator', Literal' := a.Literal', Callee' := a.Callee',
    Construct' := a.Construct', WriteForm' := a.WriteForm', Description' := a.Description',
    Position' := a.Position', CallForm' := a.CallForm', ExitKind' := a.ExitKind',
    Extensions' := listRow extensions a.Extensions', Outcomes' := outcomes }

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [query.Assumptions]
variable [Graph.TypeAssumptions] [Node.TypeAssumptions] [Attributes.TypeAssumptions]
variable [AttributeSet.TypeAssumptions] [AttributeTables.TypeAssumptions]
variable [Producer.TypeAssumptions] [Function.TypeAssumptions] [Span.TypeAssumptions]
variable [CountFact.TypeAssumptions] [Conformance.TypeAssumptions] [Fact.TypeAssumptions]
variable [Position.TypeAssumptions] [CallForm.TypeAssumptions] [ExitKind.TypeAssumptions]
variable [ConformanceStatus.TypeAssumptions] [Category.TypeAssumptions]
variable [FuncUnfold listAt [Ref.ty] (listAt.impl Ref.ty)]
variable [FuncUnfold listAt [Access.ty] (listAt.impl Access.ty)]
variable [FuncUnfold listAt [Extension.ty] (listAt.impl Extension.ty)]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.pdg

omit [Node.TypeAssumptions] [Category.TypeAssumptions] in
theorem wp_expand (receiver : Loc) (graph : Graph) (a : AttributeSet)
    (references accesses extensions : List GoSlice) (values : List w32) (dq : DFrac)
    (halloc : values.length ≤ 2^46) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗
       (receiver ↦{dq} graph ∗ graph.AttributeTables'.References' ↦*{dq} references ∗
        graph.AttributeTables'.Accesses' ↦*{dq} accesses ∗ graph.AttributeTables'.Extensions' ↦*{dq} extensions ∗
        listRow references a.Outcomes' ↦*{dq} values) }}
      (App (Val #(methods (go.PointerType Graph.ty) go!"expand" #receiver)) (Val #a))
    {{ (out : GoSlice), RET #(expandedAttributes a references accesses extensions out);
       receiver ↦{dq} graph ∗ graph.AttributeTables'.References' ↦*{dq} references ∗
       graph.AttributeTables'.Accesses' ↦*{dq} accesses ∗ graph.AttributeTables'.Extensions' ↦*{dq} extensions ∗
       listRow references a.Outcomes' ↦*{dq} values ∗ out ↦* values ∗ ⌜values = [] → out = slice.nil⌝ }} := by
  wp_start as ⟨Hgraph, Hreferences, Haccesses, Hextensions, Hvalues⟩
  try wp_auto
  wp_apply wp_listAt Ref.ty graph.AttributeTables'.References' references dq a.Symbols' $$ [Hreferences] with Hreferences
  · iframe
  try wp_auto
  wp_apply wp_listAt Ref.ty graph.AttributeTables'.References' references dq a.Defines' $$ [Hreferences] with Hreferences
  · iframe
  try wp_auto
  wp_apply wp_listAt Access.ty graph.AttributeTables'.Accesses' accesses dq a.Reads' $$ [Haccesses] with Haccesses
  · iframe
  try wp_auto
  wp_apply wp_listAt Access.ty graph.AttributeTables'.Accesses' accesses dq a.Writes' $$ [Haccesses] with Haccesses
  · iframe
  try wp_auto
  wp_apply wp_listAt Extension.ty graph.AttributeTables'.Extensions' extensions dq a.Extensions' $$ [Hextensions] with Hextensions
  · iframe
  try wp_auto
  wp_apply wp_outcomes receiver graph references a.Outcomes' values dq halloc $$ [Hgraph Hreferences Hvalues] with %out ⟨Hgraph, Hreferences, Hvalues, Hout, %Hempty⟩
  · iframe
  try wp_auto
  unfold expandedAttributes
  wp_end

omit [AttributeSet.TypeAssumptions] [AttributeTables.TypeAssumptions]
  [Producer.TypeAssumptions] [Function.TypeAssumptions] [Span.TypeAssumptions]
  [CountFact.TypeAssumptions] [Conformance.TypeAssumptions] [ConformanceStatus.TypeAssumptions]
  [FuncUnfold listAt [Ref.ty] (listAt.impl Ref.ty)]
  [FuncUnfold listAt [Access.ty] (listAt.impl Access.ty)]
  [FuncUnfold listAt [Extension.ty] (listAt.impl Extension.ty)] in
theorem wp_NodeAttributes_zero (receiver : Loc) (node : Node)
    (hz : node.Attributes' = 0#32) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (Val #(methods (go.PointerType Graph.ty) go!"NodeAttributes" #receiver)) (Val #node))
    {{ RET #(zero_val Attributes); True }} := by
  wp_start
  try wp_auto
  wp_if_destruct
  · wp_end
  · exfalso; simp_all

omit [FuncUnfold listAt [Ref.ty] (listAt.impl Ref.ty)]
  [FuncUnfold listAt [Access.ty] (listAt.impl Access.ty)]
  [FuncUnfold listAt [Extension.ty] (listAt.impl Extension.ty)] in
theorem wp_NodeAttributes_absent (receiver : Loc) (graph : Graph) (node : Node)
    (records : List AttributeSet) (dq : DFrac)
    (hv : ¬ (0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ records.length)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{dq} graph ∗ graph.Attributes' ↦*{dq} records) }}
      (App (Val #(methods (go.PointerType Graph.ty) go!"NodeAttributes" #receiver)) (Val #node))
    {{ RET #(zero_val Attributes); receiver ↦{dq} graph ∗ graph.Attributes' ↦*{dq} records }} := by
  wp_start as ⟨Hgraph, Hrecords⟩
  ihave %Hlen := ownSlice_len $$ Hrecords
  try wp_auto
  wp_if_destruct
  · wp_end
  · wp_if_destruct
    · wp_end
    · exfalso
      apply hv
      constructor <;> word
theorem wp_NodeAttributes_present (receiver : Loc) (graph : Graph) (node : Node)
    (records : List AttributeSet) (a : AttributeSet)
    (references accesses extensions : List GoSlice) (values : List w32) (dq : DFrac)
    (hv : 0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ records.length)
    (hlookup : records[(uint.Z node.Attributes' - 1).toNat]? = some a)
    (halloc : values.length ≤ 2^46) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗
       (receiver ↦{dq} graph ∗ graph.Attributes' ↦*{dq} records ∗
        graph.AttributeTables'.References' ↦*{dq} references ∗
        graph.AttributeTables'.Accesses' ↦*{dq} accesses ∗ graph.AttributeTables'.Extensions' ↦*{dq} extensions ∗
        listRow references a.Outcomes' ↦*{dq} values) }}
      (App (Val #(methods (go.PointerType Graph.ty) go!"NodeAttributes" #receiver)) (Val #node))
    {{ (out : GoSlice), RET #(expandedAttributes a references accesses extensions out);
       receiver ↦{dq} graph ∗ graph.Attributes' ↦*{dq} records ∗
       graph.AttributeTables'.References' ↦*{dq} references ∗
       graph.AttributeTables'.Accesses' ↦*{dq} accesses ∗ graph.AttributeTables'.Extensions' ↦*{dq} extensions ∗
       listRow references a.Outcomes' ↦*{dq} values ∗ out ↦* values ∗ ⌜values = [] → out = slice.nil⌝ }} := by
  wp_start as ⟨Hgraph, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues⟩
  ihave %Hlen := ownSlice_len $$ Hrecords
  try wp_auto
  repeat (wp_if_destruct <;> try (exfalso; obtain ⟨_, _⟩ := hv; word))
  have Hi : sint.Z (W64 (uint.Z (node.Attributes' - W32 1))) = uint.Z node.Attributes' - 1 := by obtain ⟨_, _⟩ := hv; word
  rw [Hi]
  rw [ite_eq_left (by obtain ⟨_, _⟩ := hv; constructor <;> word : 0 ≤ uint.Z node.Attributes' - 1 ∧ uint.Z node.Attributes' - 1 < sint.Z graph.Attributes'.len)]
  wp_apply wp_load_slice_index graph.Attributes' (uint.Z node.Attributes' - 1) records dq a (by obtain ⟨_, _⟩ := hv; word) $$ [Hrecords] with Hrecords
  · iframe; ipureintro; exact hlookup
  try wp_auto
  wp_apply wp_expand receiver graph a references accesses extensions values dq halloc $$ [Hgraph Hreferences Haccesses Hextensions Hvalues] with %out ⟨Hgraph, Hreferences, Haccesses, Hextensions, Hvalues, Hout, %Hempty⟩
  · iframe
  wp_end

end proof
end github_com.rstyled.gstyle.domain.pdg
end Perennial
