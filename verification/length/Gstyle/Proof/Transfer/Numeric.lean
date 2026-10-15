module
public import Gstyle.Proof.Transfer.Tracking
public import Gstyle.Proof.RuntimeValue
public import Gstyle.Proof.Transfer.NumericConstrainQueries
public import Gstyle.Proof.StateMap.Binding
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [FfiSyntax]

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
  exact wp_constrain_numeric_query receiver view kinds unstable tracked state m key kind op changing active b
    hk hkind hu ht hunstable htracked hnil dq
    (wp_integerConstraint (lengthConstrained (bindingValue kind changing key m) op b) op b.Lower')

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
