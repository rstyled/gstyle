from dataclasses import replace
import os
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

    def test_output_audit_is_separate_and_fail_closed(self):
        profile = verify.output_profile()
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
        accepted = ["unsolved goals", "Tactic `rewrite` failed: Did not find an occurrence of the pattern", "iexists: cannot turn Φ #true into an existential quantifier", "iapply: cannot apply Φ #false to Φ #true",
                    "Tactic `rfl` failed: The left-hand side", "wp_apply: cannot apply a slice contract", "wp_apply: the goal forPostcondition is not a WP",
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

    def test_each_mutation_builds_only_its_profile_proof(self):
        for factory, module in ((verify.length_profile, "Gstyle.Proof.Length"),
                                (verify.pdg_profile, "Gstyle.Proof.Queries"),
                                (verify.output_profile, "Gstyle.Proof.Output.Classify"),
                                (verify.interval_profile, "Gstyle.Proof.Interval.Constrain"),
                                (verify.runtime_value_profile, "Gstyle.Proof.RuntimeValue"),
                                (verify.state_map_profile, "Gstyle.Proof.StateMap.Join"),
                                (verify.numeric_transfer_profile, "Gstyle.Proof.Transfer.Numeric"),
                                (verify.signed_step_profile, "Gstyle.Proof.Transfer.Step"),
                                (verify.assignment_profile, "Gstyle.Proof.Transfer.Assignment"),
                                (verify.branch_profile, "Gstyle.Proof.Transfer.Branch"),
                                (verify.pdg_text_profile, "Gstyle.Proof.Pdg.Text"),
                                (verify.operator_profile, "Gstyle.Proof.Transfer.Operator")):
            profile = factory()
            mutation = next(iter(profile.mutations.items()))
            with self.subTest(package=profile.package), tempfile.TemporaryDirectory(dir=verify.PROJECT, prefix=".guard-") as directory:
                profile = replace(profile, report=Path(directory), mutations=dict([mutation]))
                with patch.object(verify, "seed_build") as seed, patch.object(verify, "generate"), patch.object(verify, "run") as run, patch.object(verify, "print", create=True):
                    verify.check_mutations(profile, Path("goose"), Path("proofgen"))
                seed.assert_called_once()
                run.assert_called_once()
                self.assertEqual(run.call_args.args[0], ["lake", "build", module])
                self.assertTrue(run.call_args.kwargs["expect_failure"])
                self.assertEqual(run.call_args.kwargs["proof_file"], profile.mutation_modules.get(mutation[0], profile.proof_file))

    def test_split_mutations_have_explicit_existing_contracts(self):
        for factory in (verify.interval_profile, verify.output_profile):
            profile = factory()
            self.assertEqual(set(profile.mutation_modules), set(profile.mutations))
            for name, proof in profile.mutation_modules.items():
                with self.subTest(mutation=name):
                    self.assertTrue((verify.PROJECT / proof).is_file())
                    self.assertNotEqual(proof, profile.proof_file)

    def test_shards_cover_every_mutation_once(self):
        for factory, count in ((verify.interval_profile, 4), (verify.output_profile, 3),
                               (verify.runtime_value_profile, 2),
                               (verify.state_map_profile, 2),
                               (verify.numeric_transfer_profile, 3),
                               (verify.signed_step_profile, 2),
                               (verify.assignment_profile, 2),
                               (verify.branch_profile, 2),
                               (verify.pdg_text_profile, 1),
                               (verify.operator_profile, 1)):
            mutations = factory().mutations
            selected = [name for index in range(count)
                        for name in verify.shard_mutations(mutations, (index, count))]
            self.assertCountEqual(selected, mutations)
            self.assertEqual(len(set(selected)), len(selected))

    def test_invalid_shards_fail_closed(self):
        for raw in ("", "0", "-1/2", "0/0", "2/2", "0/2/3", "x/2"):
            with self.subTest(value=raw), patch.dict(os.environ, {"GSTYLE_MUTATION_SHARD": raw}):
                with self.assertRaises(RuntimeError):
                    verify.mutation_shard()
        with patch.dict(os.environ, {"GSTYLE_MUTATION_SHARD": "1/4"}):
            self.assertEqual(verify.mutation_shard(), (1, 4))
        with self.assertRaises(RuntimeError):
            verify.shard_mutations({"one": ()}, (0, 2))

    def test_parallel_mutation_failure_aborts_verification(self):
        profile = verify.length_profile()
        with tempfile.TemporaryDirectory(dir=verify.PROJECT, prefix=".guard-") as directory:
            profile = replace(profile, report=Path(directory))
            with patch.object(verify, "check_mutation", side_effect=RuntimeError("proof accepted mutant")):
                with self.assertRaisesRegex(RuntimeError, "proof accepted mutant"):
                    verify.check_mutations(profile, Path("goose"), Path("proofgen"))

    def test_parallel_mutation_results_keep_profile_order(self):
        profile = verify.length_profile()
        with tempfile.TemporaryDirectory(dir=verify.PROJECT, prefix=".guard-") as directory:
            profile = replace(profile, report=Path(directory))
            with patch.object(verify, "check_mutation", side_effect=lambda profile, goose, proofgen, item: item[0]):
                result = verify.check_mutations(profile, Path("goose"), Path("proofgen"))
            self.assertEqual(result, list(profile.mutations))

    def test_seeded_build_is_private_and_preserves_trace(self):
        with tempfile.TemporaryDirectory(dir=verify.PROJECT, prefix=".guard-") as directory:
            root = Path(directory)
            baseline = root / "baseline"
            artifact = baseline / ".lake/build/lib/lean/Example.olean"
            artifact.parent.mkdir(parents=True)
            artifact.write_bytes(b"baseline")
            trace = artifact.with_suffix(".trace")
            trace.write_bytes(b"input hashes")
            lean = root / "mutant"
            with patch.object(verify, "PROJECT", baseline):
                verify.seed_build(lean)
            copied = lean / artifact.relative_to(baseline)
            self.assertEqual(copied.read_bytes(), b"baseline")
            self.assertEqual(copied.with_suffix(".trace").read_bytes(), b"input hashes")
            self.assertEqual(copied.stat().st_mtime_ns, artifact.stat().st_mtime_ns)
            copied.write_bytes(b"mutant")
            self.assertEqual(artifact.read_bytes(), b"baseline")

    def test_runtime_value_generates_only_actual_code(self):
        profile = verify.runtime_value_profile()
        with patch.object(verify, "run") as run:
            verify.generate(profile, Path("goose"), Path("proofgen"), verify.ROOT, Path("out"), "source")
        run.assert_called_once()
        self.assertEqual(run.call_args.args[0][0], "goose")
        self.assertEqual(profile.generated, ("Code",))
        self.assertEqual(run.call_args.args[0][-4:], ["./domain/runtimeguard", "./domain/runtimeguard/interval", "./domain/pdg", "./domain/pdg/query"])

    def test_runtime_value_rejects_stale_imported_code(self):
        profile = verify.runtime_value_profile()
        with tempfile.TemporaryDirectory(dir=verify.PROJECT, prefix=".guard-") as directory:
            root = Path(directory)
            generated = root / "regenerated"
            for package in (profile.package, *[package for package, _ in profile.bindings]):
                for base in (root / "Gstyle", generated):
                    target = base / "Code" / package
                    target.parent.mkdir(parents=True, exist_ok=True)
                    target.write_text("same")
            with patch.object(verify, "PROJECT", root):
                verify.check_generated(profile, generated)
                for package, _ in profile.bindings:
                    target = generated / "Code" / package
                    target.write_text("stale")
                    with self.assertRaisesRegex(RuntimeError, "stale generated source"):
                        verify.check_generated(profile, generated)
                    target.write_text("same")
                verify.check_generated(profile, generated)

    def test_mutation_requires_a_bound_source(self):
        profile = replace(verify.runtime_value_profile(), mutations={"unbound": ("before", "after", "outside.go")})
        with tempfile.TemporaryDirectory(dir=verify.PROJECT, prefix=".guard-") as directory:
            with self.assertRaisesRegex(RuntimeError, "source is not bound"):
                verify.check_mutations(replace(profile, report=Path(directory)), Path("goose"), Path("proofgen"))

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
