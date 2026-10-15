module
public import Gstyle.Proof.StateMap.Model
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [go.Semantics] [Value.TypeAssumptions] [State.TypeAssumptions] [pdg.query.Assumptions]
variable [FuncUnfold copyState [] copyState.impl]

theorem map_body_done (P : IProp GF) (Φ : val → IProp GF) :
    P ⊢ forMapPostcondition P Φ executeVal := by
  iintro HP
  unfold forMapPostcondition
  iright; ileft
  isplitr
  · ipureintro; rfl
  · iexact HP


theorem wp_copyState (state : Loc) (m : StateData) :
    {{ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard ∗ state ↦$ m }}
      (App (Val (@! copyState)) (Val #state))
    {{ (out : Loc), RET #out; state ↦$ m ∗ out ↦$ m }} := by
  wp_start as Hstate
  wp_auto
  wp_apply wp_map_len state m (DFrac.own 1) $$ [Hstate] with Hstate
  · iframe
  rw [go.make2_underlying, go.is_underlying (t := State.ty)]
  wp_apply wp_map_make2 (K := w32) (V := Value) (W64 (GMap.size m)) pdg.Ref.ty Value.ty with %out Hout
  wp_func_lits
  wp_bind (App (App (Val (map.forRange _ _)) _) _)
  iapply (wp_map_for_range (fun keys i => iprop(∃ (sv : w32) (vv : Value) (om : StateData),
    symbol_ptr ↦ sv ∗ value_ptr ↦ vv ∗ out_ptr ↦ out ∗ out ↦$ om ∗
    ⌜0 ≤ i ∧ prefixMap keys i (fun k => m !! k) om⌝))) $$ Hstate
  iintro %keys %Hkeys
  isplitl [symbol value out Hout]
  · iexists _, _, ∅
    iframe
    ipureintro
    exact ⟨by omega, prefixMap_start keys _⟩
  isplitl []
  · imodintro
    iintro %i %key %v ⟨%Hkey, %Hlookup⟩ ⟨%sv, %vv, %om, Hsymbol, Hvalue, Houtptr, Hout, %Hi, %Hprefix⟩
    wp_auto
    wp_apply wp_mapInsert pdg.Ref.ty out om key v $$ [Hout] with Hout
    · iframe
    iapply map_body_done
    iexists key, v, (<[key := v]> om)
    iframe
    ipureintro
    exact ⟨by omega, prefixMap_step keys i _ om key v Hi Hkey Hlookup Hprefix⟩
  iintro ⟨%sv, %vv, %om, Hsymbol, Hvalue, Houtptr, Hout, %Hi, %Hprefix⟩ Hstate
  have Hsame : om = m := by
    apply GMap.ext
    intro k
    have hp := Hprefix k
    have hlen := Hkeys.2.1
    have htake : keys.take (Int.toNat (GMap.size m : Int)) = keys := by
      rw [← hlen]; simp
    rw [htake] at hp
    by_cases hk : k ∈ keys
    · simpa [hk] using hp
    · have hm : m !! k = none := by
        cases h : m !! k with
        | none => rfl
        | some v => exact False.elim (hk ((Hkeys.1 k).mpr (by simp [h])))
      simpa [hk, hm] using hp
  subst om
  wp_auto
  wp_end

theorem wp_copyState_nil :
    {{ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard }}
      (App (Val (@! copyState)) (Val #map.nil))
    {{ (out : Loc), RET #out; out ↦$ (∅ : StateData) }} := by
  wp_start
  wp_auto
  rw [go.make2_underlying, go.is_underlying (t := State.ty)]
  wp_apply wp_map_make2 (K := w32) (V := Value) (W64 0) pdg.Ref.ty Value.ty with %out Hout
  wp_func_lits
  wp_auto
  wp_end

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
