module
public import Gstyle.Proof.RuntimeValueModel
public import Gstyle.Proof.StateMap.Model
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [FfiSyntax]

def isExact (v : Value) : Bool := v.HasLower' && v.HasUpper' && decide (v.Lower' = v.Upper')
def isEmpty (v : Value) : Bool := v.HasLower' && v.HasUpper' && decide (sint.Z v.Lower' > sint.Z v.Upper')
def lengthConstrained (v : Value) (op : GoString) (b : Value) : Value :=
  if b.Length' && decide (op = go!"<" ∨ op = go!"<=") then
    { v with UpperChecked' := true, BelowLength' := v.BelowLength' || decide (op = go!"<") }
  else v

def numericConstrained (v : Value) (op : GoString) (b : Value) : Value :=
  let v := lengthConstrained v op b
  if isExact b then constrainedValue v op b.Lower' else v

def impossibleComparison (kind op : GoString) (b : Value) : Bool :=
  decide (kind = go!"\"signed\"") && isExact b &&
    ((decide (op = go!"<") && decide (b.Lower' = 9223372036854775808#64)) ||
     (decide (op = go!">") && decide (b.Lower' = W64 9223372036854775807)))

def constraintValue (kind : GoString) (changing : Bool) (key : w32)
    (m : StateData) (op : GoString) (b : Value) : Value :=
  numericConstrained (bindingValue kind changing key m) op b

def constraintRejected (kind : GoString) (changing : Bool) (key : w32)
    (m : StateData) (op : GoString) (b : Value) : Bool :=
  isEmpty (constraintValue kind changing key m op b) || impossibleComparison kind op b

def constraintState (active : Bool) (kind : GoString) (changing : Bool) (key : w32)
    (m : StateData) (op : GoString) (b : Value) : StateData :=
  if active && !constraintRejected kind changing key m op b then
    <[key := constraintValue kind changing key m op b]>m
  else m

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
