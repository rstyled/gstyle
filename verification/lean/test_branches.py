from pathlib import Path
import sys
import tempfile
import unittest
from unittest import mock

import verify_branches


class StructuralGuardTests(unittest.TestCase):
    def test_missing_source_invalidates_previous_success(self):
        with tempfile.TemporaryDirectory() as directory:
            report = Path(directory)
            success = report / "result.json"
            success.write_text('{"verified": "previous source"}\n')
            args = ["verify_branches.py", "--source", str(report / "missing.go")]
            with mock.patch.object(verify_branches, "REPORT", report), mock.patch.object(sys, "argv", args):
                with self.assertRaises(FileNotFoundError):
                    verify_branches.main()
            self.assertFalse(success.exists(), "failed source read leaves stale success")

    def test_only_named_logical_failures_reject_a_control(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "Structural.lean"
            path.write_text("theorem covered_sound : False := by\n  omega\n")
            output = f"{path}:2:2: error: omega could not prove the goal:\n"
            self.assertTrue(verify_branches.rejected(output, path, {"covered_sound"}))
            self.assertFalse(verify_branches.rejected(output, path, {"inverse_holds"}))
            for error in ("unexpected token", "unknown module prefix", "Type mismatch"):
                bad = f"{path}:2:2: error: {error}\n"
                self.assertFalse(verify_branches.rejected(bad, path, {"covered_sound"}))
                self.assertFalse(verify_branches.rejected(output + bad, path, {"covered_sound"}))
            self.assertFalse(verify_branches.rejected(output + "error: setup failed\n", path, {"covered_sound"}))
