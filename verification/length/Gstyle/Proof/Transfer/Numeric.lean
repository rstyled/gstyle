module
public import Gstyle.Proof.RuntimeValue
public import Gstyle.Proof.StateMap.Binding
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [FfiSyntax]

def isExact (v : Value) : Bool := v.HasLower' && v.HasUpper' && decide (v.Lower' = v.Upper')
def isEmpty (v : Value) : Bool := v.HasLower' && v.HasUpper' && decide (sint.Z v.Lower' > sint.Z v.Upper')
def lengthConstrained (v : Value) (op : GoString) (b : Value) : Value :=
  if b.Length' && decide (op = go!"<" ∨ op = go!"<=") then
    { v with UpperChecked' := true, BelowLength' := v.BelowLength' || decide (op = go!"<") }
  else v

def numericConstrained (v : Value) (op : GoString) (b : Value) : Value :=
  let v := lengthConstrained v op b
  if isExact b then constrainedValue v op b.Lower' else v

def impossibleComparison (kind op : GoString) (b : Value) : Bool :=
  decide (kind = go!"\"signed\"") && isExact b &&
    ((decide (op = go!"<") && decide (b.Lower' = 9223372036854775808#64)) ||
     (decide (op = go!">") && decide (b.Lower' = W64 9223372036854775807)))

def constraintValue (kind : GoString) (changing : Bool) (key : w32)
    (m : StateData) (op : GoString) (b : Value) : Value :=
  numericConstrained (bindingValue kind changing key m) op b

def constraintRejected (kind : GoString) (changing : Bool) (key : w32)
    (m : StateData) (op : GoString) (b : Value) : Bool :=
  isEmpty (constraintValue kind changing key m op b) || impossibleComparison kind op b

def constraintState (active : Bool) (kind : GoString) (changing : Bool) (key : w32)
    (m : StateData) (op : GoString) (b : Value) : StateData :=
  if active && !constraintRejected kind changing key m op b then
    <[key := constraintValue kind changing key m op b]>m
  else m

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [graphView.TypeAssumptions] [Value.TypeAssumptions] [State.TypeAssumptions] [pdg.query.Assumptions]
variable [FuncUnfold integerConstraint [] integerConstraint.impl]
variable [FuncUnfold lengthConstraint [] lengthConstraint.impl]
variable [FuncUnfold exactInteger [] exactInteger.impl]
variable [FuncUnfold emptyRange [] emptyRange.impl]
variable [interval.Assumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

theorem wp_exactInteger (v : Value) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (Val (@! exactInteger)) (Val #v))
    {{ RET #(isExact v); True }} := by
  wp_start
  cases hl : v.HasLower' <;> cases hu : v.HasUpper' <;> by_cases he : v.Lower' = v.Upper'
  all_goals iterate 6 (try simp [isExact, hl, hu, he]; try wp_auto)
  all_goals wp_end

theorem wp_emptyRange (v : Value) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (Val (@! emptyRange)) (Val #v))
    {{ RET #(isEmpty v); True }} := by
  wp_start
  cases hl : v.HasLower' <;> cases hu : v.HasUpper' <;> by_cases he : sint.Z v.Lower' > sint.Z v.Upper'
  all_goals iterate 6 (try simp [isEmpty, hl, hu, he]; try wp_auto)
  all_goals wp_end

theorem wp_lengthConstraint (v : Value) (op : GoString) (b : Value) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (App (Val (@! lengthConstraint)) (Val #v)) (Val #op)) (Val #b))
    {{ RET #(lengthConstrained v op b); True }} := by
  wp_start
  cases hb : b.Length' <;> cases hv : v.BelowLength' <;>
    by_cases hl : op = go!"<" <;> by_cases he : op = go!"<="
  all_goals iterate 8 (try simp [lengthConstrained, hb, hv, hl, he]; try wp_auto)
  all_goals wp_end

theorem wp_trackedBinding (receiver : Loc) (view : graphView) (tracked : List Bool)
    (key : w32) (active : Bool)
    (hk : 0 < uint.Z key ∧ uint.Z key ≤ tracked.length)
    (htracked : tracked[(uint.Z key - 1).toNat]? = some active) (dq : DFrac := DFrac.own 1) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{dq} view ∗ view.tracked' ↦*{dq} tracked) }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"trackedBinding" #receiver)) (Val #key))
    {{ RET #active; receiver ↦{dq} view ∗ view.tracked' ↦*{dq} tracked }} := by
  wp_start as ⟨Hview, Htracked⟩
  ihave %Hlen := ownSlice_len $$ Htracked
  wp_auto
  obtain ⟨hk0, hk1⟩ := hk
  repeat (wp_if_destruct <;> try (exfalso; word))
  have Hi : sint.Z (W64 (uint.Z (key - W32 1))) = uint.Z key - 1 := by word
  rw [Hi]
  rw [ite_eq_left (by constructor <;> word : 0 ≤ uint.Z key - 1 ∧ uint.Z key - 1 < sint.Z view.tracked'.len)]
  wp_apply wp_load_slice_index view.tracked' (uint.Z key - 1) tracked dq active (by word) $$ [Htracked] with Htracked
  · iframe; ipureintro; exact htracked
  wp_end

theorem wp_trackedBinding_invalid (receiver : Loc) (view : graphView) (tracked : List Bool)
    (key : w32) (dq : DFrac) (hkey : key = W32 0 ∨ tracked.length < uint.Z key) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{dq} view ∗ view.tracked' ↦*{dq} tracked) }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"trackedBinding" #receiver)) (Val #key))
    {{ RET #false; receiver ↦{dq} view ∗ view.tracked' ↦*{dq} tracked }} := by
  wp_start as ⟨Hview, Htracked⟩
  ihave %Hlen := ownSlice_len $$ Htracked
  try wp_auto
  rcases hkey with hz | hout
  · subst key
    try wp_auto
    wp_end
  · repeat (wp_if_destruct <;> try (exfalso; word))
    all_goals wp_end

theorem wp_impossibleSignedComparison (receiver : Loc) (view : graphView) (kinds : List GoString)
    (key : w32) (kind op : GoString) (b : Value)
    (hk : 0 < uint.Z key ∧ uint.Z key ≤ kinds.length)
    (hkind : kinds[(uint.Z key - 1).toNat]? = some kind) (dq : DFrac := DFrac.own 1) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds) }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"impossibleSignedComparison" #receiver)) (Val #key)) (Val #op)) (Val #b))
    {{ RET #(impossibleComparison kind op b); receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds }} := by
  wp_start as ⟨Hview, Hkinds⟩
  ihave %Hlen := ownSlice_len $$ Hkinds
  wp_auto
  have Hi : sint.Z (W64 (uint.Z (key - W32 1))) = uint.Z key - 1 := by obtain ⟨_, _⟩ := hk; word
  rw [Hi]
  rw [ite_eq_left (by obtain ⟨_, _⟩ := hk; constructor <;> word : 0 ≤ uint.Z key - 1 ∧ uint.Z key - 1 < sint.Z view.kinds'.len)]
  wp_apply wp_load_slice_index view.kinds' (uint.Z key - 1) kinds dq kind (by obtain ⟨_, _⟩ := hk; word) $$ [Hkinds] with Hkinds
  · iframe; ipureintro; exact hkind
  by_cases hs : kind = go!"\"signed\""
  · simp [hs]
    wp_auto
    wp_apply wp_exactInteger b
    cases he : isExact b <;> by_cases hl : op = go!"<" <;> by_cases hg : op = go!">"
    all_goals iterate 8 (try simp [impossibleComparison, hs, he, hl, hg]; try wp_auto)
    all_goals wp_end
  · iterate 5 (try simp [impossibleComparison, hs]; try wp_auto)
    wp_end

theorem wp_infeasibleConstraint (receiver : Loc) (view : graphView) (kinds : List GoString)
    (key : w32) (kind op : GoString) (b v : Value)
    (hk : 0 < uint.Z key ∧ uint.Z key ≤ kinds.length)
    (hkind : kinds[(uint.Z key - 1).toNat]? = some kind) (dq : DFrac := DFrac.own 1) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"infeasibleConstraint" #receiver)) (Val #key)) (Val #op)) (Val #b)) (Val #v))
    {{ RET #(isEmpty v || impossibleComparison kind op b); receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds }} := by
  wp_start as ⟨Hview, Hkinds⟩
  wp_auto
  wp_apply wp_emptyRange v
  cases he : isEmpty v
  · simp [he]
    wp_auto
    wp_apply wp_impossibleSignedComparison receiver view kinds key kind op b hk hkind dq $$ [Hview Hkinds] with ⟨Hview, Hkinds⟩
    · iframe
    wp_end
  · simp [he]
    wp_auto
    wp_end

theorem wp_constrain_numeric (receiver : Loc) (view : graphView)
    (kinds : List GoString) (unstable tracked : List Bool)
    (state : Loc) (m : StateData) (key : w32) (kind op : GoString) (changing active : Bool) (b : Value)
    (hk : 0 < uint.Z key ∧ uint.Z key ≤ kinds.length)
    (hkind : kinds[(uint.Z key - 1).toNat]? = some kind)
    (hu : unstable.length = kinds.length)
    (ht : tracked.length = kinds.length)
    (hunstable : unstable[(uint.Z key - 1).toNat]? = some changing)
    (htracked : tracked[(uint.Z key - 1).toNat]? = some active)
    (hnil : b.Nil' = false) (dq : DFrac := DFrac.own 1) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
      (receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds ∗ view.unstable' ↦*{dq} unstable ∗ view.tracked' ↦*{dq} tracked ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"constrain" #receiver)) (Val #key)) (Val #op)) (Val #b)) (Val #state))
    {{ RET #(if active && constraintRejected kind changing key m op b then map.nil else state);
      receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds ∗ view.unstable' ↦*{dq} unstable ∗ view.tracked' ↦*{dq} tracked ∗
      state ↦$ (constraintState active kind changing key m op b) }} := by
  wp_start as ⟨Hview, Hkinds, Hunstable, Htracked, Hstate⟩
  wp_auto
  wp_apply wp_trackedBinding receiver view tracked key active (by simpa [ht] using hk) htracked dq $$ [Hview Htracked] with ⟨Hview, Htracked⟩
  · iframe
  cases ha : active
  · simp [ha, constraintState]
    wp_auto
    wp_end
  · simp [ha]
    wp_auto
    wp_apply wp_binding receiver view kinds unstable state m key kind changing hk hkind hu hunstable dq $$ [Hview Hkinds Hunstable Hstate] with ⟨Hview, Hkinds, Hunstable, Hstate⟩
    · iframe
    simp [hnil]
    wp_auto
    wp_apply wp_lengthConstraint (bindingValue kind changing key m) op b
    wp_apply wp_exactInteger b
    cases he : isExact b
    · wp_auto
      wp_apply wp_infeasibleConstraint receiver view kinds key kind op b (lengthConstrained (bindingValue kind changing key m) op b) hk hkind dq $$ [Hview Hkinds] with ⟨Hview, Hkinds⟩
      · iframe
      have hv : constraintValue kind changing key m op b = lengthConstrained (bindingValue kind changing key m) op b := by simp [constraintValue, numericConstrained, he]
      rw [← hv]
      cases hr : constraintRejected kind changing key m op b
      all_goals unfold constraintRejected at hr
      all_goals simp [hr, constraintState, constraintRejected]
      all_goals try wp_auto
      · wp_apply wp_mapInsert pdg.Ref.ty state m key (constraintValue kind changing key m op b) $$ [Hstate] with Hstate
        · iframe
        try wp_auto
        wp_end
      · wp_end
    · wp_auto
      wp_apply wp_integerConstraint (lengthConstrained (bindingValue kind changing key m) op b) op b.Lower' $$ []
      · iframe; iassumption
      wp_apply wp_infeasibleConstraint receiver view kinds key kind op b (constrainedValue (lengthConstrained (bindingValue kind changing key m) op b) op b.Lower') hk hkind dq $$ [Hview Hkinds] with ⟨Hview, Hkinds⟩
      · iframe
      have hv : constraintValue kind changing key m op b = constrainedValue (lengthConstrained (bindingValue kind changing key m) op b) op b.Lower' := by simp [constraintValue, numericConstrained, he]
      rw [← hv]
      cases hr : constraintRejected kind changing key m op b
      all_goals unfold constraintRejected at hr
      all_goals simp [hr, constraintState, constraintRejected]
      all_goals try wp_auto
      · wp_apply wp_mapInsert pdg.Ref.ty state m key (constraintValue kind changing key m op b) $$ [Hstate] with Hstate
        · iframe
        try wp_auto
        wp_end
      · wp_end

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
