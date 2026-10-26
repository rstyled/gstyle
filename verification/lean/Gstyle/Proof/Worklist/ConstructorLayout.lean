module
public import Gstyle.Proof.Worklist.FlowLayout
public import Gstyle.Proof.StateMap.Layout
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI
namespace github_com.rstyled.gstyle.domain.pdg
section layout
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [query.Assumptions]
variable [Symbol.TypeAssumptions] [Role.TypeAssumptions] [Fact.TypeAssumptions]
variable [Edge.TypeAssumptions] [EdgeKind.TypeAssumptions] [Position.TypeAssumptions]
namespace Symbol
instance Symbol_typed_pointsto : TypedPointsto (GF := GF) Symbol where
  typedPointstoDef l v dq := iprop(
    "ID" ∷ typedPointsto (structFieldRef Symbol go!"ID" l) v.ID' dq ∗
    "Scope" ∷ typedPointsto (structFieldRef Symbol go!"Scope" l) v.Scope' dq ∗
    "Name" ∷ typedPointsto (structFieldRef Symbol go!"Name" l) v.Name' dq ∗
    "Type" ∷ typedPointsto (structFieldRef Symbol go!"Type" l) v.Type' dq ∗
    "Role" ∷ typedPointsto (structFieldRef Symbol go!"Role" l) v.Role' dq ∗
    "Extensions" ∷ typedPointsto (structFieldRef Symbol go!"Extensions" l) v.Extensions' dq ∗
    "_" ∷ True)
  typedPointstoDef_dfractional := by solve_typed_pointsto_dfractional
  typedPointstoDef_timeless := by solve_typed_pointsto_timeless
  typedPointsto_agree := by solve_typed_pointsto_agree
instance Symbol_into_val_typed : IntoValTypedUnderlying (GF := GF) Symbol Symbol.underlying := by
  solve_into_val_typed_struct
instance Symbol_access_load_Role (l : Loc) (v : Symbol) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef Symbol go!"Role" l) v.Role' dq)
      (typedPointsto (structFieldRef Symbol go!"Role" l) v.Role' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) := by
  solve_pointsto_access_struct
instance Symbol_access_load_Extensions (l : Loc) (v : Symbol) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef Symbol go!"Extensions" l) v.Extensions' dq)
      (typedPointsto (structFieldRef Symbol go!"Extensions" l) v.Extensions' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) := by
  solve_pointsto_access_struct
end Symbol
namespace Edge
instance Edge_typed_pointsto : TypedPointsto (GF := GF) Edge where
  typedPointstoDef l v dq := iprop(
    "ID" ∷ typedPointsto (structFieldRef Edge go!"ID" l) v.ID' dq ∗
    "Subkind" ∷ typedPointsto (structFieldRef Edge go!"Subkind" l) v.Subkind' dq ∗
    "Source" ∷ typedPointsto (structFieldRef Edge go!"Source" l) v.Source' dq ∗
    "Target" ∷ typedPointsto (structFieldRef Edge go!"Target" l) v.Target' dq ∗
    "Evidence" ∷ typedPointsto (structFieldRef Edge go!"Evidence" l) v.Evidence' dq ∗
    "Provenance" ∷ typedPointsto (structFieldRef Edge go!"Provenance" l) v.Provenance' dq ∗
    "Annotations" ∷ typedPointsto (structFieldRef Edge go!"Annotations" l) v.Annotations' dq ∗
    "Symbol" ∷ typedPointsto (structFieldRef Edge go!"Symbol" l) v.Symbol' dq ∗
    "Definition" ∷ typedPointsto (structFieldRef Edge go!"Definition" l) v.Definition' dq ∗
    "Location" ∷ typedPointsto (structFieldRef Edge go!"Location" l) v.Location' dq ∗
    "Outcome" ∷ typedPointsto (structFieldRef Edge go!"Outcome" l) v.Outcome' dq ∗
    "Position" ∷ typedPointsto (structFieldRef Edge go!"Position" l) v.Position' dq ∗
    "Kind" ∷ typedPointsto (structFieldRef Edge go!"Kind" l) v.Kind' dq ∗
    "LoopCarried" ∷ typedPointsto (structFieldRef Edge go!"LoopCarried" l) v.LoopCarried' dq ∗
    "_" ∷ True)
  typedPointstoDef_dfractional := by solve_typed_pointsto_dfractional
  typedPointstoDef_timeless := by solve_typed_pointsto_timeless
  typedPointsto_agree := by solve_typed_pointsto_agree
instance Edge_into_val_typed : IntoValTypedUnderlying (GF := GF) Edge Edge.underlying := by
  solve_into_val_typed_struct
instance Edge_access_load_Subkind (l : Loc) (v : Edge) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef Edge go!"Subkind" l) v.Subkind' dq)
      (typedPointsto (structFieldRef Edge go!"Subkind" l) v.Subkind' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) := by
  solve_pointsto_access_struct
instance Edge_access_load_Source (l : Loc) (v : Edge) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef Edge go!"Source" l) v.Source' dq)
      (typedPointsto (structFieldRef Edge go!"Source" l) v.Source' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) := by
  solve_pointsto_access_struct
instance Edge_access_load_Target (l : Loc) (v : Edge) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef Edge go!"Target" l) v.Target' dq)
      (typedPointsto (structFieldRef Edge go!"Target" l) v.Target' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) := by
  solve_pointsto_access_struct
instance Edge_access_load_Position (l : Loc) (v : Edge) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef Edge go!"Position" l) v.Position' dq)
      (typedPointsto (structFieldRef Edge go!"Position" l) v.Position' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) := by
  solve_pointsto_access_struct
instance Edge_access_load_Kind (l : Loc) (v : Edge) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef Edge go!"Kind" l) v.Kind' dq)
      (typedPointsto (structFieldRef Edge go!"Kind" l) v.Kind' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) := by
  solve_pointsto_access_struct
end Edge
variable [Graph.TypeAssumptions] [Node.TypeAssumptions] [AttributeSet.TypeAssumptions]
variable [AttributeTables.TypeAssumptions] [Producer.TypeAssumptions] [Function.TypeAssumptions]
variable [Span.TypeAssumptions] [NormalFlow.TypeAssumptions] [CountFact.TypeAssumptions]
variable [Conformance.TypeAssumptions] [CallForm.TypeAssumptions] [ExitKind.TypeAssumptions]
variable [ConformanceStatus.TypeAssumptions] [Category.TypeAssumptions]
instance Graph_access_load_Symbols (l : Loc) (v : Graph) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef Graph go!"Symbols" l) v.Symbols' dq)
      (typedPointsto (structFieldRef Graph go!"Symbols" l) v.Symbols' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) := by
  solve_pointsto_access_struct
instance Graph_access_load_Edges (l : Loc) (v : Graph) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef Graph go!"Edges" l) v.Edges' dq)
      (typedPointsto (structFieldRef Graph go!"Edges" l) v.Edges' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) := by
  solve_pointsto_access_struct
end layout
end github_com.rstyled.gstyle.domain.pdg
end Perennial
