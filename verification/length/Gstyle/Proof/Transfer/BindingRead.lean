module
public import Gstyle.Proof.Pdg.AccessLayout
public import Gstyle.Proof.Transfer.Facts
public import Perennial.Golang.Theory.Array
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

def directReadValue [FfiSyntax] (node : pdg.Node) (reads : List pdg.Access) : Bool :=
  match reads with
  | [entry] => decide (node.Category' = (W8 1) ∧ entry.Location' = W32 0)
  | _ => false

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [pdg.query.Assumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard
section direct
variable [pdg.Node.TypeAssumptions] [pdg.Attributes.TypeAssumptions]
variable [pdg.Producer.TypeAssumptions] [pdg.Function.TypeAssumptions] [pdg.Span.TypeAssumptions]
variable [pdg.CountFact.TypeAssumptions] [pdg.Conformance.TypeAssumptions] [pdg.Fact.TypeAssumptions]
variable [pdg.Position.TypeAssumptions] [pdg.CallForm.TypeAssumptions] [pdg.ExitKind.TypeAssumptions]
variable [pdg.ConformanceStatus.TypeAssumptions] [pdg.Category.TypeAssumptions]
variable [FuncUnfold directBindingRead [] directBindingRead.impl]

theorem wp_directBindingRead_other (node : pdg.Node) (attrs : pdg.Attributes)
    (hc : node.Category' ≠ (W8 1)) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! directBindingRead)) (Val #node)) (Val #attrs))
    {{ RET #false; True }} := by
  wp_start
  try wp_auto
  rw [decide_eq_false hc]
  try wp_auto
  wp_end

theorem wp_directBindingRead_arity (node : pdg.Node) (attrs : pdg.Attributes)
    (hn : attrs.Reads'.len ≠ W64 1) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! directBindingRead)) (Val #node)) (Val #attrs))
    {{ RET #false; True }} := by
  wp_start
  try wp_auto
  by_cases hc : node.Category' = (W8 1)
  · rw [decide_eq_true hc]
    try wp_auto
    rw [decide_eq_false hn]
    try wp_auto
    wp_end
  · rw [decide_eq_false hc]
    try wp_auto
    wp_end

variable [pdg.Access.TypeAssumptions]

theorem wp_directBindingRead_single (node : pdg.Node) (attrs : pdg.Attributes)
    (entry : pdg.Access) (dq : DFrac) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ attrs.Reads' ↦*{dq} [entry] }}
      (App (App (Val (@! directBindingRead)) (Val #node)) (Val #attrs))
    {{ RET #(directReadValue node [entry]); attrs.Reads' ↦*{dq} [entry] }} := by
  wp_start as Hreads
  ihave %Hlen := ownSlice_len $$ Hreads
  have hlen : attrs.Reads'.len = W64 1 := by simp only [List.length_singleton] at Hlen; word
  try wp_auto
  by_cases hc : node.Category' = (W8 1)
  · rw [decide_eq_true hc]
    try wp_auto
    rw [hlen]
    try wp_auto
    rw [show sint.Z (W64 0) = 0 from rfl]
    rw [ite_eq_left (by simp [hlen] : (0 : Int) ≤ 0 ∧ 0 < sint.Z attrs.Reads'.len)]
    icases ownSlice_elem_acc 0 entry attrs.Reads' dq [entry] (by omega) (by rfl) $$ Hreads with ⟨Hentry, Hreads⟩
    try wp_auto
    ihave Hreads := Hreads $$ %entry [Hentry]
    · iexact Hentry
    rw [list_set_lookup_self [entry] (Int.toNat (0 : Int)) entry (by rfl)]
    have hr : directReadValue node [entry] = decide (entry.Location' = W32 0) := by simp only [directReadValue, hc, _root_.true_and]
    rw [hr]
    wp_end
  · rw [decide_eq_false hc]
    try wp_auto
    have hr : directReadValue node [entry] = false := by simp only [directReadValue, hc, _root_.false_and, _root_.decide_false]
    rw [hr]
    wp_end

theorem wp_directBindingRead (node : pdg.Node) (attrs : pdg.Attributes)
    (reads : List pdg.Access) (dq : DFrac) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ attrs.Reads' ↦*{dq} reads }}
      (App (App (Val (@! directBindingRead)) (Val #node)) (Val #attrs))
    {{ RET #(directReadValue node reads); attrs.Reads' ↦*{dq} reads }} := by
  wp_start_folded as Hreads
  cases reads with
  | nil =>
    ihave %Hlen := ownSlice_len $$ Hreads
    have hn : attrs.Reads'.len ≠ W64 1 := by
      intro h
      have hm := Hlen.1
      rw [h] at hm
      change 0 = 1 at hm
      omega
    wp_apply wp_directBindingRead_arity node attrs hn
    simp only [directReadValue]
    wp_end
  | cons entry tail =>
    cases tail with
    | nil =>
      wp_apply wp_directBindingRead_single node attrs entry dq $$ [Hreads] with Hreads
      · iframe
      wp_end
    | cons second rest =>
      ihave %Hlen := ownSlice_len $$ Hreads
      have hn : attrs.Reads'.len ≠ W64 1 := by
        intro h
        have hm := Hlen.1
        rw [h] at hm
        change (entry :: second :: rest).length = 1 at hm
        simp only [List.length_cons] at hm
        omega
      wp_apply wp_directBindingRead_arity node attrs hn
      simp only [directReadValue]
      wp_end

end direct

variable [graphView.TypeAssumptions]

theorem wp_operand_readonly (receiver : Loc) (view : graphView) (rows : List (GoArray w32 2))
    (ref : w32) (position : w64) (row : GoArray w32 2) (value : w32)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ rows.length)
    (hp : 0 ≤ sint.Z position ∧ sint.Z position < 2)
    (hrow : rows[(uint.Z ref - 1).toNat]? = some row)
    (hvalue : row.arr[(sint.Z position).toNat]? = some value) (vd rd : DFrac) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{vd} view ∗ view.operands' ↦*{rd} rows) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"operand" #receiver)) (Val #ref)) (Val #position))
    {{ RET #value; receiver ↦{vd} view ∗ view.operands' ↦*{rd} rows }} := by
  wp_start as ⟨Hview, Hrows⟩
  ihave %Hlen := ownSlice_len $$ Hrows
  try wp_auto
  repeat (wp_if_destruct <;> try (exfalso; obtain ⟨_, _⟩ := href; obtain ⟨_, _⟩ := hp; word))
  have Hi : sint.Z (W64 (uint.Z (ref - W32 1))) = uint.Z ref - 1 := by obtain ⟨_, _⟩ := href; word
  rw [Hi]
  rw [ite_eq_left (by obtain ⟨_, _⟩ := href; constructor <;> word : 0 ≤ uint.Z ref - 1 ∧ uint.Z ref - 1 < sint.Z view.operands'.len)]
  icases ownSlice_elem_acc (uint.Z ref - 1) row view.operands' rd rows (by omega) hrow $$ Hrows with ⟨Hrow, Hrows⟩
  try wp_auto
  rw [ite_eq_left hp]
  icases array_acc (sliceIndexRef (GoArray w32 2) (uint.Z ref - 1) view.operands') (sint.Z position) rd 2 row value hp.1 hvalue $$ Hrow with ⟨Hvalue, Hrow⟩
  try wp_auto
  ihave Hrow := Hrow $$ %value [Hvalue]
  · iexact Hvalue
  rw [list_set_lookup_self row.arr (sint.Z position).toNat value hvalue]
  ihave Hrows := Hrows $$ %row [Hrow]
  · iexact Hrow
  rw [list_set_lookup_self rows (uint.Z ref - 1).toNat row hrow]
  wp_end

theorem wp_symbol_zero (receiver : Loc) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"symbol" #receiver)) (Val #(W32 0)))
    {{ RET #(W32 0); True }} := by
  wp_start
  try wp_auto
  wp_end

theorem wp_symbol_cached (receiver : Loc) (view : graphView) (fields : List w32)
    (ref cached : w32) (vd fd : DFrac)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ fields.length)
    (hfield : fields[(uint.Z ref - 1).toNat]? = some cached) (hc : cached ≠ W32 0) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{vd} view ∗ view.fields' ↦*{fd} fields) }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"symbol" #receiver)) (Val #ref))
    {{ RET #cached; receiver ↦{vd} view ∗ view.fields' ↦*{fd} fields }} := by
  wp_start as ⟨Hview, Hfields⟩
  ihave %Hlen := ownSlice_len $$ Hfields
  try wp_auto
  wp_if_destruct
  · exfalso; obtain ⟨_, _⟩ := href; word
  · have Hi : sint.Z (W64 (uint.Z (ref - W32 1))) = uint.Z ref - 1 := by obtain ⟨_, _⟩ := href; word
    rw [Hi]
    have hbound : 0 ≤ uint.Z ref - 1 ∧ uint.Z ref - 1 < sint.Z view.fields'.len := by
      obtain ⟨_, _⟩ := href; word
    rw [ite_eq_left hbound]
    wp_apply wp_load_slice_index view.fields' (uint.Z ref - 1) fields fd cached (by omega) $$ [Hfields] with Hfields
    · iframe; ipureintro; exact hfield
    try wp_auto
    wp_if_destruct
    · exfalso; contradiction
    · rw [Hi, ite_eq_left hbound]
      wp_apply wp_load_slice_index view.fields' (uint.Z ref - 1) fields fd cached (by omega) $$ [Hfields] with Hfields
      · iframe; ipureintro; exact hfield
      try wp_auto
      wp_end

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
