"""Assemble complete, validated evidence after every required job succeeds."""

import json
from pathlib import Path
import re
import shutil

import verify


def assemble(entries, artifacts, destination):
    expected = {entry['cache-index'] for entry in entries}
    if len(expected) != len(entries):
        raise RuntimeError('duplicate mutation cache indices')
    records = {}
    for entry in entries:
        index = entry['cache-index']
        fresh = artifacts / f"{entry['profile']}-proof-{index}" / 'mutation-evidence.json'
        cached = artifacts / 'cached-proof-hits' / str(index) / 'evidence.json'
        source = fresh if fresh.is_file() else cached
        if not source.is_file():
            raise RuntimeError(f'missing verified mutation evidence: {index}')
        record = json.loads(source.read_text())
        if not isinstance(record.get('key'), str) or not re.fullmatch('[a-f0-9]{64}', record['key']):
            raise RuntimeError(f'invalid mutation evidence key: {index}')
        profile = verify.PROFILE_FACTORIES[tuple(entry['argument'].split())]()
        shard = tuple(map(int, entry['shard'].split('/')))
        mutations = verify.shard_mutations(profile.mutations, shard)
        proofs = record.get('proofs')
        if not isinstance(proofs, dict) or set(proofs) != set(mutations):
            raise RuntimeError(f'incomplete mutation evidence: {index}')
        for name, output in proofs.items():
            proof = profile.mutation_modules.get(name, profile.proof_file)
            if not isinstance(output, str) or not verify.proof_rejected(output, proof):
                raise RuntimeError(f'invalid mutation rejection: {index}/{name}')
        records[index] = source
    # Never publish a partial bundle when one control is absent or malformed.
    if destination.exists():
        shutil.rmtree(destination)
    for index, source in records.items():
        target = destination / str(index) / 'evidence.json'
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(source, target)
    print(f'Complete mutation evidence: {len(records)} shards', flush=True)


def main():
    entries = json.loads((verify.PROJECT / 'ci_profiles.json').read_text())
    assemble(entries, verify.ROOT / 'dist/proof-artifacts', verify.ROOT / 'dist/mutation-cache')


if __name__ == '__main__':
    try:
        main()
    except (OSError, RuntimeError, ValueError, KeyError, TypeError) as error:
        raise SystemExit(str(error)) from error
