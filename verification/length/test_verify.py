import tempfile
from pathlib import Path
from types import SimpleNamespace
import unittest
from unittest.mock import patch

import verify


class VerificationGuardTests(unittest.TestCase):
    def audit(self, axiom="propext"):
        return "\n".join(f"'{name}' depends on axioms: [{axiom}]" for name in verify.THEOREMS)

    def test_standard_axioms_pass(self):
        verify.check_axioms(self.audit("propext, Classical.choice, Quot.sound"))

    def test_unfinished_and_native_proofs_fail(self):
        for axiom in ("sorryAx", "Lean.ofReduceBool", "customAssumption"):
            with self.subTest(axiom=axiom), self.assertRaises(RuntimeError):
                verify.check_axioms(self.audit(axiom))

    def test_missing_audit_entries_fail(self):
        with self.assertRaises(RuntimeError):
            verify.check_axioms(self.audit().split("\n", 1)[1])

    def test_pdg_audit_is_separate_and_fail_closed(self):
        profile = verify.pdg_profile()
        output = "\n".join(f"'{name}' depends on axioms: [propext]" for name in profile.theorems)
        verify.check_axioms(output, profile.theorems)
        for bad in (output.replace("propext", "sorryAx"), self.audit()):
            with self.assertRaises(RuntimeError):
                verify.check_axioms(bad, profile.theorems)

    def test_setup_error_does_not_reject_mutation(self):
        result = SimpleNamespace(returncode=1, stdout="error: unknown package", stderr="")
        with tempfile.TemporaryDirectory(dir=verify.PROJECT, prefix=".guard-") as directory:
            with patch.object(verify, "REPORT", Path(directory)), patch.object(verify.subprocess, "run", return_value=result):
                with self.assertRaises(RuntimeError):
                    verify.run(["lake", "build"], verify.PROJECT, "setup.log", expect_failure=True, proof_file="Gstyle/Proof/Queries.lean")

    def test_only_expected_proof_diagnostics_reject_mutation(self):
        proof = "Gstyle/Proof/Queries.lean"
        accepted = ["unsolved goals", "iapply: cannot apply Φ #false to Φ #true",
                    "wp_if_destruct: no `decide` or Boolean variable in the expression"]
        for message in accepted:
            output = f"error: {proof}:2:0: {message}\nerror: build failed\n"
            self.assertTrue(verify.proof_rejected(output, proof))
        rejected = [
            f"error: {proof}:2:0: unknown module prefix 'Gstyle.GeneratedProof'",
            f"error: {proof}:2:0: Unknown identifier 'missing'",
            f"error: {proof}:2:0: unexpected token",
            "error: Gstyle/Proof/Length.lean:2:0: unsolved goals",
            "error: build failed",
            f"error: {proof}:2:0: unsolved goals\nerror: failed to download dependency",
            f"error: {proof}:2:0: unsolved goals\nerror: {proof}:3:0: unknown module prefix 'Missing'",
        ]
        for output in rejected:
            with self.subTest(output=output):
                self.assertFalse(verify.proof_rejected(output, proof))

    def test_missing_module_in_proof_file_aborts_run(self):
        proof = "Gstyle/Proof/Queries.lean"
        result = SimpleNamespace(returncode=1, stdout=f"error: {proof}:2:0: unknown module prefix 'Gstyle.GeneratedProof'\n", stderr="")
        with tempfile.TemporaryDirectory(dir=verify.PROJECT, prefix=".guard-") as directory:
            with patch.object(verify.subprocess, "run", return_value=result):
                with self.assertRaises(RuntimeError):
                    verify.run(["lake", "build"], verify.PROJECT, "setup.log", expect_failure=True,
                               report=Path(directory), proof_file=proof)

    def test_failed_run_clears_success(self):
        with tempfile.TemporaryDirectory(dir=verify.PROJECT, prefix=".guard-") as directory:
            report = Path(directory)
            (report / "result.json").write_text("stale success")
            with patch.object(verify, "REPORT", report), patch.object(verify, "run", side_effect=RuntimeError("build failed")):
                with self.assertRaises(RuntimeError):
                    verify.main()
            self.assertFalse((report / "result.json").exists())


if __name__ == "__main__":
    unittest.main()
