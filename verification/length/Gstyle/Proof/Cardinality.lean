module
public import Gstyle.Proof.Length
@[expose] public section
namespace Perennial.github_com.rstyled.gstyle.domain.quality.linecount

/-- The integer increment in the Go proof equals the cardinality of the new
physical line numbers, including the empty contribution of a duplicate token. -/
theorem freshLines_length_int (through first last : Int)
    (ht : 0 ≤ through) (hf : 1 ≤ first) (hl : first ≤ last) (ho : through ≤ first) :
    ((freshLines through.toNat first.toNat last.toNat).length : Int) =
      last - max first (through + 1) + 1 := by
  rw [freshLines_length]
  omega
end Perennial.github_com.rstyled.gstyle.domain.quality.linecount

namespace Perennial.github_com.rstyled.gstyle.domain.quality.linecount
/-- Contributions compose without duplicates: no previously counted line can
occur in a source-ordered span's fresh contribution. -/
theorem freshLines_extend (old : List Nat) (through first last : Nat)
    (unique : old.Nodup) (bounded : ∀ n ∈ old, n ≤ through) :
    (old ++ freshLines through first last).Nodup ∧
    (old ++ freshLines through first last).length =
      old.length + (freshLines through first last).length := by
  constructor
  · rw [List.nodup_append]
    refine ⟨unique, freshLines_nodup .., ?_⟩
    intro a ha b hb hab
    have prior := bounded a ha
    have fresh := (freshLines_mem through first last b).mp hb
    omega
  · exact List.length_append
end Perennial.github_com.rstyled.gstyle.domain.quality.linecount
