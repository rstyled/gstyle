module
public import Gstyle.Proof.Pdg.FactLayout
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI
namespace github_com.rstyled.gstyle.domain.pdg
namespace Producer
section def_

variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi] [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics]
variable [query.Assumptions] [Producer.TypeAssumptions] [Span.TypeAssumptions] [CountFact.TypeAssumptions] [Function.TypeAssumptions] [Conformance.TypeAssumptions] [AttributeTables.TypeAssumptions] [AttributeSet.TypeAssumptions] [Attributes.TypeAssumptions] [Node.TypeAssumptions] [Graph.TypeAssumptions] [Position.TypeAssumptions] [CallForm.TypeAssumptions] [ExitKind.TypeAssumptions] [ConformanceStatus.TypeAssumptions] [Category.TypeAssumptions] [Fact.TypeAssumptions]

instance Producer_typed_pointsto :
    TypedPointsto (GF := GF) github_com.rstyled.gstyle.domain.pdg.Producer where
  typedPointstoDef l v dq := iprop(
    "Name" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Producer go!"Name" l) v.Name' dq ∗
    "Version" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Producer go!"Version" l) v.Version' dq ∗
    "Language" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Producer go!"Language" l) v.Language' dq ∗
    "LanguageVersion" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Producer go!"LanguageVersion" l) v.LanguageVersion' dq ∗
    "Configuration" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Producer go!"Configuration" l) v.Configuration' dq ∗
    "_" ∷ True)
  typedPointstoDef_dfractional := by solve_typed_pointsto_dfractional
  typedPointstoDef_timeless := by solve_typed_pointsto_timeless
  typedPointsto_agree := by solve_typed_pointsto_agree

instance Producer_into_val_typed :
    IntoValTypedUnderlying (GF := GF) github_com.rstyled.gstyle.domain.pdg.Producer github_com.rstyled.gstyle.domain.pdg.Producer.underlying := by
  solve_into_val_typed_struct

end def_
end Producer

namespace Span
section def_

variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi] [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics]
variable [query.Assumptions] [Producer.TypeAssumptions] [Span.TypeAssumptions] [CountFact.TypeAssumptions] [Function.TypeAssumptions] [Conformance.TypeAssumptions] [AttributeTables.TypeAssumptions] [AttributeSet.TypeAssumptions] [Attributes.TypeAssumptions] [Node.TypeAssumptions] [Graph.TypeAssumptions] [Position.TypeAssumptions] [CallForm.TypeAssumptions] [ExitKind.TypeAssumptions] [ConformanceStatus.TypeAssumptions] [Category.TypeAssumptions] [Fact.TypeAssumptions]

instance Span_typed_pointsto :
    TypedPointsto (GF := GF) github_com.rstyled.gstyle.domain.pdg.Span where
  typedPointstoDef l v dq := iprop(
    "Source" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Span go!"Source" l) v.Source' dq ∗
    "Start" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Span go!"Start" l) v.Start' dq ∗
    "End" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Span go!"End" l) v.End' dq ∗
    "_" ∷ True)
  typedPointstoDef_dfractional := by solve_typed_pointsto_dfractional
  typedPointstoDef_timeless := by solve_typed_pointsto_timeless
  typedPointsto_agree := by solve_typed_pointsto_agree

instance Span_into_val_typed :
    IntoValTypedUnderlying (GF := GF) github_com.rstyled.gstyle.domain.pdg.Span github_com.rstyled.gstyle.domain.pdg.Span.underlying := by
  solve_into_val_typed_struct

end def_
end Span

namespace CountFact
section def_

variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi] [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics]
variable [query.Assumptions] [Producer.TypeAssumptions] [Span.TypeAssumptions] [CountFact.TypeAssumptions] [Function.TypeAssumptions] [Conformance.TypeAssumptions] [AttributeTables.TypeAssumptions] [AttributeSet.TypeAssumptions] [Attributes.TypeAssumptions] [Node.TypeAssumptions] [Graph.TypeAssumptions] [Position.TypeAssumptions] [CallForm.TypeAssumptions] [ExitKind.TypeAssumptions] [ConformanceStatus.TypeAssumptions] [Category.TypeAssumptions] [Fact.TypeAssumptions]

instance CountFact_typed_pointsto :
    TypedPointsto (GF := GF) github_com.rstyled.gstyle.domain.pdg.CountFact where
  typedPointstoDef l v dq := iprop(
    "Value" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.CountFact go!"Value" l) v.Value' dq ∗
    "Reason" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.CountFact go!"Reason" l) v.Reason' dq ∗
    "Status" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.CountFact go!"Status" l) v.Status' dq ∗
    "_" ∷ True)
  typedPointstoDef_dfractional := by solve_typed_pointsto_dfractional
  typedPointstoDef_timeless := by solve_typed_pointsto_timeless
  typedPointsto_agree := by solve_typed_pointsto_agree

instance CountFact_into_val_typed :
    IntoValTypedUnderlying (GF := GF) github_com.rstyled.gstyle.domain.pdg.CountFact github_com.rstyled.gstyle.domain.pdg.CountFact.underlying := by
  solve_into_val_typed_struct

end def_
end CountFact

namespace Function
section def_

variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi] [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics]
variable [query.Assumptions] [Producer.TypeAssumptions] [Span.TypeAssumptions] [CountFact.TypeAssumptions] [Function.TypeAssumptions] [Conformance.TypeAssumptions] [AttributeTables.TypeAssumptions] [AttributeSet.TypeAssumptions] [Attributes.TypeAssumptions] [Node.TypeAssumptions] [Graph.TypeAssumptions] [Position.TypeAssumptions] [CallForm.TypeAssumptions] [ExitKind.TypeAssumptions] [ConformanceStatus.TypeAssumptions] [Category.TypeAssumptions] [Fact.TypeAssumptions]

instance Function_typed_pointsto :
    TypedPointsto (GF := GF) github_com.rstyled.gstyle.domain.pdg.Function where
  typedPointstoDef l v dq := iprop(
    "ID" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Function go!"ID" l) v.ID' dq ∗
    "Name" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Function go!"Name" l) v.Name' dq ∗
    "QualifiedName" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Function go!"QualifiedName" l) v.QualifiedName' dq ∗
    "Span" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Function go!"Span" l) v.Span' dq ∗
    "Inputs" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Function go!"Inputs" l) v.Inputs' dq ∗
    "Outputs" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Function go!"Outputs" l) v.Outputs' dq ∗
    "Interface" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Function go!"Interface" l) v.Interface' dq ∗
    "ReferenceCount" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Function go!"ReferenceCount" l) v.ReferenceCount' dq ∗
    "_" ∷ True)
  typedPointstoDef_dfractional := by solve_typed_pointsto_dfractional
  typedPointstoDef_timeless := by solve_typed_pointsto_timeless
  typedPointsto_agree := by solve_typed_pointsto_agree

instance Function_into_val_typed :
    IntoValTypedUnderlying (GF := GF) github_com.rstyled.gstyle.domain.pdg.Function github_com.rstyled.gstyle.domain.pdg.Function.underlying := by
  solve_into_val_typed_struct

end def_
end Function

namespace Conformance
section def_

variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi] [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics]
variable [query.Assumptions] [Producer.TypeAssumptions] [Span.TypeAssumptions] [CountFact.TypeAssumptions] [Function.TypeAssumptions] [Conformance.TypeAssumptions] [AttributeTables.TypeAssumptions] [AttributeSet.TypeAssumptions] [Attributes.TypeAssumptions] [Node.TypeAssumptions] [Graph.TypeAssumptions] [Position.TypeAssumptions] [CallForm.TypeAssumptions] [ExitKind.TypeAssumptions] [ConformanceStatus.TypeAssumptions] [Category.TypeAssumptions] [Fact.TypeAssumptions]

instance Conformance_typed_pointsto :
    TypedPointsto (GF := GF) github_com.rstyled.gstyle.domain.pdg.Conformance where
  typedPointstoDef l v dq := iprop(
    "Reasons" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Conformance go!"Reasons" l) v.Reasons' dq ∗
    "Status" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Conformance go!"Status" l) v.Status' dq ∗
    "_" ∷ True)
  typedPointstoDef_dfractional := by solve_typed_pointsto_dfractional
  typedPointstoDef_timeless := by solve_typed_pointsto_timeless
  typedPointsto_agree := by solve_typed_pointsto_agree

instance Conformance_into_val_typed :
    IntoValTypedUnderlying (GF := GF) github_com.rstyled.gstyle.domain.pdg.Conformance github_com.rstyled.gstyle.domain.pdg.Conformance.underlying := by
  solve_into_val_typed_struct

end def_
end Conformance

namespace AttributeTables
section def_

variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi] [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics]
variable [query.Assumptions] [Producer.TypeAssumptions] [Span.TypeAssumptions] [CountFact.TypeAssumptions] [Function.TypeAssumptions] [Conformance.TypeAssumptions] [AttributeTables.TypeAssumptions] [AttributeSet.TypeAssumptions] [Attributes.TypeAssumptions] [Node.TypeAssumptions] [Graph.TypeAssumptions] [Position.TypeAssumptions] [CallForm.TypeAssumptions] [ExitKind.TypeAssumptions] [ConformanceStatus.TypeAssumptions] [Category.TypeAssumptions] [Fact.TypeAssumptions]

instance AttributeTables_typed_pointsto :
    TypedPointsto (GF := GF) github_com.rstyled.gstyle.domain.pdg.AttributeTables where
  typedPointstoDef l v dq := iprop(
    "References" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeTables go!"References" l) v.References' dq ∗
    "Accesses" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeTables go!"Accesses" l) v.Accesses' dq ∗
    "Extensions" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeTables go!"Extensions" l) v.Extensions' dq ∗
    "_" ∷ True)
  typedPointstoDef_dfractional := by solve_typed_pointsto_dfractional
  typedPointstoDef_timeless := by solve_typed_pointsto_timeless
  typedPointsto_agree := by solve_typed_pointsto_agree

instance AttributeTables_into_val_typed :
    IntoValTypedUnderlying (GF := GF) github_com.rstyled.gstyle.domain.pdg.AttributeTables github_com.rstyled.gstyle.domain.pdg.AttributeTables.underlying := by
  solve_into_val_typed_struct

instance AttributeTables_access_load_References (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.AttributeTables) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeTables go!"References" l) v.References' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeTables go!"References" l) v.References' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance AttributeTables_access_load_Accesses (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.AttributeTables) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeTables go!"Accesses" l) v.Accesses' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeTables go!"Accesses" l) v.Accesses' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance AttributeTables_access_load_Extensions (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.AttributeTables) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeTables go!"Extensions" l) v.Extensions' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeTables go!"Extensions" l) v.Extensions' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

end def_
end AttributeTables

namespace AttributeSet
section def_

variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi] [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics]
variable [query.Assumptions] [Producer.TypeAssumptions] [Span.TypeAssumptions] [CountFact.TypeAssumptions] [Function.TypeAssumptions] [Conformance.TypeAssumptions] [AttributeTables.TypeAssumptions] [AttributeSet.TypeAssumptions] [Attributes.TypeAssumptions] [Node.TypeAssumptions] [Graph.TypeAssumptions] [Position.TypeAssumptions] [CallForm.TypeAssumptions] [ExitKind.TypeAssumptions] [ConformanceStatus.TypeAssumptions] [Category.TypeAssumptions] [Fact.TypeAssumptions]

instance AttributeSet_typed_pointsto :
    TypedPointsto (GF := GF) github_com.rstyled.gstyle.domain.pdg.AttributeSet where
  typedPointstoDef l v dq := iprop(
    "Symbols" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Symbols" l) v.Symbols' dq ∗
    "Defines" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Defines" l) v.Defines' dq ∗
    "Reads" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Reads" l) v.Reads' dq ∗
    "Writes" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Writes" l) v.Writes' dq ∗
    "Outcomes" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Outcomes" l) v.Outcomes' dq ∗
    "Extensions" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Extensions" l) v.Extensions' dq ∗
    "Operator" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Operator" l) v.Operator' dq ∗
    "Literal" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Literal" l) v.Literal' dq ∗
    "Callee" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Callee" l) v.Callee' dq ∗
    "Construct" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Construct" l) v.Construct' dq ∗
    "WriteForm" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"WriteForm" l) v.WriteForm' dq ∗
    "Description" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Description" l) v.Description' dq ∗
    "Position" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Position" l) v.Position' dq ∗
    "CallForm" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"CallForm" l) v.CallForm' dq ∗
    "ExitKind" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"ExitKind" l) v.ExitKind' dq ∗
    "_" ∷ True)
  typedPointstoDef_dfractional := by solve_typed_pointsto_dfractional
  typedPointstoDef_timeless := by solve_typed_pointsto_timeless
  typedPointsto_agree := by solve_typed_pointsto_agree

instance AttributeSet_into_val_typed :
    IntoValTypedUnderlying (GF := GF) github_com.rstyled.gstyle.domain.pdg.AttributeSet github_com.rstyled.gstyle.domain.pdg.AttributeSet.underlying := by
  solve_into_val_typed_struct

instance AttributeSet_access_load_Symbols (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.AttributeSet) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Symbols" l) v.Symbols' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Symbols" l) v.Symbols' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance AttributeSet_access_load_Defines (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.AttributeSet) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Defines" l) v.Defines' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Defines" l) v.Defines' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance AttributeSet_access_load_Reads (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.AttributeSet) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Reads" l) v.Reads' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Reads" l) v.Reads' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance AttributeSet_access_load_Writes (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.AttributeSet) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Writes" l) v.Writes' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Writes" l) v.Writes' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance AttributeSet_access_load_Outcomes (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.AttributeSet) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Outcomes" l) v.Outcomes' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Outcomes" l) v.Outcomes' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance AttributeSet_access_load_Extensions (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.AttributeSet) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Extensions" l) v.Extensions' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Extensions" l) v.Extensions' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance AttributeSet_access_load_Operator (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.AttributeSet) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Operator" l) v.Operator' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Operator" l) v.Operator' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance AttributeSet_access_load_Literal (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.AttributeSet) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Literal" l) v.Literal' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Literal" l) v.Literal' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance AttributeSet_access_load_Callee (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.AttributeSet) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Callee" l) v.Callee' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Callee" l) v.Callee' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance AttributeSet_access_load_Construct (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.AttributeSet) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Construct" l) v.Construct' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Construct" l) v.Construct' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance AttributeSet_access_load_WriteForm (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.AttributeSet) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"WriteForm" l) v.WriteForm' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"WriteForm" l) v.WriteForm' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance AttributeSet_access_load_Description (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.AttributeSet) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Description" l) v.Description' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Description" l) v.Description' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance AttributeSet_access_load_Position (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.AttributeSet) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Position" l) v.Position' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"Position" l) v.Position' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance AttributeSet_access_load_CallForm (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.AttributeSet) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"CallForm" l) v.CallForm' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"CallForm" l) v.CallForm' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance AttributeSet_access_load_ExitKind (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.AttributeSet) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"ExitKind" l) v.ExitKind' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.AttributeSet go!"ExitKind" l) v.ExitKind' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

end def_
end AttributeSet

namespace Attributes
section def_

variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi] [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics]
variable [query.Assumptions] [Producer.TypeAssumptions] [Span.TypeAssumptions] [CountFact.TypeAssumptions] [Function.TypeAssumptions] [Conformance.TypeAssumptions] [AttributeTables.TypeAssumptions] [AttributeSet.TypeAssumptions] [Attributes.TypeAssumptions] [Node.TypeAssumptions] [Graph.TypeAssumptions] [Position.TypeAssumptions] [CallForm.TypeAssumptions] [ExitKind.TypeAssumptions] [ConformanceStatus.TypeAssumptions] [Category.TypeAssumptions] [Fact.TypeAssumptions]

instance Attributes_typed_pointsto :
    TypedPointsto (GF := GF) github_com.rstyled.gstyle.domain.pdg.Attributes where
  typedPointstoDef l v dq := iprop(
    "Symbols" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Attributes go!"Symbols" l) v.Symbols' dq ∗
    "Defines" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Attributes go!"Defines" l) v.Defines' dq ∗
    "Reads" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Attributes go!"Reads" l) v.Reads' dq ∗
    "Writes" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Attributes go!"Writes" l) v.Writes' dq ∗
    "Operator" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Attributes go!"Operator" l) v.Operator' dq ∗
    "Literal" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Attributes go!"Literal" l) v.Literal' dq ∗
    "Callee" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Attributes go!"Callee" l) v.Callee' dq ∗
    "Construct" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Attributes go!"Construct" l) v.Construct' dq ∗
    "WriteForm" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Attributes go!"WriteForm" l) v.WriteForm' dq ∗
    "Description" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Attributes go!"Description" l) v.Description' dq ∗
    "Outcomes" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Attributes go!"Outcomes" l) v.Outcomes' dq ∗
    "Position" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Attributes go!"Position" l) v.Position' dq ∗
    "CallForm" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Attributes go!"CallForm" l) v.CallForm' dq ∗
    "ExitKind" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Attributes go!"ExitKind" l) v.ExitKind' dq ∗
    "Extensions" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Attributes go!"Extensions" l) v.Extensions' dq ∗
    "_" ∷ True)
  typedPointstoDef_dfractional := by solve_typed_pointsto_dfractional
  typedPointstoDef_timeless := by solve_typed_pointsto_timeless
  typedPointsto_agree := by solve_typed_pointsto_agree

instance Attributes_into_val_typed :
    IntoValTypedUnderlying (GF := GF) github_com.rstyled.gstyle.domain.pdg.Attributes github_com.rstyled.gstyle.domain.pdg.Attributes.underlying := by
  solve_into_val_typed_struct

instance Attributes_access_load_Reads (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Attributes) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Attributes go!"Reads" l) v.Reads' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Attributes go!"Reads" l) v.Reads' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) := by
  solve_pointsto_access_struct

instance Attributes_access_load_Operator (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Attributes) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Attributes go!"Operator" l) v.Operator' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Attributes go!"Operator" l) v.Operator' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) := by
  solve_pointsto_access_struct

instance Attributes_access_load_Construct (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Attributes) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Attributes go!"Construct" l) v.Construct' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Attributes go!"Construct" l) v.Construct' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) := by
  solve_pointsto_access_struct

instance Attributes_access_load_Callee (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Attributes) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Attributes go!"Callee" l) v.Callee' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Attributes go!"Callee" l) v.Callee' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) := by
  solve_pointsto_access_struct

end def_
end Attributes

namespace Node
section def_

variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi] [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics]
variable [query.Assumptions] [Producer.TypeAssumptions] [Span.TypeAssumptions] [CountFact.TypeAssumptions] [Function.TypeAssumptions] [Conformance.TypeAssumptions] [AttributeTables.TypeAssumptions] [AttributeSet.TypeAssumptions] [Attributes.TypeAssumptions] [Node.TypeAssumptions] [Graph.TypeAssumptions] [Position.TypeAssumptions] [CallForm.TypeAssumptions] [ExitKind.TypeAssumptions] [ConformanceStatus.TypeAssumptions] [Category.TypeAssumptions] [Fact.TypeAssumptions]

instance Node_typed_pointsto :
    TypedPointsto (GF := GF) github_com.rstyled.gstyle.domain.pdg.Node where
  typedPointstoDef l v dq := iprop(
    "MatchLabel" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Node go!"MatchLabel" l) v.MatchLabel' dq ∗
    "Line" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Node go!"Line" l) v.Line' dq ∗
    "ID" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Node go!"ID" l) v.ID' dq ∗
    "OriginalKind" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Node go!"OriginalKind" l) v.OriginalKind' dq ∗
    "Span" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Node go!"Span" l) v.Span' dq ∗
    "Attributes" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Node go!"Attributes" l) v.Attributes' dq ∗
    "Provenance" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Node go!"Provenance" l) v.Provenance' dq ∗
    "Evidence" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Node go!"Evidence" l) v.Evidence' dq ∗
    "Annotations" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Node go!"Annotations" l) v.Annotations' dq ∗
    "Category" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Node go!"Category" l) v.Category' dq ∗
    "Synthetic" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Node go!"Synthetic" l) v.Synthetic' dq ∗
    "_" ∷ True)
  typedPointstoDef_dfractional := by solve_typed_pointsto_dfractional
  typedPointstoDef_timeless := by solve_typed_pointsto_timeless
  typedPointsto_agree := by solve_typed_pointsto_agree

instance Node_into_val_typed :
    IntoValTypedUnderlying (GF := GF) github_com.rstyled.gstyle.domain.pdg.Node github_com.rstyled.gstyle.domain.pdg.Node.underlying := by
  solve_into_val_typed_struct

instance Node_access_load_Attributes (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Node) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Node go!"Attributes" l) v.Attributes' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Node go!"Attributes" l) v.Attributes' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Node_access_load_Category (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Node) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Node go!"Category" l) v.Category' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Node go!"Category" l) v.Category' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) := by
  solve_pointsto_access_struct

end def_
end Node

namespace Graph
section def_

variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi] [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics]
variable [query.Assumptions] [Producer.TypeAssumptions] [Span.TypeAssumptions] [CountFact.TypeAssumptions] [Function.TypeAssumptions] [Conformance.TypeAssumptions] [AttributeTables.TypeAssumptions] [AttributeSet.TypeAssumptions] [Attributes.TypeAssumptions] [Node.TypeAssumptions] [Graph.TypeAssumptions] [Position.TypeAssumptions] [CallForm.TypeAssumptions] [ExitKind.TypeAssumptions] [ConformanceStatus.TypeAssumptions] [Category.TypeAssumptions] [Fact.TypeAssumptions]

instance Graph_typed_pointsto :
    TypedPointsto (GF := GF) github_com.rstyled.gstyle.domain.pdg.Graph where
  typedPointstoDef l v dq := iprop(
    "Tables" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"Tables" l) v.Tables' dq ∗
    "NormalFlow" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"NormalFlow" l) v.NormalFlow' dq ∗
    "ValueFlow" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"ValueFlow" l) v.ValueFlow' dq ∗
    "Profile" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"Profile" l) v.Profile' dq ∗
    "Producer" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"Producer" l) v.Producer' dq ∗
    "Sources" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"Sources" l) v.Sources' dq ∗
    "Function" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"Function" l) v.Function' dq ∗
    "Symbols" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"Symbols" l) v.Symbols' dq ∗
    "Definitions" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"Definitions" l) v.Definitions' dq ∗
    "Locations" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"Locations" l) v.Locations' dq ∗
    "Nodes" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"Nodes" l) v.Nodes' dq ∗
    "Edges" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"Edges" l) v.Edges' dq ∗
    "Spans" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"Spans" l) v.Spans' dq ∗
    "Attributes" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"Attributes" l) v.Attributes' dq ∗
    "AttributeTables" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"AttributeTables" l) v.AttributeTables' dq ∗
    "Annotations" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"Annotations" l) v.Annotations' dq ∗
    "Capabilities" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"Capabilities" l) v.Capabilities' dq ∗
    "Diagnostics" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"Diagnostics" l) v.Diagnostics' dq ∗
    "Conformance" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"Conformance" l) v.Conformance' dq ∗
    "Extensions" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"Extensions" l) v.Extensions' dq ∗
    "_" ∷ True)
  typedPointstoDef_dfractional := by solve_typed_pointsto_dfractional
  typedPointstoDef_timeless := by solve_typed_pointsto_timeless
  typedPointsto_agree := by solve_typed_pointsto_agree

instance Graph_into_val_typed :
    IntoValTypedUnderlying (GF := GF) github_com.rstyled.gstyle.domain.pdg.Graph github_com.rstyled.gstyle.domain.pdg.Graph.underlying := by
  solve_into_val_typed_struct

instance Graph_access_load_Attributes (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Graph) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"Attributes" l) v.Attributes' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"Attributes" l) v.Attributes' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Graph_access_load_AttributeTables (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Graph) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"AttributeTables" l) v.AttributeTables' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"AttributeTables" l) v.AttributeTables' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Graph_access_load_Tables (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Graph) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"Tables" l) v.Tables' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"Tables" l) v.Tables' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Graph_access_load_Nodes (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Graph) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"Nodes" l) v.Nodes' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Graph go!"Nodes" l) v.Nodes' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

end def_
end Graph
end github_com.rstyled.gstyle.domain.pdg
end Perennial
