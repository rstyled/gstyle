module
public import Gstyle.Proof.StateMap.Layout
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

variable [FfiSyntax]

abbrev StateData := GMap w32 Value

def baseKind (kind : GoString) : Value :=
  if kind = go!"\"unsigned\"" then
    { (zero_val Value) with Lower' := W64 0, HasLower' := true } else zero_val Value

def defaultKind (kind : GoString) (key : w32) : Value :=
  if kind = go!"\"pointer\"" then
    { (baseKind kind) with Identity' := W64 (uint.Z key) * W64 2 } else baseKind kind

def bindingValue (kind : GoString) (unstable : Bool) (key : w32) (m : StateData) : Value :=
  if unstable then baseKind kind else (m !! key).getD (defaultKind kind key)

def prefixMap (keys : List w32) (i : Int) (f : w32 → Option Value) (m : StateData) : Prop :=
  ∀ k, m !! k = if k ∈ keys.take i.toNat then f k else none

theorem prefixMap_start (keys : List w32) (f : w32 → Option Value) :
    prefixMap keys 0 f ∅ := by simp [prefixMap]

theorem prefixMap_step (keys : List w32) (i : Int) (f : w32 → Option Value)
    (m : StateData) (key : w32) (value : Value) (hi : 0 ≤ i)
    (hk : keys[i.toNat]? = some key) (hv : f key = some value)
    (hm : prefixMap keys i f m) :
    prefixMap keys (i + 1) f (<[key := value]> m) := by
  intro k
  have hn : (i + 1).toNat = i.toNat + 1 := by omega
  rw [hn, List.take_add_one, hk]
  by_cases he : key = k
  · subst k
    simp [GMap.lookup_insert, hv]
  · simp [GMap.lookup_insert_eq_iff, he, Ne.symm he, hm k]


def leftLookup (kind : w32 → GoString) (changing : w32 → Bool)
    (p n : StateData) (key : w32) : Option Value :=
  (p !! key).map (fun v => merged v (bindingValue (kind key) (changing key) key n))

def joinLookup (kind : w32 → GoString) (changing : w32 → Bool)
    (p n : StateData) (key : w32) : Option Value :=
  match p !! key with
  | some v => some (merged v (bindingValue (kind key) (changing key) key n))
  | none => (n !! key).map (fun v => merged v (bindingValue (kind key) (changing key) key p))

def overlayPrefix (keys : List w32) (i : Int) (f base : w32 → Option Value) (m : StateData) : Prop :=
  ∀ k, m !! k = if k ∈ keys.take i.toNat then f k else base k

theorem overlayPrefix_step (keys : List w32) (i : Int) (f base : w32 → Option Value)
    (m : StateData) (key : w32) (value : Value) (hi : 0 ≤ i)
    (hk : keys[i.toNat]? = some key) (hv : f key = some value)
    (hm : overlayPrefix keys i f base m) :
    overlayPrefix keys (i + 1) f base (<[key := value]> m) := by
  intro k
  have hn : (i + 1).toNat = i.toNat + 1 := by omega
  rw [hn, List.take_add_one, hk]
  by_cases he : key = k
  · subst k; simp [GMap.lookup_insert, hv]
  · simp [GMap.lookup_insert_eq_iff, he, Ne.symm he, hm k]

theorem overlayPrefix_skip (keys : List w32) (i : Int) (f base : w32 → Option Value)
    (m : StateData) (key : w32) (hi : 0 ≤ i)
    (hk : keys[i.toNat]? = some key) (hv : f key = base key)
    (hm : overlayPrefix keys i f base m) :
    overlayPrefix keys (i + 1) f base m := by
  intro k
  have hn : (i + 1).toNat = i.toNat + 1 := by omega
  rw [hn, List.take_add_one, hk]
  by_cases he : key = k
  · subst k; simp [hm key, hv]
  · simp [he, Ne.symm he, hm k]


theorem joinLookup_domain (kind : w32 → GoString) (changing : w32 → Bool) (p n : StateData) (key : w32) :
    (joinLookup kind changing p n key).isSome = ((p !! key).isSome || (n !! key).isSome) := by
  cases hp : p !! key <;> cases hn : n !! key <;> simp [joinLookup, hp, hn]

theorem state_join_sound (kind : w32 → GoString) (changing : w32 → Bool)
    (p n out : StateData) (key : w32) (x : Int)
    (hout : ∀ k, out !! k = joinLookup kind changing p n k)
    (hx : interval.contains (bounds (bindingValue (kind key) (changing key) key p)) x ∨
      interval.contains (bounds (bindingValue (kind key) (changing key) key n)) x) :
    interval.contains (bounds (bindingValue (kind key) (changing key) key out)) x := by
  cases hc : changing key
  · simp only [bindingValue, hc, Bool.false_eq_true, ↓reduceIte] at hx ⊢
    have ho := hout key
    cases hp : p !! key with
    | some pv =>
      simp only [joinLookup, hp] at ho
      rw [ho]
      simp only [Option.getD_some]
      have hjoin := interval.join_sound (bounds pv)
        (bounds (bindingValue (kind key) false key n)) x
      have hin : interval.contains (bounds pv) x ∨ interval.contains (bounds (bindingValue (kind key) false key n)) x := by
        simpa [bindingValue, hp, hc] using hx
      simpa [bounds, merged, mergedPointers, mergedBounds, hc] using hjoin hin
    | none =>
      cases hn : n !! key with
      | some nv =>
        simp only [joinLookup, hp, hn, Option.map_some] at ho
        rw [ho]
        simp only [Option.getD_some]
        have hjoin := interval.join_sound (bounds nv)
          (bounds (bindingValue (kind key) false key p)) x
        have hin : interval.contains (bounds nv) x ∨ interval.contains (bounds (bindingValue (kind key) false key p)) x := by
          simpa [bindingValue, hp, hn, hc] using hx.symm
        simpa [bounds, merged, mergedPointers, mergedBounds, hc] using hjoin hin
      | none =>
        simp only [joinLookup, hp, hn, Option.map_none] at ho
        simpa [ho, hp, hn] using hx
  · simpa [bindingValue, hc] using hx

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
