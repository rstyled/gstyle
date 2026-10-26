from dataclasses import replace
import json
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import ci_cached
import verify


class SetupProfileTests(unittest.TestCase):
    def test_profile_registration_and_ci_coverage(self):
        profile = verify.setup_profile()
        self.assertIs(verify.PROFILE_FACTORIES[("--setup",)], verify.setup_profile)
        self.assertEqual(profile.report, verify.ROOT / "dist/verification/setup")
        makefile = (verify.ROOT / "Makefile").read_text()
        self.assertIn(".PHONY: verify-setup\nverify-setup:\n", makefile)
        self.assertIn("\tpython3 -B verification/lean/verify.py --setup\n", makefile)
        entries = ci_cached.load_entries(verify.PROJECT / "ci_profiles.json")
        selected = [entry for entry in entries if entry["argument"] == "--setup"]
        self.assertEqual(len(selected), 1)
        self.assertEqual(selected[0]["profile"], "setup")
        self.assertEqual(selected[0]["target"], "verify-setup")
        self.assertEqual(selected[0]["shard"], "0/1")

    def test_profile_binds_inherited_sources_and_setup_proofs(self):
        base, profile = verify.solver_profile(), verify.setup_profile()
        self.assertTrue(set(base.sources) <= set(profile.sources))
        for name in ("SetupSyntax", "Setup", "Initial"):
            source = verify.PROJECT / f"Gstyle/Proof/Worklist/{name}.lean"
            self.assertTrue(source.is_file())
            self.assertIn(source, profile.sources)
        worklist = verify.PROJECT / "Gstyle/Proof/Worklist"
        for pattern in ("Setup*.lean", "Initial.lean", "Constructor*.lean"):
            with self.subTest(pattern=pattern):
                self.assertLessEqual(set(worklist.glob(pattern)), set(profile.sources))
        for field in ("source", "package", "go_package", "generated", "bindings"):
            self.assertEqual(getattr(profile, field), getattr(base, field))
        modules = verify.baseline_modules(profile)
        self.assertIn("Gstyle.Proof.Worklist.Setup", modules)
        self.assertIn("Gstyle.Proof.Worklist.Initial", modules)
        self.assertNotIn("Gstyle", modules)

    def test_initial_and_syntax_audit_inventory_is_explicit_and_complete(self):
        profile = verify.setup_profile()
        prefix = "Perennial.github_com.rstyled.gstyle.domain.runtimeguard."
        initial = (verify.PROJECT / "Gstyle/Proof/Worklist/Initial.lean").read_text()
        names = set(re.findall(r"^theorem ([A-Za-z_][A-Za-z_0-9]*)", initial, re.MULTILINE))
        self.assertEqual(len(names), 19)
        self.assertTrue({prefix + name for name in names} <= profile.theorems)
        for name in ("solve_setup_stops_before_cursor", "setupContext_terminal", "wp_solve_setup_invariant"):
            self.assertIn(prefix + name, profile.theorems)

    def test_audit_matches_registered_theorems_and_rejects_unfinished_proofs(self):
        profile = verify.setup_profile()
        audit = (verify.PROJECT / profile.audit).read_text().splitlines()
        names = [line.removeprefix("#print axioms ") for line in audit if line.startswith("#print axioms ")]
        self.assertEqual(len(names), len(set(names)))
        self.assertEqual(set(names), profile.theorems)
        output = "\n".join(f"'{name}' depends on axioms: [propext]" for name in names)
        verify.check_axioms(output, profile.theorems)
        for bad in (output.replace("propext", "sorryAx"), output.split("\n", 1)[1]):
            with self.assertRaises(RuntimeError):
                verify.check_axioms(bad, profile.theorems)

    def test_setup_controls_target_the_execution_proof(self):
        profile = verify.setup_profile()
        expected = {"setup-wrong-inputs-length", "setup-wrong-flags-length", "setup-nil-entry-map",
                    "setup-wrong-entry-slot", "setup-missing-entry-enqueue", "setup-wrong-view"}
        self.assertTrue(expected <= set(profile.mutations))
        self.assertEqual(set(profile.mutations), set(profile.mutation_modules))
        for name in expected:
            self.assertEqual(profile.mutation_modules[name], "Gstyle/Proof/Worklist/Setup.lean")
        constructor_targets = {
            "constructor-wrong-symbol-kind": "ConstructorSymbols",
            "constructor-inverts-global-instability": "ConstructorSymbols",
            "constructor-wrong-operand-row": "ConstructorOperands",
            "constructor-wrong-operand-value": "ConstructorOperands",
        }
        for name, module in constructor_targets.items():
            self.assertEqual(profile.mutation_modules[name], f"Gstyle/Proof/Worklist/{module}.lean")
            self.assertEqual(profile.mutations[name][2], "domain/runtimeguard/graph.go")
        self.assertTrue(all((verify.ROOT / mutation[2] if len(mutation) == 3 else profile.source).suffix == ".go"
                            for mutation in profile.mutations.values()))

    def test_each_control_builds_its_named_proof(self):
        profile = verify.setup_profile()
        for name, mutation in profile.mutations.items():
            with self.subTest(mutation=name), tempfile.TemporaryDirectory() as directory:
                selected = replace(profile, report=Path(directory), mutations={name: mutation})
                with patch.object(verify, "seed_build"), patch.object(verify, "generate") as generate, \
                     patch.object(verify, "run") as run, patch.object(verify, "print", create=True):
                    verify.check_mutations(selected, Path("goose"), Path("proofgen"))
                generate.assert_called_once()
                proof = profile.mutation_modules[name]
                self.assertEqual(run.call_args.args[0], ["lake", "build", proof.removesuffix(".lean").replace("/", ".")])
                self.assertEqual(run.call_args.kwargs["proof_file"], proof)
                self.assertTrue(run.call_args.kwargs["expect_failure"])

    def test_controls_reject_only_the_named_proof_failure(self):
        profile = verify.setup_profile()
        for proof in set(profile.mutation_modules.values()):
            failure = f"error: {proof}:1:0: unsolved goals\n"
            self.assertTrue(verify.proof_rejected(failure + "error: build failed\n", proof))
            self.assertFalse(verify.proof_rejected(failure, "Gstyle/Proof/Worklist/SetupSyntax.lean"))
            for error in ("unexpected token", "Unknown identifier 'missing'", "Type mismatch",
                          "failed to synthesize instance of type class", "unknown module prefix"):
                with self.subTest(proof=proof, error=error):
                    diagnostic = f"error: {proof}:2:0: {error}\n"
                    self.assertFalse(verify.proof_rejected(diagnostic, proof))
                    self.assertFalse(verify.proof_rejected(failure + diagnostic, proof))
            self.assertFalse(verify.proof_rejected(failure + "error: setup failed\n", proof))

    @unittest.skipUnless(shutil.which("go"), "requires Go to compile production controls")
    def test_production_controls_compile(self):
        profile = verify.setup_profile()
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for name, mutation in profile.mutations.items():
                with self.subTest(mutation=name):
                    source = verify.ROOT / mutation[2] if len(mutation) == 3 else profile.source
                    before, after = mutation[:2]
                    original = source.read_text()
                    self.assertEqual(original.count(before), 1)
                    changed = root / source.name
                    changed.write_text(original.replace(before, after))
                    overlay = root / "overlay.json"
                    overlay.write_text(json.dumps({"Replace": {str(source): str(changed)}}))
                    result = subprocess.run(["go", "test", "-run", "^$", "-vet=off", "-overlay", str(overlay), profile.go_package],
                                            cwd=verify.ROOT, text=True, capture_output=True, timeout=180)
                    self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
