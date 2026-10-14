import importlib.util
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location("proof_gate", Path(__file__).with_name("unchecked_output.py"))
GATE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(GATE)


class ProofGateTests(unittest.TestCase):
    def setUp(self):
        silent = patch.object(GATE, "print", create=True)
        silent.start()
        self.addCleanup(silent.stop)

    def test_failed_rerun_removes_stale_success(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory)
            report = output / "report.json"
            report.write_text('{"decision_core_verified":true}')
            with patch.object(GATE, "OUTPUT", output), patch.object(GATE.subprocess, "run", side_effect=subprocess.CalledProcessError(1, ["docker"])):
                with self.assertRaises(subprocess.CalledProcessError):
                    GATE.verify()
            self.assertFalse(report.exists())

    def test_tool_failures_do_not_count_as_rejected_mutants(self):
        for code, log in [(1, "parse error"), (0, "Postcondition might not hold")]:
            with self.subTest(code=code, log=log), tempfile.TemporaryDirectory() as directory:
                with patch.object(GATE, "OUTPUT", Path(directory)), patch.object(GATE.subprocess, "run"), patch.object(GATE, "run_verifier", side_effect=[(0, "Gobra found 0 errors."), (code, log)]):
                    with self.assertRaisesRegex(RuntimeError, "mutant must fail its postcondition"):
                        GATE.verify()
                self.assertFalse((Path(directory) / "report.json").exists())

    def test_failed_production_cannot_pass_on_log_text(self):
        with tempfile.TemporaryDirectory() as directory:
            with patch.object(GATE, "OUTPUT", Path(directory)), patch.object(GATE.subprocess, "run"), patch.object(GATE, "run_verifier", return_value=(1, "Gobra found 0 errors.")):
                with self.assertRaisesRegex(RuntimeError, "production proof failed"):
                    GATE.verify()

    def test_timeout_removes_owned_container(self):
        expired = subprocess.TimeoutExpired(["docker", "run"], 60)
        with patch.object(GATE.subprocess, "run", side_effect=[expired, subprocess.CompletedProcess([], 0)]) as run:
            with self.assertRaises(subprocess.TimeoutExpired):
                GATE.run_verifier(GATE.SOURCE, "timeout-test")
            cleanup = run.call_args_list[1].args[0]
            self.assertEqual(cleanup[:3], ["docker", "rm", "--force"])
            self.assertTrue(cleanup[3].endswith("-timeout-test"))


if __name__ == "__main__":
    unittest.main()
