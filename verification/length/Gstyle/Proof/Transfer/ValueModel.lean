module
public import Gstyle.Proof.Transfer.FactSnapshot
public import Gstyle.Proof.Transfer.ConstantModel
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

def snapshotConstant [FfiSyntax] (s : QuerySnapshot) (ref : w32) : Value :=
  constantRecord (snapshotFact s ref (go!"non-nil")) (snapshotFact s ref (go!"nil"))
    (decimalInt64 (snapshotFact s ref (go!"integer"))) (decimalInt64 (snapshotFact s ref (go!"literal-extent")))

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
