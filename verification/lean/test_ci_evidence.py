import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import ci_evidence
import verify


class EvidenceBundleTests(unittest.TestCase):
    def test_bundle_requires_complete_validated_hits_or_fresh_evidence(self):
        entries = [{'profile': 'function-length', 'argument': '', 'shard': f'{index}/4', 'cache-index': index}
                   for index in range(4)]
        profile = verify.length_profile()
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            for entry in entries:
                index = entry['cache-index']
                parent = root / 'artifacts' / (f'function-length-proof-{index}' if index % 2 else f'cached-proof-hits/{index}')
                parent.mkdir(parents=True)
                name = list(profile.mutations)[index]
                output = f'error: {profile.proof_file}:1:0: unsolved goals\nerror: build failed\n'
                record = {'key': 'a' * 64, 'proofs': {name: output}}
                (parent / ('mutation-evidence.json' if index % 2 else 'evidence.json')).write_text(json.dumps(record))
            with patch.object(ci_evidence, 'print', create=True):
                ci_evidence.assemble(entries, root / 'artifacts', root / 'bundle')
            self.assertEqual(len(list((root / 'bundle').rglob('evidence.json'))), 4)
            source = root / 'artifacts/function-length-proof-1/mutation-evidence.json'
            for broken in ({'key': 'a' * 64, 'proofs': {}}, {'key': 'short', 'proofs': record['proofs']},
                           {'key': 'a' * 64, 'proofs': {list(profile.mutations)[1]: 'error: build failed'}}):
                source.write_text(json.dumps(broken))
                with self.assertRaises(RuntimeError):
                    ci_evidence.assemble(entries, root / 'artifacts', root / 'absent-bundle')
                self.assertFalse((root / 'absent-bundle').exists())
            source.unlink()
            with self.assertRaisesRegex(RuntimeError, 'missing verified'):
                ci_evidence.assemble(entries, root / 'artifacts', root / 'absent-bundle')

    def test_matrix_contains_every_control_once(self):
        entries = json.loads((verify.PROJECT / 'ci_profiles.json').read_text())
        self.assertEqual({entry['cache-index'] for entry in entries}, set(range(len(entries))))
        seen = {}
        for entry in entries:
            profile = verify.PROFILE_FACTORIES[tuple(entry['argument'].split())]()
            self.assertEqual(profile.report.name, entry['profile'])
            shard = tuple(map(int, entry['shard'].split('/')))
            controls = verify.shard_mutations(profile.mutations, shard)
            actual = seen.setdefault(entry['argument'], set())
            self.assertFalse(actual & set(controls))
            actual.update(controls)
        self.assertEqual(len(seen), len(verify.PROFILE_FACTORIES))
        for arguments, factory in verify.PROFILE_FACTORIES.items():
            self.assertEqual(seen[' '.join(arguments)], set(factory().mutations))
        self.assertEqual(sum(map(len, seen.values())), 487)
