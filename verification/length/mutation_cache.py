"""Reuse complete mutation evidence only for identical verification inputs."""

import hashlib
import json
import os
from pathlib import Path
import platform
import subprocess
import tempfile


def file_digest(path):
    if path.is_symlink() and path.is_dir():
        target = path.resolve()
        contents = digest_files([item for item in target.rglob('*') if item.is_file()], target)
        return hashlib.sha256(json.dumps([str(path.readlink()), contents], sort_keys=True).encode()).hexdigest()
    return hashlib.sha256(path.read_bytes()).hexdigest()


def digest_files(paths, base):
    return {str(path.relative_to(base)): file_digest(path)
            for path in sorted(set(paths))}


def go_configuration(root):
    settings = json.loads(subprocess.check_output(['go', 'env', '-json'], cwd=root))
    # Derived flags contain a fresh temporary directory on every invocation.
    # The compiler, platform and explicit CGO settings remain in the key.
    settings.pop('GOGCCFLAGS', None)
    return settings


def fingerprint(project, root, profile, tools):
    files = list((project / 'Gstyle').rglob('*'))
    files = [path for path in files if path.is_file()]
    files += list(project.glob('*.lean')) + list(project.glob('*.py'))
    files += [project / name for name in ('lakefile.toml', 'lake-manifest.json', 'lean-toolchain')]
    files += list(profile.sources)
    files += [root / name for name in ('go.mod', 'go.sum', 'go.work', 'go.work.sum') if (root / name).exists()]
    dependencies = {}
    manifest = json.loads((project / 'lake-manifest.json').read_text())
    for package in manifest['packages']:
        directory = project / '.lake/packages' / package['name']
        names = subprocess.check_output(['git', 'ls-files', '-z'], cwd=directory).decode().split('\0')
        untracked = subprocess.check_output(['git', 'ls-files', '--others', '--exclude-standard', '--', '.', ':(exclude).lake'], cwd=directory)
        if untracked:
            raise RuntimeError(f"untracked dependency inputs: {package['name']}")
        dependencies[package['name']] = digest_files([directory / name for name in names if name], directory)
    payload = {
        'schema': 1,
        'files': digest_files(files, root),
        'dependencies': dependencies,
        'tools': {path.name: hashlib.sha256(path.read_bytes()).hexdigest() for path in tools},
        'lean': subprocess.check_output(['lake', 'env', 'lean', '--version'], cwd=project).decode(),
        'go': go_configuration(root),
        'platform': [platform.system(), platform.machine(), platform.python_version()],
        'environment': {name: value for name, value in os.environ.items()
                        if name.startswith(('LEAN_', 'LAKE_', 'GO', 'CGO_', 'CC', 'CXX'))},
        'profile': {
            'package': profile.package, 'go_package': profile.go_package,
            'generated': profile.generated, 'bindings': profile.bindings,
            'proof_file': profile.proof_file, 'audit': profile.audit,
            'mutations': profile.mutations, 'mutation_modules': profile.mutation_modules,
        },
    }
    return hashlib.sha256(json.dumps(payload, sort_keys=True).encode()).hexdigest()


def restore(directory, key, profile, rejected):
    try:
        record = json.loads((directory / 'evidence.json').read_text())
        evidence = record['proofs']
        if record['key'] != key or set(evidence) != set(profile.mutations):
            return False
        for name, output in evidence.items():
            proof = profile.mutation_modules.get(name, profile.proof_file)
            if not isinstance(output, str) or not rejected(output, proof):
                return False
    except (OSError, ValueError, KeyError, TypeError):
        return False
    for name, output in evidence.items():
        (profile.report / f'{name}-proof.log').write_text(output)
    return True


def save(directory, key, profile):
    proofs = {name: (profile.report / f'{name}-proof.log').read_text() for name in profile.mutations}
    directory.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='.write-', dir=directory) as scratch:
        temporary = Path(scratch) / 'evidence.json'
        temporary.write_text(json.dumps({'key': key, 'proofs': proofs}, sort_keys=True) + '\n')
        temporary.replace(directory / 'evidence.json')
