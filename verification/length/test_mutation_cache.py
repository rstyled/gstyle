from dataclasses import replace
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import mutation_cache
import verify


class MutationCacheTests(unittest.TestCase):
    def test_only_complete_matching_proof_evidence_is_reused(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            profile = replace(verify.length_profile(), report=root, mutations={'sample': ('before', 'after')})
            log = root / 'sample-proof.log'
            output = f'error: {profile.proof_file}:2:0: unsolved goals\nerror: build failed\n'
            log.write_text(output)
            mutation_cache.save(root / 'cache', 'key', profile)
            log.unlink()
            self.assertFalse(mutation_cache.restore(root / 'cache', 'different', profile, verify.proof_rejected))
            self.assertFalse(log.exists())
            self.assertTrue(mutation_cache.restore(root / 'cache', 'key', profile, verify.proof_rejected))
            self.assertEqual(log.read_text(), output)
            for evidence in ({}, {'sample': 'error: unknown module'}, {'sample': output, 'extra': output},
                             {'sample': 1}, [], None):
                (root / 'cache/evidence.json').write_text(json.dumps({'key': 'key', 'proofs': evidence}))
                self.assertFalse(mutation_cache.restore(root / 'cache', 'key', profile, verify.proof_rejected))
            for content in ('broken json', '{}', '[]', 'null'):
                (root / 'cache/evidence.json').write_text(content)
                self.assertFalse(mutation_cache.restore(root / 'cache', 'key', profile, verify.proof_rejected))

    def test_fingerprint_tracks_source_proof_translator_and_dependency_changes(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            project = root / 'verification'
            (project / 'Gstyle').mkdir(parents=True)
            dependency = project / '.lake/packages/example'
            dependency.mkdir(parents=True)
            paths = [project / 'Gstyle/Proof.lean', project / 'verify.py', project / 'mutation_cache.py',
                     project / 'Audit.lean', project / 'lakefile.toml', project / 'lean-toolchain',
                     root / 'input.go', root / 'go.mod', root / 'goose', root / 'proofgen', dependency / 'Input.lean']
            for path in paths:
                path.write_text('original')
            (project / 'lake-manifest.json').write_text(json.dumps({'packages': [{'name': 'example'}]}))
            profile = replace(verify.length_profile(), source=root / 'input.go', sources=(root / 'input.go',))

            def command(arguments, **kwargs):
                if arguments[:3] == ['git', 'ls-files', '-z']:
                    return b'Input.lean\0'
                if arguments[0] == 'git':
                    return b''
                if arguments[0] == 'go':
                    return b'{"GOVERSION": "go1.26", "GOGCCFLAGS": "temporary"}'
                return b'fixed tool version'

            with patch.object(mutation_cache.subprocess, 'check_output', side_effect=command):
                def key(selected=profile):
                    return mutation_cache.fingerprint(project, root, selected, (root / 'goose', root / 'proofgen'))
                original = key()
                self.assertEqual(key(), original)
                for path in paths:
                    with self.subTest(path=path.name):
                        path.write_text('changed')
                        self.assertNotEqual(key(), original)
                        path.write_text('original')
                self.assertNotEqual(key(replace(profile, mutations={'new': ('before', 'after')})), original)
                self.assertNotEqual(key(replace(profile, proof_file='Different.lean')), original)
                with patch.dict(mutation_cache.os.environ, {'LEAN_NUM_THREADS': '17'}):
                    self.assertNotEqual(key(), original)
            with patch.object(mutation_cache.subprocess, 'check_output', return_value=b'untracked.lean'):
                with self.assertRaisesRegex(RuntimeError, 'untracked dependency'):
                    key()

    def test_cache_hits_keep_preflight_checks_and_drift_cannot_publish(self):
        manifest = json.loads((verify.PROJECT / 'lake-manifest.json').read_text())
        revisions = {package['name']: package['rev'] for package in manifest['packages']}
        for drift in (False, True):
            with self.subTest(drift=drift), tempfile.TemporaryDirectory() as temporary:
                root = Path(temporary)
                profile = replace(verify.length_profile(), report=root / 'report')

                def run(command, cwd, log, **kwargs):
                    if command[:2] == ['git', 'rev-parse']:
                        return revisions[cwd.name]
                    if log == 'axioms.log':
                        return '\n'.join(f"'{name}' depends on axioms: [propext]" for name in profile.theorems)
                    return ''

                with patch.dict(verify.os.environ, {'GSTYLE_MUTATION_CACHE': str(root / 'cache')}), \
                     patch.object(verify, 'run', side_effect=run) as commands, \
                     patch.object(verify, 'generate') as generate, \
                     patch.object(verify, 'check_generated') as binding, \
                     patch.object(mutation_cache, 'fingerprint', side_effect=['key', 'changed' if drift else 'key']), \
                     patch.object(mutation_cache, 'restore', return_value=True), \
                     patch.object(mutation_cache, 'save') as save, \
                     patch.object(verify, 'check_mutations') as mutations, \
                     patch.object(verify, 'print', create=True):
                    if drift:
                        with self.assertRaisesRegex(RuntimeError, 'inputs changed'):
                            verify.main(profile)
                        self.assertFalse((profile.report / 'result.json').exists())
                    else:
                        verify.main(profile)
                        result = json.loads((profile.report / 'result.json').read_text())
                        self.assertEqual(result['mutation_evidence'], 'reused')
                    mutations.assert_not_called()
                    save.assert_not_called()
                    generate.assert_called_once()
                    binding.assert_called_once()
                    self.assertIn('build.log', [call.args[2] for call in commands.call_args_list])
                    self.assertIn('axioms.log', [call.args[2] for call in commands.call_args_list])
