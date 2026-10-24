module
public import Gstyle.Proof.Worklist.Enqueue
@[expose] public section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

/-- Production references are one-based; zero is never a block. -/
def queueIndex (ref : w32) : Nat := (uint.Z ref - 1).toNat

def queueRefValid (flags : List Bool) (ref : w32) : Prop :=
  0 < uint.Z ref ∧ uint.Z ref ≤ flags.length

theorem queueIndex_lt (flags : List Bool) (ref : w32) (h : queueRefValid flags ref) :
    queueIndex ref < flags.length := by
  unfold queueRefValid at h
  unfold queueIndex
  omega

theorem queueIndex_injective (left right : w32)
    (hl : 0 < uint.Z left) (hr : 0 < uint.Z right) :
    queueIndex left = queueIndex right ↔ left = right := by
  constructor
  · intro h
    have hz : uint.Z left = uint.Z right := by
      unfold queueIndex at h
      omega
    word
  · intro h; subst right; rfl

/-- Flags describe only the unprocessed suffix, not entries already visited. -/
structure QueueInvariant (flags : List Bool) (cursor : Nat) (queue : List w32) : Prop where
  position : cursor ≤ queue.length
  references : ∀ ref ∈ queue.drop cursor, queueRefValid flags ref
  unique : (queue.drop cursor).Nodup
  marked : ∀ ref, queueRefValid flags ref →
    (flags[queueIndex ref]? = some true ↔ ref ∈ queue.drop cursor)

theorem QueueInvariant.enqueue (flags : List Bool) (cursor : Nat) (queue : List w32)
    (ref : w32) (h : QueueInvariant flags cursor queue) (href : queueRefValid flags ref)
    (hf : flags[queueIndex ref]? = some false) :
    QueueInvariant (flags.set (queueIndex ref) true) cursor (queue ++ [ref]) := by
  have hi := queueIndex_lt flags ref href
  have hn : ref ∉ queue.drop cursor := by
    intro hm
    have ht := (h.marked ref href).mpr hm
    rw [hf] at ht
    contradiction
  have hd : (queue ++ [ref]).drop cursor = queue.drop cursor ++ [ref] := by
    rw [List.drop_append]
    simp [Nat.sub_eq_zero_of_le h.position]
  constructor
  · simp only [List.length_append, List.length_singleton]
    have hp := h.position
    omega
  · intro key hk
    rw [hd] at hk
    rcases List.mem_append.mp hk with hk | hk
    · simpa [queueRefValid] using h.references key hk
    · have he : key = ref := by simpa using hk
      subst key
      simpa [queueRefValid] using href
  · rw [hd, List.nodup_append]
    refine ⟨h.unique, by simp, ?_⟩
    intro key hk other ho
    have he : other = ref := by simpa using ho
    subst other
    intro he
    subst key
    exact hn hk
  · intro key hk
    have hk' : queueRefValid flags key := by simpa [queueRefValid] using hk
    rw [hd]
    by_cases he : key = ref
    · subst key
      simp [hi]
    · have hine : queueIndex ref ≠ queueIndex key := by
        intro heq
        exact he ((queueIndex_injective ref key href.1 hk'.1).mp heq).symm
      rw [List.getElem?_set_ne hine]
      simpa [he] using h.marked key hk'

/-- The dequeue step clears exactly the head flag; earlier entries may be re-enqueued. -/
theorem QueueInvariant.advance (flags : List Bool) (cursor : Nat) (queue : List w32)
    (ref : w32) (h : QueueInvariant flags cursor queue)
    (head : queue[cursor]? = some ref) :
    QueueInvariant (flags.set (queueIndex ref) false) (cursor + 1) queue := by
  obtain ⟨hc, he⟩ := List.getElem?_eq_some_iff.mp head
  have hd : queue.drop cursor = ref :: queue.drop (cursor + 1) := by
    rw [List.drop_eq_getElem_cons hc, he]
  have hr := h.references ref (by rw [hd]; simp)
  have hu : ref ∉ queue.drop (cursor + 1) ∧ (queue.drop (cursor + 1)).Nodup := by
    simpa [hd] using h.unique
  constructor
  · omega
  · intro key hk
    have old : key ∈ queue.drop cursor := by rw [hd]; simp [hk]
    simpa [queueRefValid] using h.references key old
  · exact hu.2
  · intro key hk
    have hk' : queueRefValid flags key := by simpa [queueRefValid] using hk
    by_cases heq : key = ref
    · subst key
      simp [queueIndex_lt flags ref hr, hu.1]
    · have hine : queueIndex ref ≠ queueIndex key := by
        intro hi
        exact heq ((queueIndex_injective ref key hr.1 hk'.1).mp hi).symm
      rw [List.getElem?_set_ne hine]
      simpa [hd, heq] using h.marked key hk'

theorem QueueInvariant.exhausted (flags : List Bool) (queue : List w32)
    (h : QueueInvariant flags queue.length queue) (ref : w32)
    (href : queueRefValid flags ref) : flags[queueIndex ref]? = some false := by
  have hi := queueIndex_lt flags ref href
  have hf : flags[queueIndex ref]? ≠ some true := by simpa using h.marked ref href
  have hex := List.getElem?_eq_getElem hi
  cases hb : flags[queueIndex ref] <;> simp_all

section execution
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [analysis.TypeAssumptions] [pdg.query.Assumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

/-- Production enqueue preserves the pending-suffix invariant, including repeated requests. -/
theorem wp_enqueue_invariant (receiver : Loc) (a : analysis) (flags : List Bool)
    (queue : List w32) (cursor : Nat) (ref : w32) (present : Bool)
    (h : QueueInvariant flags cursor queue) (href : queueRefValid flags ref)
    (hflag : flags[queueIndex ref]? = some present) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ receiver ↦ a ∗ a.queued' ↦* flags ∗
        a.queue' ↦* queue ∗ ownSliceCap w32 a.queue' (DFrac.own 1) }}
      (App (Val #(methods (go.PointerType analysis.ty) go!"enqueue" #receiver)) (Val #ref))
    {{ RET #(); ∃ (out : GoSlice),
        receiver ↦ { a with queue' := out } ∗
        a.queued' ↦* (if present then flags else flags.set (queueIndex ref) true) ∗
        out ↦* (if present then queue else queue ++ [ref]) ∗
        ownSliceCap w32 out (DFrac.own 1) ∗
        ⌜QueueInvariant (if present then flags else flags.set (queueIndex ref) true)
          cursor (if present then queue else queue ++ [ref])⌝ }} := by
  wp_start_folded as ⟨Ha, Hflags, Hqueue, Hcap⟩
  wp_apply wp_enqueue receiver a flags queue ref present href hflag
      $$ [Ha Hflags Hqueue Hcap] with ⟨%out, Ha, Hflags, Hqueue, Hcap⟩
  · iframe
  iapply HΦ
  iexists out
  iframe
  ipureintro
  cases present
  · exact QueueInvariant.enqueue flags cursor queue ref h href hflag
  · exact h

end execution

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
