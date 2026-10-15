module
public import Gstyle.Proof.Pdg.GraphLayout
public import Gstyle.Proof.Pdg.List
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.pdg
def filledPrefix (values : List w32) (i : Nat) : List w32 :=
  values.take i ++ List.replicate (values.length - i) (0#32)

theorem filledPrefix_step (values : List w32) (i : Nat) (value : w32)
    (hi : i < values.length) (hv : values[i]? = some value) :
    (filledPrefix values i).set i value = filledPrefix values (i+1) := by
  unfold filledPrefix
  have ht : (values.take i).length = i := by simp [Nat.min_eq_left (by omega : i ≤ values.length)]
  rw [List.set_append_right i value (by omega), ht, Nat.sub_self]
  have hr : values.length - i = (values.length - (i+1)) + 1 := by omega
  rw [hr, List.replicate_succ, List.set_cons_zero, List.take_add_one, hv]
  simp [List.append_assoc]

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [query.Assumptions]
variable [Graph.TypeAssumptions]
variable [AttributeTables.TypeAssumptions]
variable [Producer.TypeAssumptions] [Function.TypeAssumptions] [Span.TypeAssumptions]
variable [CountFact.TypeAssumptions] [Conformance.TypeAssumptions] [Fact.TypeAssumptions]
variable [ConformanceStatus.TypeAssumptions]
variable [FuncUnfold listAt [Ref.ty] (listAt.impl Ref.ty)]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.pdg

theorem wp_outcomes (receiver : Loc) (graph : Graph) (rows : List GoSlice)
    (ref : w32) (values : List w32) (dq : DFrac)
    (halloc : values.length ≤ 2^46) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗
       (receiver ↦{dq} graph ∗ graph.AttributeTables'.References' ↦*{dq} rows ∗
        listRow rows ref ↦*{dq} values) }}
      (App (Val #(methods (go.PointerType Graph.ty) go!"outcomes" #receiver)) (Val #ref))
    {{ (out : GoSlice), RET #out;
       receiver ↦{dq} graph ∗ graph.AttributeTables'.References' ↦*{dq} rows ∗
       listRow rows ref ↦*{dq} values ∗ out ↦* values ∗ ⌜values = [] → out = slice.nil⌝ }} := by
  wp_start as ⟨Hgraph, Hrows, Hvalues⟩
  try wp_auto
  wp_apply wp_listAt Ref.ty graph.AttributeTables'.References' rows dq ref $$ [Hrows] with Hrows
  · iframe
  try wp_auto
  ihave %Hlen := ownSlice_len $$ Hvalues
  wp_if_destruct
  · have hz : values = [] := by apply List.eq_nil_of_length_eq_zero; word
    subst values
    wp_end
    iframe
    isplitr
    · iapply ownSlice_nil
    ipureintro; intro _; rfl
  · wp_apply wp_slice_make2 (V := w32) (listRow rows ref).len (by
      intro _
      simp only [go.typeSize_w32]
      word) $$ [] with %out ⟨Hout, Hcap⟩
    · ipureintro; exact Hlen.2
    try wp_auto
    rename_i body_i_ptr
    irename : (body_i_ptr ↦ zero_val w64 : IProp GF) => body_i
    ihave %Houtlen := ownSlice_len $$ Hout
    simp only [List.length_replicate] at Houtlen
    have Hsize : sint.Z out.len = sint.Z (listRow rows ref).len := by word
    ihave HI : (∃ (iv bv : w64) (vv : w32),
        "i" ∷ i_ptr ↦ iv ∗
        "body_i" ∷ body_i_ptr ↦ bv ∗
        "value" ∷ value_ptr ↦ vv ∗
        "Hvalues" ∷ listRow rows ref ↦*{dq} values ∗
        "Hout" ∷ out ↦* filledPrefix values (sint.nat iv) ∗
        "%Hbounds" ∷ ⌜0 ≤ sint.Z iv ∧ sint.Z iv ≤ sint.Z (listRow rows ref).len⌝ : IProp GF)
        $$ [i body_i value Hvalues Hout]
    · iexists (W64 0), (zero_val w64), (zero_val w32)
      simp only [show sint.nat (W64 0) = 0 from rfl, filledPrefix, List.take_zero, Nat.sub_zero, List.nil_append]
      rw [Hlen.1]
      iframe
      ipureintro
      constructor <;> word
    wp_for HI
    wp_if_destruct
    · have Hindex : sint.nat iv < values.length := by word
      list_elem values (sint.nat iv) as entry
      rw [ite_eq_left (by constructor <;> word : 0 ≤ sint.Z iv ∧ sint.Z iv < sint.Z (listRow rows ref).len)]
      wp_apply wp_load_slice_index (listRow rows ref) (sint.Z iv) values dq entry (by word) $$ [Hvalues] with Hvalues
      · iframe; ipureintro; exact Hentry_lookup
      try wp_auto
      rw [ite_eq_left (by constructor <;> word : 0 ≤ sint.Z iv ∧ sint.Z iv < sint.Z out.len)]
      wp_apply wp_store_slice_index out (sint.Z iv) (filledPrefix values (sint.nat iv)) entry $$ [Hout] with Hout
      · iframe
        ipureintro
        simp only [filledPrefix, List.length_append, List.length_take, List.length_replicate]
        constructor <;> word
      wp_for_post
      iframe
      iexists (iv + W64 1), iv, entry
      have hi : sint.nat (iv + W64 1) = sint.nat iv + 1 := by word
      rw [hi, ← filledPrefix_step values (sint.nat iv) entry Hindex Hentry_lookup]
      iframe
      ipureintro
      constructor <;> word
    · have hi : sint.nat iv = values.length := by word
      rw [hi, filledPrefix, List.take_length, Nat.sub_self, List.replicate_zero, List.append_nil]
      wp_end
      iframe Hgraph Hrows Hvalues
      isplitl [Hout]
      · iapply Hout
      ipureintro
      intro hz
      subst values
      simp at Hlen
      obtain ⟨hzero, hnonneg⟩ := Hlen
      exfalso
      apply Hif
      word
end proof
end github_com.rstyled.gstyle.domain.pdg
end Perennial
