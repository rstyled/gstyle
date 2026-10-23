module
public import Gstyle.Proof.Worklist.Layout
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [analysis.TypeAssumptions] [pdg.query.Assumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

/-- Exact execution of the production enqueue body, including its deduplication guard. -/
theorem wp_enqueue (receiver : Loc) (a : analysis) (flags : List Bool)
    (queue : List w32) (ref : w32) (present : Bool)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ flags.length)
    (hflag : flags[(uint.Z ref - 1).toNat]? = some present) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ receiver ↦ a ∗ a.queued' ↦* flags ∗
        a.queue' ↦* queue ∗ ownSliceCap w32 a.queue' (DFrac.own 1) }}
      (App (Val #(methods (go.PointerType analysis.ty) go!"enqueue" #receiver)) (Val #ref))
    {{ RET #(); ∃ (out : GoSlice), receiver ↦ { a with queue' := out } ∗
        a.queued' ↦* (if present then flags else flags.set (uint.Z ref - 1).toNat true) ∗
        out ↦* (if present then queue else queue ++ [ref]) ∗
        ownSliceCap w32 out (DFrac.own 1) }} := by
  wp_start as ⟨Ha, Hflags, Hqueue, Hcap⟩
  ihave %Hlen := ownSlice_len _ _ _ $$ Hflags
  wp_auto
  have Hi : sint.Z (W64 (uint.Z (ref - W32 1))) = uint.Z ref - 1 := by
    obtain ⟨_, _⟩ := href
    word
  rw [Hi]
  rw [ite_eq_left (by constructor <;> word :
    0 ≤ uint.Z ref - 1 ∧ uint.Z ref - 1 < sint.Z a.queued'.len)]
  wp_apply wp_load_slice_index a.queued' (uint.Z ref - 1) flags (DFrac.own 1) present
      (by omega) $$ [Hflags] with Hflags
  · iframe; ipureintro; exact hflag
  cases present
  · wp_auto
    rw [Hi]
    rw [ite_eq_left (by constructor <;> word :
      0 ≤ uint.Z ref - 1 ∧ uint.Z ref - 1 < sint.Z a.queued'.len)]
    wp_apply wp_store_slice_index a.queued' (uint.Z ref - 1) flags true $$ [Hflags] with Hflags
    · iframe; ipureintro; constructor <;> omega
    wp_apply wp_slice_literal (V := w32) [ref]
    isplitr
    · ipureintro; rfl
    iintro %one ⟨Hone, -⟩
    wp_auto
    wp_apply wp_slice_append $$ [Hqueue Hcap Hone] with %out ⟨Hqueue, Hcap, -⟩
    · iframe
    iapply HΦ
    iexists out
    iframe
  · wp_auto
    iapply HΦ
    iexists a.queue'
    iframe

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
