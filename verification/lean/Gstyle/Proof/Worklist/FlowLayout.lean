module
public import Gstyle.Proof.Pdg.GraphLayout
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI
namespace github_com.rstyled.gstyle.domain.pdg

namespace NormalFlow
section def_

variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi] [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics]
variable [NormalFlow.TypeAssumptions] [query.Assumptions]

instance NormalFlow_typed_pointsto :
    TypedPointsto (GF := GF) github_com.rstyled.gstyle.domain.pdg.NormalFlow where
  typedPointstoDef l v dq := iprop(
    "ExitContext" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.NormalFlow go!"ExitContext" l) v.ExitContext' dq ∗
    "Blocks" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.NormalFlow go!"Blocks" l) v.Blocks' dq ∗
    "Entry" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.NormalFlow go!"Entry" l) v.Entry' dq ∗
    "ReadBindings" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.NormalFlow go!"ReadBindings" l) v.ReadBindings' dq ∗
    "_" ∷ True)
  typedPointstoDef_dfractional := by solve_typed_pointsto_dfractional
  typedPointstoDef_timeless := by solve_typed_pointsto_timeless
  typedPointsto_agree := by solve_typed_pointsto_agree

instance NormalFlow_into_val_typed :
    IntoValTypedUnderlying (GF := GF) github_com.rstyled.gstyle.domain.pdg.NormalFlow github_com.rstyled.gstyle.domain.pdg.NormalFlow.underlying := by
  solve_into_val_typed_struct

instance NormalFlow_access_load_Blocks (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.NormalFlow) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.NormalFlow go!"Blocks" l) v.Blocks' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.NormalFlow go!"Blocks" l) v.Blocks' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance NormalFlow_access_load_Entry (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.NormalFlow) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.NormalFlow go!"Entry" l) v.Entry' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.NormalFlow go!"Entry" l) v.Entry' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

end def_
end NormalFlow

namespace FlowBlock
section def_

variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi] [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics]
variable [FlowBlock.TypeAssumptions] [query.Assumptions]

instance FlowBlock_typed_pointsto :
    TypedPointsto (GF := GF) github_com.rstyled.gstyle.domain.pdg.FlowBlock where
  typedPointstoDef l v dq := iprop(
    "Operations" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.FlowBlock go!"Operations" l) v.Operations' dq ∗
    "Successors" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.FlowBlock go!"Successors" l) v.Successors' dq ∗
    "Condition" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.FlowBlock go!"Condition" l) v.Condition' dq ∗
    "Reachable" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.FlowBlock go!"Reachable" l) v.Reachable' dq ∗
    "_" ∷ True)
  typedPointstoDef_dfractional := by solve_typed_pointsto_dfractional
  typedPointstoDef_timeless := by solve_typed_pointsto_timeless
  typedPointsto_agree := by solve_typed_pointsto_agree

instance FlowBlock_into_val_typed :
    IntoValTypedUnderlying (GF := GF) github_com.rstyled.gstyle.domain.pdg.FlowBlock github_com.rstyled.gstyle.domain.pdg.FlowBlock.underlying := by
  solve_into_val_typed_struct

instance FlowBlock_access_load_Operations (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.FlowBlock) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.FlowBlock go!"Operations" l) v.Operations' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.FlowBlock go!"Operations" l) v.Operations' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance FlowBlock_access_load_Successors (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.FlowBlock) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.FlowBlock go!"Successors" l) v.Successors' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.FlowBlock go!"Successors" l) v.Successors' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance FlowBlock_access_load_Condition (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.FlowBlock) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.FlowBlock go!"Condition" l) v.Condition' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.FlowBlock go!"Condition" l) v.Condition' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

end def_
end FlowBlock

end github_com.rstyled.gstyle.domain.pdg
end Perennial
