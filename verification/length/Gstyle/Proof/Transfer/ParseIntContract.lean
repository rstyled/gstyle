module
public import Gstyle.Proof.Transfer.ValueModel
public import Gstyle.Proof.StrconvPackage
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

def parseInt64Result [FfiSyntax] (bytes : GoString) (value : w64) (err : GoError) : Prop :=
  match decimalInt64 bytes with
  | none => err ≠ interface.nil
  | some integer => err = interface.nil ∧ value = W64 integer

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics]

def parseInt64Contract : Prop :=
  ∀ bytes : GoString,
    {{ isPkgInit (PROP := IProp GF) pkg_id.strconv }}
      (App (App (App (Val (@! strconv.ParseInt)) (Val #bytes)) (Val #(W64 10))) (Val #(W64 64)))
    {{ (value : w64) (err : GoError), RET (PairV #value #err);
      ⌜parseInt64Result bytes value err⌝ }}

end proof

theorem parseInt64Result_success [FfiSyntax] (bytes : GoString) (value : w64)
    (h : parseInt64Result bytes value interface.nil) :
    ∃ integer, decimalInt64 bytes = some integer ∧ value = W64 integer ∧
      -(2^63) ≤ integer ∧ integer < 2^63 := by
  unfold parseInt64Result at h
  cases hm : decimalInt64 bytes with
  | none => simp [hm] at h
  | some integer =>
    simp only [hm] at h
    exact ⟨integer, rfl, h.2, decimalInt64_range bytes integer hm⟩

theorem parseInt64Result_error [FfiSyntax] (bytes : GoString) (value : w64) (err : GoError)
    (h : parseInt64Result bytes value err) (he : err ≠ interface.nil) :
    decimalInt64 bytes = none := by
  unfold parseInt64Result at h
  cases hm : decimalInt64 bytes with
  | none => rfl
  | some integer =>
    simp only [hm] at h
    exact False.elim (he h.1)

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
