module
public import Gstyle.Proof.Transfer.FactSnapshot
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

def decimalDigits : GoString → Nat → Option Nat
  | [], acc => some acc
  | digit :: rest, acc =>
    if 48 ≤ digit.toNat ∧ digit.toNat ≤ 57 then
      decimalDigits rest (acc * 10 + (digit.toNat - 48)) else none

def decimalMagnitude : GoString → Option Nat
  | [] => none
  | bytes@(_ :: _) => decimalDigits bytes 0

def signedDecimal (negative : Bool) (bytes : GoString) : Option Int :=
  (decimalMagnitude bytes).bind fun magnitude =>
    let value : Int := if negative then -(magnitude : Int) else magnitude
    if -(2^63) ≤ value ∧ value < 2^63 then some value else none

def decimalInt64 : GoString → Option Int
  | [] => none
  | byte :: rest =>
    if byte = W8 45 then signedDecimal true rest
    else if byte = W8 43 then signedDecimal false rest
    else signedDecimal false (byte :: rest)

theorem signedDecimal_range (negative : Bool) (bytes : GoString) (value : Int)
    (h : signedDecimal negative bytes = some value) : -(2^63) ≤ value ∧ value < 2^63 := by
  unfold signedDecimal at h
  cases hm : decimalMagnitude bytes with
  | none => simp [hm] at h
  | some magnitude =>
    simp only [hm, Option.bind_some] at h
    by_cases bounds : -(2^63) ≤ (if negative then -(magnitude : Int) else magnitude) ∧
        (if negative then -(magnitude : Int) else magnitude) < 2^63
    · rw [ite_eq_left bounds] at h
      cases h
      exact bounds
    · rw [ite_eq_right bounds] at h
      cases h

theorem decimalInt64_range (bytes : GoString) (value : Int)
    (h : decimalInt64 bytes = some value) : -(2^63) ≤ value ∧ value < 2^63 := by
  cases bytes with
  | nil => cases h
  | cons byte rest =>
    change (if byte = W8 45 then signedDecimal true rest
      else if byte = W8 43 then signedDecimal false rest
      else signedDecimal false (byte :: rest)) = some value at h
    by_cases minus : byte = W8 45
    · rw [ite_eq_left minus] at h
      exact signedDecimal_range true rest value h
    · rw [ite_eq_right minus] at h
      by_cases plus : byte = W8 43
      · rw [ite_eq_left plus] at h
        exact signedDecimal_range false rest value h
      · rw [ite_eq_right plus] at h
        exact signedDecimal_range false (byte :: rest) value h

theorem decimalInt64_zero : decimalInt64 (go!"0") = some 0 := by decide

theorem decimalInt64_negative_zero : decimalInt64 (go!"-0") = some 0 := by decide

theorem decimalInt64_positive : decimalInt64 (go!"+17") = some 17 := by decide

theorem decimalInt64_negative : decimalInt64 (go!"-17") = some (-17) := by decide

theorem decimalInt64_leading_zero : decimalInt64 (go!"00017") = some 17 := by decide

theorem decimalInt64_minimum : decimalInt64 (go!"-9223372036854775808") = some (-9223372036854775808) := by decide

theorem decimalInt64_maximum : decimalInt64 (go!"9223372036854775807") = some 9223372036854775807 := by decide

theorem decimalInt64_positive_overflow : decimalInt64 (go!"9223372036854775808") = none := by decide

theorem decimalInt64_negative_overflow : decimalInt64 (go!"-9223372036854775809") = none := by decide

theorem decimalInt64_empty : decimalInt64 (go!"") = none := by decide

theorem decimalInt64_minus_only : decimalInt64 (go!"-") = none := by decide

theorem decimalInt64_plus_only : decimalInt64 (go!"+") = none := by decide

theorem decimalInt64_space : decimalInt64 (go!" 1") = none := by decide

theorem decimalInt64_underscore : decimalInt64 (go!"1_0") = none := by decide

theorem decimalInt64_hex : decimalInt64 (go!"0x10") = none := by decide

theorem decimalInt64_unicode : decimalInt64 (go!"１２") = none := by decide

def constantRecord [FfiSyntax] (nonNil nilFact : GoString) (integer extent : Option Int) : Value :=
  let initial : Value := { (zero_val Value) with NonNil' := decide (nonNil = go!"true"), Nil' := decide (nilFact = go!"true") }
  let numeric := match integer with
    | none => initial
    | some value => { initial with Lower' := W64 value, Upper' := W64 value, HasLower' := true, HasUpper' := true, Integer' := true }
  match extent with
  | none => numeric
  | some value => { numeric with Extent' := W64 value, HasExtent' := true }

def snapshotConstant [FfiSyntax] (s : QuerySnapshot) (ref : w32) : Value :=
  constantRecord (snapshotFact s ref (go!"non-nil")) (snapshotFact s ref (go!"nil"))
    (decimalInt64 (snapshotFact s ref (go!"integer"))) (decimalInt64 (snapshotFact s ref (go!"literal-extent")))

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
