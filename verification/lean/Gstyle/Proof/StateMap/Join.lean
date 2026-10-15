module
public import Gstyle.Proof.RuntimeValue.Merge
public import Gstyle.Proof.StateMap.Binding
public import Gstyle.Proof.StateMap.Copy
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [go.Semantics] [graphView.TypeAssumptions] [Value.TypeAssumptions]
variable [State.TypeAssumptions] [pdg.query.Assumptions] [interval.Assumptions]
variable [FuncUnfold integerConstraint [] integerConstraint.impl]

theorem wp_join (receiver : Loc) (view : graphView) (kinds : List GoString) (unstable : List Bool)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (previous next : Loc) (p n : StateData)
    (hlen : unstable.length = kinds.length)
    (hmeta : ∀ key, (p !! key).isSome ∨ (n !! key).isSome →
      (0 < uint.Z key ∧ uint.Z key ≤ kinds.length) ∧
      kinds[(uint.Z key - 1).toNat]? = some (kind key) ∧
      unstable[(uint.Z key - 1).toNat]? = some (changing key)) (dq : DFrac := DFrac.own 1) :
    {{ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard ∗
      isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
      (receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds ∗ view.unstable' ↦*{dq} unstable ∗ previous ↦$ p ∗ next ↦$ n) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"join" #receiver)) (Val #previous)) (Val #next))
    {{ (out : Loc), RET #out; ∃ om : StateData,
      receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds ∗ view.unstable' ↦*{dq} unstable ∗ previous ↦$ p ∗ next ↦$ n ∗
      out ↦$ om ∗ ⌜∀ key, om !! key = joinLookup kind changing p n key⌝ }} := by
  wp_start as ⟨Hview, Hkinds, Hunstable, Hprevious, Hnext⟩
  ihave %Hpnon := ownMap_not_nil previous p (DFrac.own 1) $$ Hprevious
  ihave %Hnnon := ownMap_not_nil next n (DFrac.own 1) $$ Hnext
  wp_auto
  repeat (wp_if_destruct <;> try contradiction)
  rw [go.make1_underlying, go.is_underlying (t := State.ty)]
  wp_apply wp_map_make1 (K := w32) (V := Value) pdg.Ref.ty Value.ty with %out Hout
  wp_func_lits
  wp_bind (App (App (Val (map.forRange _ _)) _) _)
  iapply (wp_map_for_range (fun keys i => iprop(∃ (sv : w32) (vv : Value) (om : StateData),
    symbol_ptr ↦ sv ∗ value_ptr ↦ vv ∗ v_ptr ↦ receiver ∗ previous_ptr ↦ previous ∗ next_ptr ↦ next ∗ out_ptr ↦ out ∗
    receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds ∗ view.unstable' ↦*{dq} unstable ∗ next ↦$ n ∗ out ↦$ om ∗
    ⌜0 ≤ i ∧ prefixMap keys i (leftLookup kind changing p n) om⌝))) $$ Hprevious
  iintro %keys %Hkeys
  isplitl [symbol value v previous next out Hview Hkinds Hunstable Hnext Hout]
  · iexists _, _, ∅
    iframe
    ipureintro
    exact ⟨by omega, prefixMap_start keys _⟩
  isplitl []
  · imodintro
    iintro %i %key %value ⟨%Hkey, %Hlookup⟩
      ⟨%sv, %vv, %om, Hsymbol, Hvalue, Hvptr, Hpptr, Hnptr, Hoptr, Hview, Hkinds, Hunstable, Hnext, Hout, %Hi, %Hprefix⟩
    obtain ⟨Hvalid, Hkind, Hunst⟩ := hmeta key (Or.inl (by simp [Hlookup]))
    wp_auto
    wp_apply wp_binding receiver view kinds unstable next n key (kind key) (changing key) Hvalid Hkind hlen Hunst dq
      $$ [Hview Hkinds Hunstable Hnext] with ⟨Hview, Hkinds, Hunstable, Hnext⟩
    · iframe
    wp_apply wp_merge value (bindingValue (kind key) (changing key) key n) $$ []
    · iframe; iassumption
    wp_apply wp_mapInsert pdg.Ref.ty out om key (merged value (bindingValue (kind key) (changing key) key n)) $$ [Hout] with Hout
    · iframe
    iapply map_body_done
    iexists key, value, _
    iframe
    ipureintro
    refine ⟨by omega, prefixMap_step keys i _ om key _ Hi Hkey ?_ Hprefix⟩
    simp [leftLookup, Hlookup]
  iintro ⟨%sv, %vv, %om, Hsymbol, Hvalue, Hvptr, Hpptr, Hnptr, Hoptr, Hview, Hkinds, Hunstable, Hnext, Hout, %Hi, %Hprefix⟩ Hprevious
  have Hleft : ∀ key, om !! key = leftLookup kind changing p n key := by
    intro key
    have hp := Hprefix key
    have htake : keys.take (Int.toNat (GMap.size p : Int)) = keys := by rw [← Hkeys.2.1]; simp
    rw [htake] at hp
    by_cases hk : key ∈ keys
    · simpa [hk] using hp
    · have hm : p !! key = none := by
        cases h : p !! key with
        | none => rfl
        | some v => exact False.elim (hk ((Hkeys.1 key).mpr (by simp [h])))
      simpa [hk, leftLookup, hm] using hp
  iclear Hsymbol Hvalue
  wp_auto
  wp_func_lits
  wp_bind (App (App (Val (map.forRange _ _)) _) _)
  iapply (wp_map_for_range (fun keys i => iprop(∃ (sv : w32) (vv : Value) (om : StateData),
    symbol_ptr ↦ sv ∗ value_ptr ↦ vv ∗ v_ptr ↦ receiver ∗ previous_ptr ↦ previous ∗ out_ptr ↦ out ∗
    receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds ∗ view.unstable' ↦*{dq} unstable ∗ previous ↦$ p ∗ out ↦$ om ∗
    ⌜0 ≤ i ∧ overlayPrefix keys i (joinLookup kind changing p n) (leftLookup kind changing p n) om⌝))) $$ Hnext
  iintro %nkeys %Hnkeys
  isplitl [symbol value Hvptr Hpptr Hoptr Hview Hkinds Hunstable Hprevious Hout]
  · iexists _, _, om
    iframe
    ipureintro
    exact ⟨by omega, by simpa [overlayPrefix] using Hleft⟩
  isplitl []
  · imodintro
    iintro %i %key %value ⟨%Hkey, %Hlookup⟩
      ⟨%sv, %vv, %om, Hsymbol, Hvalue, Hvptr, Hpptr, Hoptr, Hview, Hkinds, Hunstable, Hprevious, Hout, %Hi, %Hprefix⟩
    wp_auto
    wp_apply wp_map_lookup2 pdg.Ref.ty Value.ty previous p key (DFrac.own 1) $$ [Hprevious] with Hprevious
    · iframe
    cases Hp : p !! key with
    | some old =>
      simp [Hp]
      wp_auto
      iapply map_body_done
      iexists key, value, om
      iframe
      ipureintro
      refine ⟨by omega, overlayPrefix_skip nkeys i _ _ om key Hi Hkey ?_ Hprefix⟩
      simp [joinLookup, leftLookup, Hp]
    | none =>
      simp [Hp]
      wp_auto
      obtain ⟨Hvalid, Hkind, Hunst⟩ := hmeta key (Or.inr (by simp [Hlookup]))
      wp_apply wp_binding receiver view kinds unstable previous p key (kind key) (changing key) Hvalid Hkind hlen Hunst dq
        $$ [Hview Hkinds Hunstable Hprevious] with ⟨Hview, Hkinds, Hunstable, Hprevious⟩
      · iframe
      wp_apply wp_merge value (bindingValue (kind key) (changing key) key p) $$ []
      · iframe; iassumption
      wp_apply wp_mapInsert pdg.Ref.ty out om key (merged value (bindingValue (kind key) (changing key) key p)) $$ [Hout] with Hout
      · iframe
      iapply map_body_done
      iexists key, value, _
      iframe
      ipureintro
      refine ⟨by omega, overlayPrefix_step nkeys i _ _ om key _ Hi Hkey ?_ Hprefix⟩
      simp [joinLookup, Hp, Hlookup]
  iintro ⟨%sv, %vv, %result, Hsymbol, Hvalue, Hvptr, Hpptr, Hoptr, Hview, Hkinds, Hunstable, Hprevious, Hout, %Hi, %Hprefix⟩ Hnext
  have Hresult : ∀ key, result !! key = joinLookup kind changing p n key := by
    intro key
    have hp := Hprefix key
    have htake : nkeys.take (Int.toNat (GMap.size n : Int)) = nkeys := by rw [← Hnkeys.2.1]; simp
    rw [htake] at hp
    by_cases hk : key ∈ nkeys
    · simpa [hk] using hp
    · have hn : n !! key = none := by
        cases h : n !! key with
        | none => rfl
        | some v => exact False.elim (hk ((Hnkeys.1 key).mpr (by simp [h])))
      cases hprev : p !! key <;> simpa [hk, joinLookup, leftLookup, hn, hprev] using hp
  wp_auto
  wp_end

theorem wp_join_sound (receiver : Loc) (view : graphView) (kinds : List GoString) (unstable : List Bool)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (previous next : Loc) (p n : StateData)
    (hlen : unstable.length = kinds.length)
    (hmeta : ∀ key, (p !! key).isSome ∨ (n !! key).isSome →
      (0 < uint.Z key ∧ uint.Z key ≤ kinds.length) ∧
      kinds[(uint.Z key - 1).toNat]? = some (kind key) ∧
      unstable[(uint.Z key - 1).toNat]? = some (changing key)) (dq : DFrac := DFrac.own 1) :
    {{ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard ∗
      isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
      (receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds ∗ view.unstable' ↦*{dq} unstable ∗ previous ↦$ p ∗ next ↦$ n) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"join" #receiver)) (Val #previous)) (Val #next))
    {{ (out : Loc), RET #out; ∃ om : StateData,
      receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds ∗ view.unstable' ↦*{dq} unstable ∗ previous ↦$ p ∗ next ↦$ n ∗ out ↦$ om ∗
      ⌜(∀ key, (om !! key).isSome = ((p !! key).isSome || (n !! key).isSome)) ∧
        ∀ key x, (interval.contains (bounds (bindingValue (kind key) (changing key) key p)) x ∨
          interval.contains (bounds (bindingValue (kind key) (changing key) key n)) x) →
          interval.contains (bounds (bindingValue (kind key) (changing key) key om)) x⌝ }} := by
  wp_start_folded as Hinputs
  wp_apply wp_join receiver view kinds unstable kind changing previous next p n hlen hmeta dq $$ [Hinputs]
    with %out ⟨%om, Hview, Hkinds, Hunstable, Hprevious, Hnext, Hout, %Hlookup⟩
  · iframe
  wp_end
  iexists om
  iframe
  ipureintro
  constructor
  · intro key
    rw [Hlookup key]
    exact joinLookup_domain kind changing p n key
  · intro key x hx
    exact state_join_sound kind changing p n om key x Hlookup hx

variable [FuncUnfold copyState [] copyState.impl]

theorem wp_join_left_nil (receiver next : Loc) (n : StateData) :
    {{ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard ∗ next ↦$ n }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"join" #receiver)) (Val #map.nil)) (Val #next))
    {{ (out : Loc), RET #out; next ↦$ n ∗ out ↦$ n }} := by
  wp_start as Hnext
  wp_auto
  wp_apply wp_copyState next n $$ [Hnext] with %out ⟨Hnext, Hout⟩
  · iframe
  wp_end

theorem wp_join_right_nil (receiver previous : Loc) (p : StateData) :
    {{ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard ∗ previous ↦$ p }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"join" #receiver)) (Val #previous)) (Val #map.nil))
    {{ (out : Loc), RET #out; previous ↦$ p ∗ out ↦$ p }} := by
  wp_start as Hprevious
  ihave %Hnon := ownMap_not_nil previous p (DFrac.own 1) $$ Hprevious
  wp_auto
  wp_if_destruct
  · contradiction
  · wp_apply wp_copyState previous p $$ [Hprevious] with %out ⟨Hprevious, Hout⟩
    · iframe
    wp_end

theorem wp_join_both_nil (receiver : Loc) :
    {{ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"join" #receiver)) (Val #map.nil)) (Val #map.nil))
    {{ (out : Loc), RET #out; out ↦$ (∅ : StateData) }} := by
  wp_start
  wp_auto
  wp_apply wp_copyState_nil with %out Hout
  wp_end

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
