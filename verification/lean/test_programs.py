from pathlib import Path
import shutil
import tempfile
import unittest

import verify_programs


class ProgramCertificateGuardTests(unittest.TestCase):
    @unittest.skipUnless(shutil.which("lake"), "requires the pinned Lean toolchain")
    def test_binding_rejects_a_record_without_certificate_proofs(self):
        # A standalone type fixture keeps this guard independent of project builds.
        fixture = """namespace Perennial.GstylePrograms
inductive Expression where | input
inductive Program where | result : Expression → Program
structure Reported where
  lower : Int
  upper : Int
  hasLower : Bool
  hasUpper : Bool
structure UncheckedReport where
  name : String
  program : Program
  report : Reported
  source : String
structure Certificate extends UncheckedReport where
  sound : True
end Perennial.GstylePrograms
open Perennial.GstylePrograms
namespace GeneratedPrograms
def unchecked : UncheckedReport := ⟨"Case0", .result .input, ⟨0, 0, false, false⟩, "source"⟩
def cCase0 : Certificate := ⟨unchecked, True.intro⟩
end GeneratedPrograms
"""
        data = {"cases": [{"name": "Case0", "source": "source",
            "program": {"value": {"input": True}},
            "range": {"Lower": 0, "Upper": 0, "HasLower": False, "HasUpper": False}}]}
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory)
            certificate = path / "Certificate.lean"
            certificate.write_text(fixture)
            verify_programs.check_bindings(path, data)
            certificate.write_text(fixture.replace(
                "def cCase0 : Certificate := ⟨unchecked, True.intro⟩",
                "def cCase0 : UncheckedReport := unchecked"))
            with self.assertRaisesRegex(RuntimeError, "command failed"):
                verify_programs.check_bindings(path, data)
            rejection = (path / "binding.log").read_text()
            self.assertIn("Type mismatch", rejection)
            self.assertIn("UncheckedReport", rejection)
            self.assertIn("but is expected to have type\n  Certificate", rejection)

    def test_binding_values_are_literals_not_lean_expressions(self):
        for value in (True, 2**63, -(2**63)-1, "0 + 1", 1.0):
            with self.assertRaises(ValueError):
                verify_programs.integer_literal(value)
        for value in (0, 1, "true"):
            with self.assertRaises(ValueError):
                verify_programs.boolean_literal(value)
        self.assertEqual(verify_programs.integer_literal(-(2**63)), "-9223372036854775808")
        self.assertEqual(verify_programs.boolean_literal(False), "false")

    def test_actual_certificate_is_bound_and_audited(self):
        from unittest import mock
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory)
            (path / "Certificate.lean").write_text("-- unchanged certificate definitions\n")
            data = {"cases": [{"name": "Case0", "source": "captured source\n",
                "program": {"value": {"input": True, "value": 0}},
                "range": {"Lower": 1, "Upper": 1, "HasLower": True, "HasUpper": True}}]}
            audit = ("'CapturedSubjects.bound0' does not depend on any axioms\n"
                     "'CapturedSubjects.subject0' does not depend on any axioms\n"
                     "'GeneratedPrograms.cCase0' does not depend on any axioms\n")
            with mock.patch.object(verify_programs, "run", return_value=audit):
                verify_programs.check_bindings(path, data)
            bridge = (path / "BoundCertificate.lean").read_text()
            for expected in ("def subject0 : Perennial.GstylePrograms.Certificate := GeneratedPrograms.«cCase0»",
                             "CapturedSubjects.subject0.program = (.result .input)",
                             "CapturedSubjects.subject0.report = (⟨(1), (1), true, true⟩",
                             '#print axioms CapturedSubjects.subject0',
                             '#print axioms GeneratedPrograms.«cCase0»',
                             'CapturedSubjects.subject0.source = "captured source\\n"'):
                self.assertIn(expected, bridge)
            bad = audit.replace("'GeneratedPrograms.cCase0' does not depend on any axioms",
                                "'GeneratedPrograms.cCase0' depends on axioms: [unapproved]")
            with mock.patch.object(verify_programs, "run", return_value=bad):
                with self.assertRaisesRegex(RuntimeError, "unexpected axioms"):
                    verify_programs.check_bindings(path, data)

    def test_only_expected_logical_failure_rejects_control(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "Certificate.lean"
            path.write_text("theorem soundCase0 : False := by\n  skip\n")
            logical = f"{path}:1:0: error: unsolved goals\n"
            self.assertTrue(verify_programs.proof_rejected(logical, path, {"soundCase0"}))
            self.assertFalse(verify_programs.proof_rejected(logical, path, {"coverage0"}))
            for bad in ("failed to synthesize Decidable", "unexpected token", "unknown module prefix"):
                output = f"{path}:1:0: error: {bad}\n"
                self.assertFalse(verify_programs.proof_rejected(output, path, {"soundCase0"}))
                self.assertFalse(verify_programs.proof_rejected(logical + output, path, {"soundCase0"}))
            self.assertFalse(verify_programs.proof_rejected(logical + "error: setup failed\n", path, {"soundCase0"}))

    def test_artifact_hash_and_source_are_both_checked(self):
        import hashlib
        import json
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory)
            source = b"package generated\nfunc observe(value int64) {}\n"
            data = {"source_sha256": hashlib.sha256(source).hexdigest(), "cases": []}
            (path / "input.go.txt").write_bytes(source)
            (path / "observations.json").write_text(json.dumps(data))
            (path / "Certificate.lean").write_text("")
            verify_programs.validate_artifacts(path)
            (path / "input.go.txt").write_bytes(source + b"changed")
            with self.assertRaisesRegex(RuntimeError, "hash"):
                verify_programs.validate_artifacts(path)
            data["source_sha256"] = hashlib.sha256(source + b"changed").hexdigest()
            (path / "observations.json").write_text(json.dumps(data))
            with self.assertRaisesRegex(RuntimeError, "analyzed file"):
                verify_programs.validate_artifacts(path)
