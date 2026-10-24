from pathlib import Path
import tempfile
import unittest

import verify_programs


class ProgramCertificateGuardTests(unittest.TestCase):
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
