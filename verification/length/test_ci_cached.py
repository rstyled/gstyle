"""Check that cache reuse cannot bypass positive checks or drift guards."""

from contextlib import ExitStack
from dataclasses import replace
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import ci_cached
import verify


class CachedPreflightTests(unittest.TestCase):
    def setup_run(self, directory, entries):
        root = Path(directory)
        config = root / "ci_profiles.json"
        config.write_text(json.dumps(entries))
        profile = replace(verify.length_profile(), report=root / "report")
        return root, config, profile

    def mocks(self, stack, root, profile, arguments=((),)):
        targets = {
            "ROOT": root, "PROJECT": root,
            "PROFILE_FACTORIES": {argument: lambda: profile for argument in arguments},
        }
        for name, value in targets.items():
            stack.enter_context(patch.object(verify, name, value))
        stack.enter_context(patch.object(ci_cached, "bootstrap_dependencies"))
        mocks = {}
        values = {"source_hashes": {"source.go": "fixed"}, "check_dependencies": root,
                  "build_generators": (root / "goose", root / "proofgen"),
                  "baseline_modules": ["Gstyle.Proof.Length"], "run": "audit",
                  "generate": None, "check_generated": None, "check_axioms": None,
                  "write_result": None}
        for name, value in values.items():
            mocks[name] = stack.enter_context(patch.object(verify, name, return_value=value))
        mocks["fingerprint"] = stack.enter_context(patch.object(ci_cached.mutation_cache, "fingerprint", return_value="fixed"))
        mocks["restore"] = stack.enter_context(patch.object(ci_cached.mutation_cache, "restore", return_value=True))
        return mocks

    def entry(self, index, name="first", shard="0/1"):
        return {"profile": name, "target": "verify-length", "shard": shard,
                "argument": "", "cache-index": index}

    def test_mixed_cache_keeps_original_index_and_runs_every_positive_check(self):
        entries = [self.entry(0, shard="0/2"), self.entry(1, shard="1/2"), self.entry(2, "second")]
        entries[2]["argument"] = "--second"
        with tempfile.TemporaryDirectory() as directory, ExitStack() as stack:
            root, config, profile = self.setup_run(directory, entries)
            for index in (0, 2):
                cache = root / "dist/mutation-cache" / str(index)
                cache.mkdir(parents=True)
                (cache / "evidence.json").write_text("saved evidence")
            output = root / "output"
            stack.enter_context(patch.dict(os.environ, {"GITHUB_OUTPUT": str(output)}))
            mocks = self.mocks(stack, root, profile, ((), ("--second",)))
            ci_cached.main(config)
            values = dict(line.split("=", 1) for line in output.read_text().splitlines())
            self.assertEqual(json.loads(values["matrix"]), {"include": [entries[1]]})
            self.assertEqual(values["has-misses"], "true")
            self.assertEqual(values["has-hits"], "true")
            self.assertEqual(mocks["generate"].call_count, 1)
            self.assertEqual(mocks["check_generated"].call_count, 2)
            self.assertEqual(mocks["check_axioms"].call_count, 2)
            self.assertEqual(mocks["check_dependencies"].call_count, 1)
            self.assertEqual(mocks["build_generators"].call_count, 1)
            self.assertEqual(mocks["write_result"].call_count, 2)
            builds = [call for call in mocks["run"].call_args_list if call.args[0][:2] == ["lake", "build"]]
            self.assertEqual(len(builds), 1)
            hit = root / "dist/mutation-cache-hits/2/evidence.json"
            self.assertEqual(hit.read_text(), "saved evidence")

    def test_drift_prevents_results_hit_artifacts_and_outputs(self):
        for drift in ("source", "fingerprint"):
            with self.subTest(drift=drift), tempfile.TemporaryDirectory() as directory, ExitStack() as stack:
                root, config, profile = self.setup_run(directory, [self.entry(0)])
                output = root / "output"
                cache = root / "dist/mutation-cache/0"
                cache.mkdir(parents=True)
                (cache / "evidence.json").write_text("candidate evidence")
                stack.enter_context(patch.dict(os.environ, {"GITHUB_OUTPUT": str(output)}))
                mocks = self.mocks(stack, root, profile)
                if drift == "source":
                    mocks["source_hashes"].side_effect = [{"source.go": "old"}, {"source.go": "new"}]
                else:
                    mocks["fingerprint"].side_effect = ["old", "new"]
                with self.assertRaisesRegex(RuntimeError, "changed"):
                    ci_cached.main(config)
                mocks["write_result"].assert_not_called()
                self.assertFalse(output.exists())
                self.assertFalse((root / "dist/mutation-cache-hits").exists())

    def test_failed_audit_never_attempts_cache_restore(self):
        with tempfile.TemporaryDirectory() as directory, ExitStack() as stack:
            root, config, profile = self.setup_run(directory, [self.entry(0)])
            mocks = self.mocks(stack, root, profile)
            mocks["check_axioms"].side_effect = RuntimeError("bad axioms")
            with self.assertRaisesRegex(RuntimeError, "bad axioms"):
                ci_cached.main(config)
            mocks["restore"].assert_not_called()
            mocks["write_result"].assert_not_called()

    def test_current_diagnostic_classifier_rejects_bad_cached_evidence(self):
        with tempfile.TemporaryDirectory() as directory, ExitStack() as stack:
            root, _, profile = self.setup_run(directory, [self.entry(0)])
            stack.enter_context(patch.object(verify, "ROOT", root))
            cache = root / "dist/mutation-cache/0"
            cache.mkdir(parents=True)
            record = {"key": "fixed", "proofs": {name: "error: unknown identifier" for name in profile.mutations}}
            (cache / "evidence.json").write_text(json.dumps(record))
            staging = root / "staging"
            staging.mkdir()
            hits, misses = ci_cached.restore_candidates([self.entry(0)], {"first": profile}, {0: "fixed"}, staging)
            self.assertEqual(hits, [])
            self.assertEqual(misses, [self.entry(0)])

    def test_runtime_requires_every_profile_and_exact_shard_coverage(self):
        cases = [
            ([self.entry(0)], {(): object(), ("--missing",): object()}, "every verification profile"),
            ([self.entry(0, shard="0/2")], {(): object()}, "exactly once"),
            ([self.entry(0, shard="0/2"), self.entry(1, shard="0/2")], {(): object()}, "exactly once"),
            ([self.entry(0, shard="0/2"), self.entry(1, shard="1/3")], {(): object()}, "counts must agree"),
        ]
        profile = verify.length_profile()
        for entries, factories, message in cases:
            with self.subTest(message=message), patch.object(verify, "PROFILE_FACTORIES", factories):
                with self.assertRaisesRegex(RuntimeError, message):
                    ci_cached.validate_coverage(entries, {"first": profile})

    def test_runtime_rejects_missing_or_duplicate_controls(self):
        profile = verify.length_profile()
        entries = [self.entry(0, shard="0/2"), self.entry(1, shard="1/2")]
        controls = list(profile.mutations)
        cases = [([{controls[0]: ()}, {controls[0]: ()}], "more than once"),
                 ([{controls[0]: ()}, {controls[1]: ()}], "omits mutation controls")]
        for selected, message in cases:
            with self.subTest(message=message), patch.object(verify, "PROFILE_FACTORIES", {(): object()}):
                with patch.object(verify, "shard_mutations", side_effect=selected):
                    with self.assertRaisesRegex(RuntimeError, message):
                        ci_cached.validate_coverage(entries, {"first": profile})

    def test_missing_pinned_dependencies_bootstrap_before_inspection(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "lake-manifest.json").write_text(json.dumps({"packages": [{"name": "example"}]}))
            with patch.object(verify, "PROJECT", root), patch.object(verify, "run") as run:
                ci_cached.bootstrap_dependencies(root)
                run.assert_called_once_with(["lake", "build"], root, "bootstrap.log", report=root)
                run.reset_mock()
                (root / ".lake/packages/example").mkdir(parents=True)
                ci_cached.bootstrap_dependencies(root)
                run.assert_not_called()

    def test_matrix_index_cannot_be_renumbered(self):
        with tempfile.TemporaryDirectory() as directory:
            root, config, _ = self.setup_run(directory, [self.entry(3)])
            with self.assertRaisesRegex(RuntimeError, "original matrix order"):
                ci_cached.load_entries(config)


if __name__ == "__main__":
    unittest.main()
