from dataclasses import replace
import os
import json
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

    def test_spatial_frame_failure_is_named_and_excludes_setup_errors(self):
        proof = "Gstyle/Proof/Worklist/Inputs.lean"
        failure = f"error: {proof}:58:8: iframe: cannot frame ownsBranch value next\n"
        self.assertTrue(verify.proof_rejected(failure + "error: build failed\n", proof))
        self.assertFalse(verify.proof_rejected(failure, "Gstyle/Proof/Worklist/Traversal.lean"))
        for unrelated in ("Type mismatch", "failed to synthesize instance of type class", "unknown module prefix"):
            with self.subTest(unrelated=unrelated):
                self.assertFalse(verify.proof_rejected(failure + f"error: {proof}:59:0: {unrelated}\n", proof))

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
                                (verify.runtime_value_profile, "Gstyle.Proof.RuntimeValue.MergeBounds"),
                                (verify.state_map_profile, "Gstyle.Proof.StateMap.Join"),
                                (verify.numeric_transfer_profile, "Gstyle.Proof.Transfer.NumericPrimitives"),
                                (verify.signed_step_profile, "Gstyle.Proof.Transfer.Step"),
                                (verify.assignment_profile, "Gstyle.Proof.Transfer.Assignment"),
                                (verify.branch_profile, "Gstyle.Proof.Transfer.Branch"),
                                (verify.pdg_text_profile, "Gstyle.Proof.Pdg.Text"),
                                (verify.operator_profile, "Gstyle.Proof.Transfer.Operator"),
                                (verify.pdg_list_profile, "Gstyle.Proof.Pdg.List"),
                                (verify.pdg_attributes_profile, "Gstyle.Proof.Pdg.Attributes"),
                                (verify.runtime_facts_profile, "Gstyle.Proof.Transfer.Extension")):
            profile = factory()
            mutation = next(iter(profile.mutations.items()))
            with self.subTest(package=profile.package), tempfile.TemporaryDirectory(dir=verify.PROJECT, prefix=".guard-") as directory:
                profile = replace(profile, report=Path(directory), mutations=dict([mutation]))
                with patch.object(verify, "seed_build") as seed, patch.object(verify, "generate") as generate, patch.object(verify, "run") as run, patch.object(verify, "print", create=True):
                    verify.check_mutations(profile, Path("goose"), Path("proofgen"))
                seed.assert_called_once()
                generate.assert_called_once()
                run.assert_called_once()
                self.assertEqual(run.call_args.args[0], ["lake", "build", module])
                self.assertTrue(run.call_args.kwargs["expect_failure"])
                self.assertEqual(run.call_args.kwargs["proof_file"], profile.mutation_modules.get(mutation[0], profile.proof_file))

    def test_source_drift_cannot_publish_success(self):
        manifest = json.loads((verify.PROJECT / "lake-manifest.json").read_text())
        revisions = {package["name"]: package["rev"] for package in manifest["packages"]}
        with tempfile.TemporaryDirectory(dir=verify.PROJECT, prefix=".guard-") as directory:
            root = Path(directory)
            source = root / "input.go"
            source.write_text("before")
            profile = replace(verify.length_profile(), source=source, sources=(source,), report=root / "report")

            def run(command, cwd, log, **kwargs):
                if command[:2] == ["git", "rev-parse"]:
                    return revisions[cwd.name]
                if log == "axioms.log":
                    return self.audit()
                return ""

            def mutate(*args):
                source.write_text("after")
                return []

            with patch.dict(os.environ, {"GSTYLE_MUTATION_CACHE": ""}), patch.object(verify, "run", side_effect=run), patch.object(verify, "generate"), patch.object(verify, "check_generated"), patch.object(verify, "check_mutations", side_effect=mutate):
                with self.assertRaisesRegex(RuntimeError, "bound source changed"):
                    verify.main(profile)
            self.assertFalse((profile.report / "result.json").exists())

    def test_model_mutation_is_private_and_keeps_generated_code(self):
        profile = verify.runtime_facts_profile()
        name = "model-cache-result"
        before, after, relative = profile.mutations[name]
        source = verify.ROOT / relative
        baseline = source.read_bytes()
        code = verify.PROJECT / "Gstyle/Code" / profile.package
        canonical = code.read_bytes()
        with tempfile.TemporaryDirectory(dir=verify.PROJECT, prefix=".guard-") as directory:
            profile = replace(profile, report=Path(directory), mutations={name: profile.mutations[name]})

            def inspect(command, cwd, log, **kwargs):
                self.assertEqual(command, ["lake", "build", "Gstyle.Proof.Transfer.SymbolModel"])
                self.assertEqual((cwd / source.relative_to(verify.PROJECT)).read_text(), baseline.decode().replace(before, after))
                self.assertEqual((cwd.parent / "go" / relative).read_bytes(), baseline)
                self.assertEqual((cwd / "Gstyle/Code" / profile.package).read_bytes(), canonical)
                self.assertTrue(kwargs["expect_failure"])
                self.assertEqual(kwargs["proof_file"], "Gstyle/Proof/Transfer/SymbolModel.lean")

            with patch.object(verify, "seed_build") as seed, patch.object(verify, "generate") as generate, patch.object(verify, "run", side_effect=inspect), patch.object(verify, "print", create=True):
                verify.check_mutations(profile, Path("goose"), Path("proofgen"))
            seed.assert_called_once()
            generate.assert_not_called()
        self.assertEqual(source.read_bytes(), baseline)
        self.assertEqual(code.read_bytes(), canonical)

    def test_go_mutation_regenerates_from_private_changed_source(self):
        profile = verify.runtime_facts_profile()
        name, mutation = next(iter(profile.mutations.items()))
        before, after = mutation
        source = profile.source
        baseline = source.read_text()
        with tempfile.TemporaryDirectory(dir=verify.PROJECT, prefix=".guard-") as directory:
            profile = replace(profile, report=Path(directory), mutations={name: mutation})

            def inspect(selected, goose, proofgen, source_root, output, label, **kwargs):
                self.assertEqual((source_root / source.relative_to(verify.ROOT)).read_text(), baseline.replace(before, after))
                self.assertNotEqual(source_root.resolve(), verify.ROOT.resolve())
                self.assertEqual(source.read_text(), baseline)
                self.assertEqual(label, name)
                environment = kwargs["environment"]
                self.assertEqual(environment["LEAN_NUM_THREADS"], os.environ.get("LEAN_NUM_THREADS", "1"))
                self.assertEqual(environment["GOMAXPROCS"], os.environ.get("GOMAXPROCS", "1"))
                self.assertIsNot(environment, os.environ)

            with patch.object(verify, "seed_build"), patch.object(verify, "generate", side_effect=inspect) as generate, patch.object(verify, "run"), patch.object(verify, "print", create=True):
                verify.check_mutations(profile, Path("goose"), Path("proofgen"))
            generate.assert_called_once()
        self.assertEqual(source.read_text(), baseline)

    def test_mutations_cannot_target_generated_or_external_sources(self):
        with tempfile.TemporaryDirectory() as directory:
            work = Path(directory)
            for source in (verify.PROJECT / "Gstyle/Code" / verify.runtime_facts_profile().package,
                           verify.PROJECT / "README.md", work / "outside.go", work / "outside.lean"):
                with self.subTest(source=source), self.assertRaises(RuntimeError):
                    verify.mutation_destination(source, work)

    def test_model_symlinks_cannot_escape_the_proof_tree(self):
        with tempfile.TemporaryDirectory() as outside, tempfile.TemporaryDirectory(dir=verify.PROJECT / "Gstyle/Proof", prefix=".guard-") as inside:
            target = Path(outside) / "model.lean"
            target.write_text("external")
            link = Path(inside) / "model.lean"
            link.symlink_to(target)
            with self.assertRaisesRegex(RuntimeError, "escapes"):
                verify.mutation_destination(link, Path(outside) / "scratch")
            self.assertEqual(target.read_text(), "external")
            link.unlink()
            link.symlink_to(verify.runtime_facts_profile().source)
            with self.assertRaisesRegex(RuntimeError, "changes language"):
                verify.mutation_destination(link, Path(outside) / "scratch")

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
                               (verify.operator_profile, 1),
                               (verify.pdg_list_profile, 1),
                               (verify.pdg_attributes_profile, 2),
                               (verify.runtime_facts_profile, 2),
                               (verify.value_evaluation_profile, 3),
                               (verify.comparison_profile, 4),
                               (verify.one_sided_profile, 1),
                               (verify.shared_flow_profile, 1),
                               (verify.assume_profile, 1),
                               (verify.worklist_profile, 2), (verify.scalar_flow_profile, 1)):
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
        for source in ("outside.go", "verification/lean/Gstyle/Proof/Transfer/SymbolModel.lean"):
            profile = replace(verify.runtime_value_profile(), mutations={"unbound": ("before", "after", source)})
            with tempfile.TemporaryDirectory(dir=verify.PROJECT, prefix=".guard-") as directory:
                with self.assertRaisesRegex(RuntimeError, "source is not bound"):
                    verify.check_mutations(replace(profile, report=Path(directory)), Path("goose"), Path("proofgen"))

    def test_numeric_controls_follow_moved_contracts(self):
        profile = verify.numeric_transfer_profile()
        self.assertEqual(set(profile.mutation_modules), set(profile.mutations))
        for proof in profile.mutation_modules.values():
            self.assertIn(verify.PROJECT / proof, profile.sources)
        self.assertTrue(profile.mutation_modules["exact-ignores-lower"].endswith("NumericPrimitives.lean"))
        self.assertTrue(profile.mutation_modules["tracked-inverts-flag"].endswith("Tracking.lean"))
        self.assertTrue(profile.mutation_modules["constrain-wrong-bound"].endswith("NumericConstrainQueries.lean"))

    def test_comparison_contracts_are_bound_and_audited(self):
        profile = verify.comparison_profile()
        for name in ("ComparisonQueries", "ComparisonNil", "NumericModel", "Tracking", "ComparisonTransferQueries", "NumericPrimitives", "NumericConstrainQueries"):
            self.assertIn(verify.PROJECT / f"Gstyle/Proof/Transfer/{name}.lean", profile.sources)
        for name in ("wp_comparison_queries_true", "wp_comparison_queries_false"):
            self.assertIn("Perennial.github_com.rstyled.gstyle.domain.runtimeguard." + name, profile.theorems)
        self.assertEqual(profile.mutation_modules["constraint-drops-write"],
                         "Gstyle/Proof/Transfer/NumericConstrainQueries.lean")
        self.assertEqual(profile.mutation_modules["constraint-tracks-zero"],
                         "Gstyle/Proof/Transfer/Tracking.lean")
        self.assertEqual(profile.mutation_modules["compare-wrong-right-bound"],
                         "Gstyle/Proof/Transfer/ComparisonTransferQueries.lean")
        self.assertEqual(profile.mutation_modules["compare-swaps-values"],
                         "Gstyle/Proof/Transfer/ComparisonQueries.lean")
        self.assertEqual(profile.mutation_modules["compare-continues-after-rejection"],
                         "Gstyle/Proof/Transfer/ComparisonNil.lean")

    def test_baseline_covers_audit_and_mutation_targets(self):
        profile = verify.runtime_facts_profile()
        modules = verify.baseline_modules(profile)
        self.assertIn("Gstyle.Proof.Transfer.Boolean", modules)
        self.assertIn("Gstyle.Proof.Transfer.SymbolResolution", modules)
        self.assertIn(profile.proof_file.removesuffix(".lean").replace("/", "."), modules)
        for proof in profile.mutation_modules.values():
            self.assertIn(proof.removesuffix(".lean").replace("/", "."), modules)
        self.assertNotIn("Gstyle", modules)
        self.assertNotIn("Gstyle.Proof.Transfer.BranchSnapshot", modules)

    def test_only_length_profile_preserves_umbrella_build(self):
        for name in dir(verify):
            if not name.endswith("_profile"):
                continue
            with self.subTest(profile=name):
                modules = verify.baseline_modules(getattr(verify, name)())
                self.assertEqual("Gstyle" in modules, name == "length_profile")

    def test_baseline_handles_line_comments_and_multiple_imports(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "Audit.lean").write_text(
                "-- audit dependencies\n"
                "import Gstyle.First Gstyle.Second -- both are required\n"
                "#print axioms example\n")
            for name in ("First", "Second"):
                target = root / "Gstyle" / (name + ".lean")
                target.parent.mkdir(exist_ok=True)
                target.touch()
            profile = replace(verify.length_profile(), proof_file="Gstyle/First.lean", umbrella=False)
            with patch.object(verify, "PROJECT", root):
                self.assertEqual(verify.baseline_modules(profile), ["Gstyle.First", "Gstyle.Second"])

    def test_baseline_rejects_missing_or_unsupported_imports(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            audit = root / "Audit.lean"
            profile = verify.length_profile()
            for content, message in (
                ("#print axioms example\n", "no project imports"),
                ("public import Gstyle.Proof.Cardinality\n", "unsupported audit import"),
                ("import Gstyle.Missing\n", "missing baseline module"),
            ):
                with self.subTest(content=content), patch.object(verify, "PROJECT", root):
                    audit.write_text(content)
                    with self.assertRaisesRegex(RuntimeError, message):
                        verify.baseline_modules(profile)

    def test_worker_limit_respects_cpu_affinity_and_rejects_invalid_config(self):
        with patch.object(verify.os, "sched_getaffinity", return_value={0, 1, 2, 3}, create=True):
            with patch.dict(os.environ, {}, clear=True):
                self.assertEqual(verify.mutation_workers(), 2)
            for raw, expected in (("1", 1), ("4", 4)):
                with self.subTest(raw=raw), patch.dict(os.environ, {"GSTYLE_MUTATION_WORKERS": raw}):
                    self.assertEqual(verify.mutation_workers(), expected)
            for raw in ("0", "5", "-1", "auto"):
                with self.subTest(raw=raw), patch.dict(os.environ, {"GSTYLE_MUTATION_WORKERS": raw}):
                    with self.assertRaisesRegex(RuntimeError, "GSTYLE_MUTATION_WORKERS"):
                        verify.mutation_workers()
        with patch.object(verify.os, "sched_getaffinity", return_value={0, 1}, create=True):
            with patch.dict(os.environ, {"GSTYLE_MUTATION_WORKERS": "4"}):
                self.assertEqual(verify.mutation_workers(), 2)

    def test_all_mutations_have_unique_bound_targets(self):
        for factory in (getattr(verify, name) for name in dir(verify) if name.endswith("_profile")):
            profile = factory()
            for name, mutation in profile.mutations.items():
                with self.subTest(profile=profile.report.name, mutation=name):
                    source = verify.ROOT / mutation[2] if len(mutation) == 3 else profile.source
                    self.assertIn(source, profile.sources)
                    self.assertEqual(source.read_text().count(mutation[0]), 1)

    def test_all_profile_audits_have_build_targets(self):
        for name in dir(verify):
            if not name.endswith("_profile"):
                continue
            with self.subTest(profile=name):
                self.assertTrue(verify.baseline_modules(getattr(verify, name)()))

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
