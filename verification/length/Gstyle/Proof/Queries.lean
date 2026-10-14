module
public import Gstyle.GeneratedProof.github_com.rstyled.gstyle.domain.pdg.query
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.pdg.query

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics] [package_sem : Assumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.pdg.query

instance isPkgInit_inst : IsPkgInit (IProp GF) pkg := define_is_pkg_init iprop(True)
instance get_isPkgInit_wf_inst : GetIsPkgInitWf (IProp GF) pkg := build_get_is_pkg_init_wf

/-- References are one-based; zero is absent, not slot zero. -/
def validReference (ref : w32) (count : w64) : Prop :=
  0 < uint.Z ref ∧ uint.Z ref ≤ sint.Z count
instance (ref : w32) (count : w64) : Decidable (validReference ref count) :=
  inferInstanceAs (Decidable (_ ∧ _))

/-- Every accepted reference identifies precisely its zero-based table slot. -/
theorem wp_ReferenceIndex (ref : w32) (count : w64) (h : validReference ref count) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! ReferenceIndex)) (Val #ref)) (Val #count))
    {{ (index : w64), RET (PairV #index #true);
      ⌜sint.Z index = uint.Z ref - 1 ∧ 0 ≤ sint.Z index ∧ sint.Z index < sint.Z count⌝ }} := by
  wp_start
  wp_auto
  obtain ⟨h0, h1⟩ := h
  repeat (wp_if_destruct <;> try (exfalso; word))
  wp_end
  ipureintro
  word

/-- Invalid references never become an apparently successful lookup. -/
theorem wp_ReferenceReject (ref : w32) (count : w64) (h : ¬ validReference ref count) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! ReferenceIndex)) (Val #ref)) (Val #count))
    {{ RET (PairV #(W64 0) #false); True }} := by
  wp_start
  wp_auto
  wp_if_destruct
  · wp_end
  · wp_if_destruct
    · wp_end
    · wp_if_destruct
      · wp_end
      · exfalso
        apply h
        exact ⟨by word, by word⟩

/-- A successful table query returns the actual referenced record and preserves
ownership of the borrowed table, rather than trusting a supplied Boolean. -/
theorem wp_EvidenceAt (table : GoSlice) (records : List Capability) (ref : w32)
    (record : Capability) (hvalid : validReference ref table.len)
    (hlookup : records[ref.toNat - 1]? = some record) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ table ↦* records }}
      (App (App (Val (@! EvidenceAt)) (Val #table)) (Val #ref))
    {{ RET (PairV #record #true); table ↦* records }} := by
  wp_start as Htable
  wp_auto
  ihave %Hlen := ownSlice_len _ _ _ $$ Htable
  wp_apply wp_ReferenceIndex ref table.len hvalid with %index %Hindex
  rw [ite_eq_left Hindex.2]
  wp_apply wp_load_slice_index (t := Capability.ty) table (sint.Z index) records _ record (by omega) $$ [Htable] with Htable
  · iframe
    ipureintro
    have hi : (sint.Z index).toNat = ref.toNat - 1 := by
      obtain ⟨hp, _⟩ := hvalid
      word
    rw [hi]
    exact hlookup
  wp_end

/-- Failed lookup preserves the table and returns an absent record. -/
theorem wp_EvidenceAbsent (table : GoSlice) (records : List Capability) (ref : w32)
    (h : ¬ validReference ref table.len) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ table ↦* records }}
      (App (App (Val (@! EvidenceAt)) (Val #table)) (Val #ref))
    {{ RET (PairV #(zero_val Capability) #false); table ↦* records }} := by
  wp_start as Htable
  wp_auto
  wp_apply wp_ReferenceReject ref table.len h
  wp_end

/-- Known records retain their exact status; unavailable and unknown differ
at EvidenceAt even though both map to Unknown for status-based policies. -/
theorem wp_EvidenceStatus (table : GoSlice) (records : List Capability) (ref : w32)
    (record : Capability) (hvalid : validReference ref table.len)
    (hlookup : records[ref.toNat - 1]? = some record) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ table ↦* records }}
      (App (App (Val (@! EvidenceStatus)) (Val #table)) (Val #ref))
    {{ RET #(record.Status'); table ↦* records }} := by
  wp_start as Htable
  wp_auto
  wp_apply wp_EvidenceAt table records ref record hvalid hlookup $$ [Htable] with Htable
  · iframe
  wp_end

/-- Missing or malformed references produce Unknown, never Supported. -/
theorem wp_EvidenceUnknown (table : GoSlice) (records : List Capability) (ref : w32)
    (h : ¬ validReference ref table.len) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ table ↦* records }}
      (App (App (Val (@! EvidenceStatus)) (Val #table)) (Val #ref))
    {{ RET #(W8 0); table ↦* records }} := by
  wp_start as Htable
  wp_auto
  wp_apply wp_EvidenceAbsent table records ref h $$ [Htable] with Htable
  · iframe
  wp_end

/-- The capability policy accepts exactly Supported, Known and NotApplicable. -/
theorem wp_CoverageSupported (status : w8) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (Val (@! CoverageSupported)) (Val #status))
    {{ RET #(decide (status = (4#8) ∨ status = (1#8) ∨ status = (3#8))); True }} := by
  wp_start
  wp_auto
  by_cases h4 : status = (4#8) <;> by_cases h1 : status = (1#8) <;> by_cases h3 : status = (3#8)
  all_goals iterate 4 (try simp [Supported, Known, NotApplicable, h4, h1, h3]; try wp_auto)
  all_goals wp_end

/-- Existing dependencies require Supported or Known, never NotApplicable. -/
theorem wp_DependencySupported (status : w8) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (Val (@! DependencySupported)) (Val #status))
    {{ RET #(decide (status = (4#8) ∨ status = (1#8))); True }} := by
  wp_start
  wp_auto
  by_cases h4 : status = (4#8) <;> by_cases h1 : status = (1#8)
  all_goals iterate 3 (try simp [Supported, Known, h4, h1]; try wp_auto)
  all_goals wp_end

/-- The semantic-pattern dependency rule composes the proven lookup and status
policy: a real referenced record must affirmatively support the edge. -/
theorem wp_DependencyAdmitted (table : GoSlice) (records : List Capability) (ref : w32)
    (record : Capability) (hvalid : validReference ref table.len)
    (hlookup : records[ref.toNat - 1]? = some record) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ table ↦* records }}
      (App (App (Val (@! DependencyAdmitted)) (Val #table)) (Val #ref))
    {{ RET #(decide (record.Status' = (4#8) ∨ record.Status' = (1#8))); table ↦* records }} := by
  wp_start as Htable
  wp_auto
  wp_apply wp_EvidenceStatus table records ref record hvalid hlookup $$ [Htable] with Htable
  · iframe
  wp_apply wp_DependencySupported
  wp_end

/-- Missing evidence can never make the pattern's admission rule succeed. -/
theorem wp_DependencyUnavailable (table : GoSlice) (records : List Capability) (ref : w32)
    (h : ¬ validReference ref table.len) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ table ↦* records }}
      (App (App (Val (@! DependencyAdmitted)) (Val #table)) (Val #ref))
    {{ RET #false; table ↦* records }} := by
  wp_start as Htable
  wp_auto
  wp_apply wp_EvidenceUnknown table records ref h $$ [Htable] with Htable
  · iframe
  wp_apply wp_DependencySupported
  wp_end

/-- A known false returns (false,true), while every unavailable Boolean returns
(false,false), regardless of its payload bit. -/
theorem wp_BooleanValue (fact : BooleanFact) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (Val (@! BooleanValue)) (Val #fact))
    {{ RET (if fact.Status' = (1#8) then PairV #fact.Value' #true else PairV #false #false); True }} := by
  wp_start
  wp_auto
  by_cases h : fact.Status' = (1#8)
  all_goals iterate 3 (try simp [Known, h]; try wp_auto)
  all_goals wp_end

/-- The interface gate succeeds exactly for a present, known false fact. -/
theorem wp_KnownFalse (fact : BooleanFact) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (Val (@! KnownFalse)) (Val #fact))
    {{ RET #(decide (fact.Status' = (1#8)) && !fact.Value'); True }} := by
  wp_start
  wp_auto
  wp_apply wp_BooleanValue
  by_cases h : fact.Status' = (1#8)
  all_goals iterate 3 (try simp [h]; try wp_auto)
  all_goals wp_end

end proof
end github_com.rstyled.gstyle.domain.pdg.query
end Perennial
