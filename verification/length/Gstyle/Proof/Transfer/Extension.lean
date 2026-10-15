module
public import Gstyle.Proof.Pdg.GraphLayout
public import Gstyle.Proof.Pdg.ExtensionLayout
public import Gstyle.Proof.Pdg.Text
public import Gstyle.Proof.RuntimePackage
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

def extensionValue [FfiSyntax] (strings : List GoString) (name : GoString) : List pdg.Extension → GoString
  | [] => go!""
  | value :: rest =>
    if pdg.tableText strings value.Name' = go!"gstyle.runtime:" ++ name then
      pdg.tableText strings value.Value'
    else extensionValue strings name rest

theorem extensionValue_first [FfiSyntax] (strings : List GoString) (name : GoString)
    (entry : pdg.Extension) (rest : List pdg.Extension)
    (hname : pdg.tableText strings entry.Name' = go!"gstyle.runtime:" ++ name) :
    extensionValue strings name (entry :: rest) = pdg.tableText strings entry.Value' := by
  simp [extensionValue, hname]

theorem extensionValue_missing [FfiSyntax] (strings : List GoString) (name : GoString)
    (records : List pdg.Extension)
    (hmissing : ∀ entry ∈ records, pdg.tableText strings entry.Name' ≠ go!"gstyle.runtime:" ++ name) :
    extensionValue strings name records = go!"" := by
  induction records with
  | nil => rfl
  | cons entry rest ih =>
    have hn := hmissing entry (by simp)
    rw [extensionValue, ite_eq_right hn]
    apply ih
    intro value hv
    exact hmissing value (by simp [hv])

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [pdg.query.Assumptions] [pdg.Extension.TypeAssumptions] [pdg.Tables.TypeAssumptions]
variable [pdg.Graph.TypeAssumptions] [pdg.AttributeTables.TypeAssumptions]
variable [pdg.Producer.TypeAssumptions] [pdg.Function.TypeAssumptions] [pdg.Span.TypeAssumptions]
variable [pdg.CountFact.TypeAssumptions] [pdg.Conformance.TypeAssumptions] [pdg.Fact.TypeAssumptions]
variable [pdg.ConformanceStatus.TypeAssumptions]
variable [FuncUnfold extension [] extension.impl]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

theorem wp_extension (receiver : Loc) (graph : pdg.Graph) (table : pdg.Tables)
    (strings : List GoString) (values : GoSlice) (records : List pdg.Extension) (name : GoString)
    (gd td ed : DFrac) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗
       (receiver ↦{gd} graph ∗ graph.Tables' ↦{td} table ∗ table.Strings' ↦*{td} strings ∗ values ↦*{ed} records) }}
      (App (App (App (Val (@! extension)) (Val #receiver)) (Val #values)) (Val #name))
    {{ RET #(extensionValue strings name records);
       receiver ↦{gd} graph ∗ graph.Tables' ↦{td} table ∗ table.Strings' ↦*{td} strings ∗ values ↦*{ed} records }} := by
  wp_start as ⟨Hgraph, Htable, Hstrings, Hvalues⟩
  ihave %Hlen := ownSlice_len $$ Hvalues
  try wp_auto
  ihave HI : (∃ (iv : w64) (previous : pdg.Extension),
      "i" ∷ i_ptr ↦ iv ∗
      "value" ∷ value_ptr ↦ previous ∗
      "Hgraph" ∷ receiver ↦{gd} graph ∗
      "Htable" ∷ graph.Tables' ↦{td} table ∗
      "Hstrings" ∷ table.Strings' ↦*{td} strings ∗
      "Hvalues" ∷ values ↦*{ed} records ∗
      "%Hbounds" ∷ ⌜0 ≤ sint.Z iv ∧ sint.Z iv ≤ sint.Z values.len⌝ ∗
      "%Hrest" ∷ ⌜extensionValue strings name (records.drop (sint.nat iv)) = extensionValue strings name records⌝ : IProp GF)
      $$ [i value Hgraph Htable Hstrings Hvalues]
  · iexists (W64 0), (zero_val pdg.Extension)
    simp only [show sint.nat (W64 0) = 0 from rfl, List.drop_zero]
    iframe
    ipureintro
    constructor <;> word
  wp_for HI
  wp_if_destruct
  · have Hindex : sint.nat iv < records.length := by word
    list_elem records (sint.nat iv) as entry
    have Hdrop : records.drop (sint.nat iv) = entry :: records.drop (sint.nat iv + 1) := by
      rw [List.drop_eq_getElem?_toList_append, Hentry_lookup]
      rfl
    rw [Hdrop, extensionValue] at Hrest
    rw [ite_eq_left (by constructor <;> word : 0 ≤ sint.Z iv ∧ sint.Z iv < sint.Z values.len)]
    wp_apply wp_load_slice_index values (sint.Z iv) records ed entry (by word) $$ [Hvalues] with Hvalues
    · iframe; ipureintro; exact Hentry_lookup
    try wp_auto
    wp_apply pdg.wp_Text graph.Tables' table strings td entry.Name' $$ [Htable Hstrings] with ⟨Htable, Hstrings⟩
    · iframe
    try wp_auto
    wp_if_destruct
    · rw [ite_eq_left Hif] at Hrest
      wp_apply pdg.wp_Text graph.Tables' table strings td entry.Value' $$ [Htable Hstrings] with ⟨Htable, Hstrings⟩
      · iframe
      wp_for_post
      rw [← Hrest]
      wp_end
    · rw [ite_eq_right Hif] at Hrest
      wp_for_post
      iframe
      iexists (iv + W64 1), entry
      iframe
      ipureintro
      constructor
      · constructor <;> word
      · have hi : sint.nat (iv + W64 1) = sint.nat iv + 1 := by word
        rw [hi]
        exact Hrest
  · have hi : sint.nat iv = records.length := by word
    rw [hi, List.drop_length, extensionValue] at Hrest
    rw [← Hrest]
    wp_end
omit [pdg.Tables.TypeAssumptions] [pdg.Graph.TypeAssumptions] [pdg.AttributeTables.TypeAssumptions]
  [pdg.Producer.TypeAssumptions] [pdg.Function.TypeAssumptions] [pdg.Span.TypeAssumptions]
  [pdg.CountFact.TypeAssumptions] [pdg.Conformance.TypeAssumptions] [pdg.Fact.TypeAssumptions]
  [pdg.ConformanceStatus.TypeAssumptions] in
theorem wp_extension_empty (receiver : Loc) (values : GoSlice) (name : GoString) (ed : DFrac) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ values ↦*{ed} ([] : List pdg.Extension) }}
      (App (App (App (Val (@! extension)) (Val #receiver)) (Val #values)) (Val #name))
    {{ RET #(go!""); values ↦*{ed} ([] : List pdg.Extension) }} := by
  wp_start as Hvalues
  ihave %Hlen := ownSlice_len $$ Hvalues
  try wp_auto
  ihave HI : ("i" ∷ i_ptr ↦ (W64 0) ∗ "value" ∷ value_ptr ↦ zero_val pdg.Extension ∗
      "Hvalues" ∷ values ↦*{ed} ([] : List pdg.Extension) : IProp GF) $$ [i value Hvalues]
  · iframe
  wp_for HI
  wp_if_destruct
  · exfalso; simp only [List.length_nil] at Hlen; word
  · wp_end

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
