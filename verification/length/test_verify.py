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

    def test_setup_error_does_not_reject_mutation(self):
        result = SimpleNamespace(returncode=1, stdout="error: unknown package", stderr="")
        with tempfile.TemporaryDirectory(dir=verify.PROJECT, prefix=".guard-") as directory:
            with patch.object(verify, "REPORT", Path(directory)), patch.object(verify.subprocess, "run", return_value=result):
                with self.assertRaises(RuntimeError):
                    verify.run(["lake", "build"], verify.PROJECT, "setup.log", expect_failure=True)

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
