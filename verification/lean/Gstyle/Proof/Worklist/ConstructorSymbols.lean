module
public import Gstyle.Proof.Worklist.ConstructorSyntax
public import Gstyle.Proof.Worklist.ConstructorLayout
public import Gstyle.Proof.Transfer.Extension
public import Gstyle.Proof.Pdg.ReadonlyRows
@[expose] public section
noncomputable section
set_option maxHeartbeats 0
set_option maxRecDepth 100000
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

def constructorPrefix {A : Type} (values : List A) (fill : A) (count : Nat) : List A :=
  values.take count ++ List.replicate (values.length - count) fill

theorem constructorPrefix_step {A : Type} (values : List A) (fill value : A) (i : Nat)
    (hi : i < values.length) (hv : values[i]? = some value) :
    (constructorPrefix values fill i).set i value = constructorPrefix values fill (i + 1) := by
  unfold constructorPrefix
  rw [List.set_append_right i value (by simp; omega)]
  have htake : (values.take i).length = i := by simp; omega
  rw [htake, Nat.sub_self]
  have hrem : values.length - i = values.length - (i + 1) + 1 := by omega
  rw [hrem, List.replicate_succ]
  simp only [List.set_cons_zero]
  rw [List.take_add_one, hv]
  simp

section execution
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [pdg.query.Assumptions] [graphView.TypeAssumptions]
variable [pdg.Graph.TypeAssumptions] [pdg.Node.TypeAssumptions]
variable [pdg.AttributeSet.TypeAssumptions] [pdg.AttributeTables.TypeAssumptions]
variable [pdg.Producer.TypeAssumptions] [pdg.Function.TypeAssumptions] [pdg.Span.TypeAssumptions]
variable [pdg.NormalFlow.TypeAssumptions] [pdg.CountFact.TypeAssumptions]
variable [pdg.Conformance.TypeAssumptions] [pdg.Fact.TypeAssumptions]
variable [pdg.CallForm.TypeAssumptions] [pdg.ExitKind.TypeAssumptions]
variable [pdg.ConformanceStatus.TypeAssumptions] [pdg.Category.TypeAssumptions]
variable [pdg.Tables.TypeAssumptions] [pdg.Symbol.TypeAssumptions] [pdg.Role.TypeAssumptions]
variable [pdg.Extension.TypeAssumptions] [FuncUnfold extension [] extension.impl]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

def constructorKinds (strings : List GoString) (symbols : List pdg.Symbol)
    (entries : List (List pdg.Extension)) : List GoString :=
  symbols.mapIdx fun i _ => extensionValue strings (go!"type") (entries[i]?.getD [])

def constructorUnstable (strings : List GoString) (symbols : List pdg.Symbol)
    (entries : List (List pdg.Extension)) : List Bool :=
  symbols.mapIdx fun i symbol => decide (symbol.Role' = W8 2) ||
    decide (extensionValue strings (go!"unstable") (entries[i]?.getD []) = go!"true")

omit [pdg.Node.TypeAssumptions] [pdg.AttributeSet.TypeAssumptions] [pdg.NormalFlow.TypeAssumptions]
  [pdg.CallForm.TypeAssumptions] [pdg.ExitKind.TypeAssumptions] [pdg.Category.TypeAssumptions] in
theorem wp_constructor_symbols_raw (graphLocal viewLocal receiver : Loc) (view : graphView)
    (graph : pdg.Graph) (table : pdg.Tables) (strings : List GoString)
    (symbols : List pdg.Symbol) (entries : List (List pdg.Extension)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ graphLocal ↦ view.graph' ∗ viewLocal ↦ receiver ∗
       receiver ↦ view ∗ view.graph' ↦□ graph ∗ graph.Tables' ↦□ table ∗ table.Strings' ↦*□ strings ∗
       graph.Symbols' ↦*□ symbols ∗ pdg.readonlyRows (symbols.map pdg.Symbol.Extensions') entries ∗
       view.kinds' ↦* List.replicate symbols.length (go!"") ∗
       view.unstable' ↦* List.replicate symbols.length false }}
      (constructorSymbolsRawRun graphLocal viewLocal)
    {{ RET executeVal; graphLocal ↦ view.graph' ∗ viewLocal ↦ receiver ∗ receiver ↦ view ∗
       view.kinds' ↦* constructorKinds strings symbols entries ∗
       view.unstable' ↦* constructorUnstable strings symbols entries }} := by
  wp_start_folded as ⟨Hg, Hv, Hview, #Hgraph, #Htable, #Hstrings, #Hsymbols, #Hentries, Hkinds, Hunstable⟩
  ihave %Hslen := ownSlice_len $$ Hsymbols
  ihave %Hklen := ownSlice_len $$ Hkinds
  ihave %Hulen := ownSlice_len $$ Hunstable
  simp only [List.length_replicate] at Hklen Hulen
  unfold constructorSymbolsRawRun constructorSymbolRange constructorBody newGraphView.impl
  dsimp only [constructorRangeSite, Option.or, Option.getD]
  simp only [Let, Lam]
  try wp_auto
  rename_i body_i_ptr
  irename : (body_i_ptr ↦ zero_val w64 : IProp GF) => body_i
  ihave HI : (∃ (iv bv : w64) (previous : pdg.Symbol),
      "i" ∷ i_ptr ↦ iv ∗ "body_i" ∷ body_i_ptr ↦ bv ∗ "symbol" ∷ symbol_ptr ↦ previous ∗
      "Hg" ∷ graphLocal ↦ view.graph' ∗ "Hv" ∷ viewLocal ↦ receiver ∗ "Hview" ∷ receiver ↦ view ∗
      "Hkinds" ∷ view.kinds' ↦* constructorPrefix (constructorKinds strings symbols entries) (go!"") (sint.nat iv) ∗
      "Hunstable" ∷ view.unstable' ↦* constructorPrefix (constructorUnstable strings symbols entries) false (sint.nat iv) ∗
      "%Hbounds" ∷ ⌜0 ≤ sint.Z iv ∧ sint.Z iv ≤ sint.Z graph.Symbols'.len⌝ : IProp GF)
      $$ [i body_i symbol Hg Hv Hview Hkinds Hunstable]
  · iexists (W64 0), (zero_val w64), (zero_val pdg.Symbol)
    simp only [show sint.nat (W64 0) = 0 from rfl, constructorPrefix, List.take_zero,
      Nat.sub_zero, List.nil_append, constructorKinds, constructorUnstable, List.length_mapIdx]
    iframe
    ipureintro; constructor <;> word
  wp_for HI
  wp_if_destruct
  · have hi : sint.nat iv < symbols.length := by word
    list_elem symbols (sint.nat iv) as entry
    rw [ite_eq_left (by constructor <;> word : 0 ≤ sint.Z iv ∧ sint.Z iv < sint.Z graph.Symbols'.len)]
    wp_apply wp_load_slice_index graph.Symbols' (sint.Z iv) symbols DFrac.discard entry (by word)
      $$ [Hsymbols] with Hsymbols
    · iframe Hsymbols; ipureintro; exact Hentry_lookup
    have hmap : ((symbols.map pdg.Symbol.Extensions')[sint.nat iv]?).getD slice.nil = entry.Extensions' := by
      rw [List.getElem?_map, Hentry_lookup]; rfl
    ihave #Hextension : (entry.Extensions' ↦*□ (entries[sint.nat iv]?.getD []) : IProp GF) $$ [Hentries]
    · rw [← hmap]
      iapply pdg.readonlyRows_at (symbols.map pdg.Symbol.Extensions') entries (sint.nat iv) $$ Hentries
    try wp_auto
    wp_apply wp_extension view.graph' graph table strings entry.Extensions' (entries[sint.nat iv]?.getD [])
      (go!"type") DFrac.discard DFrac.discard DFrac.discard $$ [Hgraph Htable Hstrings Hextension]
      with ⟨Hgraph, Htable, Hstrings, Hextension⟩
    · iframe Hgraph Htable Hstrings Hextension
    try wp_auto
    rw [ite_eq_left (by constructor <;> word : 0 ≤ sint.Z iv ∧ sint.Z iv < sint.Z view.kinds'.len)]
    wp_apply wp_store_slice_index view.kinds' (sint.Z iv)
      (constructorPrefix (constructorKinds strings symbols entries) (go!"") (sint.nat iv))
      (extensionValue strings (go!"type") (entries[sint.nat iv]?.getD [])) $$ [Hkinds] with Hkinds
    · iframe; ipureintro
      simp only [constructorPrefix, List.length_append, List.length_take, List.length_replicate,
        constructorKinds, List.length_mapIdx]
      constructor <;> word
    try wp_auto
    have hkind : (constructorKinds strings symbols entries)[sint.nat iv]? =
        some (extensionValue strings (go!"type") (entries[sint.nat iv]?.getD [])) := by
      simp only [constructorKinds, List.getElem?_mapIdx, Hentry_lookup, Option.map_some]
    rw [constructorPrefix_step _ (go!"") _ _ (by simpa [constructorKinds] using hi) hkind]
    have hu : (constructorUnstable strings symbols entries)[sint.nat iv]? = some
        (decide (entry.Role' = W8 2) || decide (extensionValue strings (go!"unstable") (entries[sint.nat iv]?.getD []) = go!"true")) := by
      simp only [constructorUnstable, List.getElem?_mapIdx, Hentry_lookup, Option.map_some]
    by_cases hg : entry.Role' = W8 2
    · rw [decide_eq_true hg]
      try wp_auto
      rw [ite_eq_left (by constructor <;> word : 0 ≤ sint.Z iv ∧ sint.Z iv < sint.Z view.unstable'.len)]
      wp_apply wp_store_slice_index view.unstable' (sint.Z iv)
        (constructorPrefix (constructorUnstable strings symbols entries) false (sint.nat iv)) true
        $$ [Hunstable] with Hunstable
      · iframe; ipureintro
        simp only [constructorPrefix, List.length_append, List.length_take, List.length_replicate,
          constructorUnstable, List.length_mapIdx]
        constructor <;> word
      simp only [hg, decide_true, Bool.true_or] at hu
      rw [constructorPrefix_step _ false _ _ (by simpa [constructorUnstable] using hi) hu]
      wp_for_post
      iframe
      iexists (iv + W64 1), iv, entry
      have hn : sint.nat (iv + W64 1) = sint.nat iv + 1 := by word
      rw [hn]
      iframe
      ipureintro; constructor <;> word
    · rw [decide_eq_false hg]
      try wp_auto
      wp_apply wp_extension view.graph' graph table strings entry.Extensions' (entries[sint.nat iv]?.getD [])
        (go!"unstable") DFrac.discard DFrac.discard DFrac.discard $$ [Hgraph Htable Hstrings Hextension]
        with ⟨Hgraph, Htable, Hstrings, Hextension⟩
      · iframe Hgraph Htable Hstrings Hextension
      try wp_auto
      rw [ite_eq_left (by constructor <;> word : 0 ≤ sint.Z iv ∧ sint.Z iv < sint.Z view.unstable'.len)]
      wp_apply wp_store_slice_index view.unstable' (sint.Z iv)
        (constructorPrefix (constructorUnstable strings symbols entries) false (sint.nat iv))
        (decide (extensionValue strings (go!"unstable") (entries[sint.nat iv]?.getD []) = go!"true"))
        $$ [Hunstable] with Hunstable
      · iframe; ipureintro
        simp only [constructorPrefix, List.length_append, List.length_take, List.length_replicate,
          constructorUnstable, List.length_mapIdx]
        constructor <;> word
      simp only [hg, decide_false, Bool.false_or] at hu
      rw [constructorPrefix_step _ false _ _ (by simpa [constructorUnstable] using hi) hu]
      wp_for_post
      iframe
      iexists (iv + W64 1), iv, entry
      have hn : sint.nat (iv + W64 1) = sint.nat iv + 1 := by word
      rw [hn]
      iframe
      ipureintro; constructor <;> word
  · have hi : sint.nat iv = symbols.length := by word
    have hk : (constructorKinds strings symbols entries).length = symbols.length := by simp [constructorKinds]
    have hu : (constructorUnstable strings symbols entries).length = symbols.length := by simp [constructorUnstable]
    have hkp : constructorPrefix (constructorKinds strings symbols entries) (go!"") (sint.nat iv) = constructorKinds strings symbols entries := by
      rw [hi, constructorPrefix, hk, List.take_of_length_le (by omega), Nat.sub_self,
        List.replicate_zero, List.append_nil]
    rw [hkp]
    have hup : constructorPrefix (constructorUnstable strings symbols entries) false (sint.nat iv) = constructorUnstable strings symbols entries := by
      rw [hi, constructorPrefix, hu, List.take_of_length_le (by omega), Nat.sub_self,
        List.replicate_zero, List.append_nil]
    rw [hup]
    wp_end

end execution
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
