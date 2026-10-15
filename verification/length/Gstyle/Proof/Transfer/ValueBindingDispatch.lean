module
public import Gstyle.Proof.RuntimePackage
public import Gstyle.Proof.Pdg.GraphLayout
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [graphView.TypeAssumptions] [pdg.query.Assumptions]
variable [pdg.Graph.TypeAssumptions] [pdg.Node.TypeAssumptions] [pdg.Attributes.TypeAssumptions]
variable [pdg.AttributeSet.TypeAssumptions] [pdg.AttributeTables.TypeAssumptions]
variable [pdg.Producer.TypeAssumptions] [pdg.Function.TypeAssumptions] [pdg.Span.TypeAssumptions]
variable [pdg.CountFact.TypeAssumptions] [pdg.Conformance.TypeAssumptions] [pdg.Fact.TypeAssumptions]
variable [pdg.Position.TypeAssumptions] [pdg.CallForm.TypeAssumptions] [pdg.ExitKind.TypeAssumptions]
variable [pdg.ConformanceStatus.TypeAssumptions] [pdg.Category.TypeAssumptions]
variable [FuncUnfold pdg.listAt [pdg.Ref.ty] (pdg.listAt.impl pdg.Ref.ty)]
variable [FuncUnfold pdg.listAt [pdg.Access.ty] (pdg.listAt.impl pdg.Access.ty)]
variable [FuncUnfold pdg.listAt [pdg.Extension.ty] (pdg.listAt.impl pdg.Extension.ty)]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

variable [pdg.Access.TypeAssumptions] [pdg.Tables.TypeAssumptions] [pdg.Extension.TypeAssumptions]
variable [FuncUnfold directBindingRead [] directBindingRead.impl]
variable [FuncUnfold operatorText [] operatorText.impl]


variable [FuncUnfold extension [] extension.impl]

variable [Value.TypeAssumptions] [State.TypeAssumptions]

theorem wp_value_binding_dispatch (receiver : Loc) (ref symbol : w32) (state : Loc)
    (constant result : Value) (resources : IProp GF)
    (href : ref ≠ W32 0) (hsymbol : symbol ≠ W32 0)
    (hattrs :
      {{ isPkgInit (PROP := IProp GF) pkg ∗ resources }}
        (App (Val #(methods (go.PointerType graphView.ty) go!"attrs" #receiver)) (Val #ref))
      {{ (attrs : pdg.Attributes), RET #attrs; resources }})
    (hconstant :
      {{ isPkgInit (PROP := IProp GF) pkg ∗ resources }}
        (App (Val #(methods (go.PointerType graphView.ty) go!"constantValue" #receiver)) (Val #ref))
      {{ RET #constant; resources }})
    (hresolve :
      {{ isPkgInit (PROP := IProp GF) pkg ∗ resources }}
        (App (Val #(methods (go.PointerType graphView.ty) go!"symbol" #receiver)) (Val #ref))
      {{ RET #symbol; resources }})
    (hbinding :
      {{ isPkgInit (PROP := IProp GF) pkg ∗ resources }}
        (App (App (Val #(methods (go.PointerType graphView.ty) go!"binding" #receiver)) (Val #symbol)) (Val #state))
      {{ RET #result; resources }}) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ resources }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"value" #receiver)) (Val #ref)) (Val #state))
    {{ RET #result; resources }} := by
  wp_start as Hresources
  try wp_auto
  wp_if_destruct
  · exfalso; simp_all
  · wp_apply hattrs $$ [Hresources] with %attrs Hresources
    · iexact Hresources
    try wp_auto
    wp_apply hconstant $$ [Hresources] with Hresources
    · iexact Hresources
    try wp_auto
    wp_apply hresolve $$ [Hresources] with Hresources
    · iexact Hresources
    try wp_auto
    wp_if_destruct
    · exfalso; simp_all
    · wp_apply hbinding $$ [Hresources] with Hresources
      · iexact Hresources
      wp_end

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
