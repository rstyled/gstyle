#!/usr/bin/env python3
"""Verify shared proof inputs once, then schedule only missing mutation evidence."""

from concurrent.futures import ThreadPoolExecutor
from dataclasses import replace
import json
import os
from pathlib import Path
import shlex
import shutil
import subprocess
import tempfile

import mutation_cache
import verify


def load_entries(path):
    entries = json.loads(path.read_text())
    if not isinstance(entries, list) or not entries:
        raise RuntimeError("CI profiles must be a nonempty list")
    for index, entry in enumerate(entries):
        required = {"profile", "target", "shard", "argument", "cache-index"}
        if not isinstance(entry, dict) or set(entry) != required:
            raise RuntimeError("invalid CI profile fields")
        if any(not isinstance(entry[name], str) for name in required - {"cache-index"}):
            raise RuntimeError("CI profile values must be strings")
        if type(entry["cache-index"]) is not int or entry["cache-index"] != index:
            raise RuntimeError("CI cache index must match the original matrix order")
        if not entry["profile"] or Path(entry["profile"]).name != entry["profile"] or entry["profile"] in {".", ".."}:
            raise RuntimeError("invalid CI profile report name")
        arguments = tuple(shlex.split(entry["argument"]))
        if arguments not in verify.PROFILE_FACTORIES:
            raise RuntimeError("unknown CI profile argument")
        parse_shard(entry["shard"])
    return entries


def parse_shard(raw):
    parts = raw.split("/")
    if len(parts) != 2 or not all(part.isdecimal() for part in parts):
        raise RuntimeError("invalid CI mutation shard")
    index, count = map(int, parts)
    if count < 1 or index >= count:
        raise RuntimeError("invalid CI mutation shard range")
    return index, count


def prepare_profiles(entries, directory):
    profiles = {}
    identities = {}
    for entry in entries:
        name = entry["profile"]
        identity = (entry["target"], entry["argument"])
        if name in identities and identities[name] != identity:
            raise RuntimeError("conflicting CI profile definitions")
        identities[name] = identity
        if name in profiles:
            continue
        factory = verify.PROFILE_FACTORIES[tuple(shlex.split(entry["argument"]))]
        profile = replace(factory(), report=directory / name)
        profile.report.mkdir(parents=True, exist_ok=True)
        profiles[name] = profile
    return profiles


def validate_coverage(entries, profiles):
    groups = {}
    for entry in entries:
        arguments = tuple(shlex.split(entry["argument"]))
        groups.setdefault(arguments, []).append(entry)
    if set(groups) != set(verify.PROFILE_FACTORIES):
        raise RuntimeError("CI matrix must cover every verification profile")
    for group in groups.values():
        identities = {(entry["profile"], entry["target"]) for entry in group}
        if len(identities) != 1:
            raise RuntimeError("CI profile argument must use one profile and target")
        shards = [parse_shard(entry["shard"]) for entry in group]
        counts = {count for _, count in shards}
        if len(counts) != 1:
            raise RuntimeError("CI profile shard counts must agree")
        count = counts.pop()
        if len(shards) != count or {index for index, _ in shards} != set(range(count)):
            raise RuntimeError("CI profile shards must appear exactly once")
        profile = profiles[group[0]["profile"]]
        controls = set()
        for shard in shards:
            selected = set(verify.shard_mutations(profile.mutations, shard))
            if controls & selected:
                raise RuntimeError("CI mutation control appears more than once")
            controls.update(selected)
        if controls != set(profile.mutations):
            raise RuntimeError("CI matrix omits mutation controls")


def selected_profile(profile, entry):
    shard = parse_shard(entry["shard"])
    return replace(profile, mutations=verify.shard_mutations(profile.mutations, shard))


def check_audit(profile):
    output = verify.run(["lake", "env", "lean", profile.audit], verify.PROJECT,
                        "axioms.log", report=profile.report)
    verify.check_axioms(output, profile.theorems)


def check_sources(profiles, tools):
    generated = {}
    for profile in profiles.values():
        identity = (profile.go_package, profile.generated, profile.bindings)
        if identity not in generated:
            output = profile.report / "regenerated/Gstyle"
            verify.generate(profile, *tools, verify.ROOT, output, "source")
            generated[identity] = output
        verify.check_generated(profile, generated[identity])
    with ThreadPoolExecutor(max_workers=4) as pool:
        list(pool.map(check_audit, profiles.values()))


def fingerprints(entries, profiles, tools):
    return {entry["cache-index"]: mutation_cache.fingerprint(
                verify.PROJECT, verify.ROOT, selected_profile(profiles[entry["profile"]], entry), tools)
            for entry in entries}


def restore_candidates(entries, profiles, keys, staging):
    hits = []
    misses = []
    for entry in entries:
        index = entry["cache-index"]
        profile = replace(selected_profile(profiles[entry["profile"]], entry), report=staging / str(index))
        profile.report.mkdir(parents=True)
        cache = verify.ROOT / "dist/mutation-cache" / str(index)
        record = profile.report / "evidence.json"
        try:
            record.write_bytes((cache / "evidence.json").read_bytes())
        except OSError:
            misses.append(entry)
            continue
        if mutation_cache.restore(profile.report, keys[index], profile, verify.proof_rejected):
            hits.append((entry, profile.report, profile.report))
        else:
            misses.append(entry)
    return hits, misses


def publish_hits(hits, profiles, bound_sources, keys):
    for entry, staged, cache in hits:
        name, index = entry["profile"], entry["cache-index"]
        profile = replace(profiles[name], report=profiles[name].report / f"shard-{index}")
        profile.report.mkdir(parents=True, exist_ok=True)
        for proof in staged.glob("*-proof.log"):
            shutil.copy2(proof, profile.report / proof.name)
        selected = selected_profile(profile, entry)
        verify.write_result(profile, bound_sources[name], list(selected.mutations),
                            parse_shard(entry["shard"]), True, keys[index])
        destination = verify.ROOT / "dist/mutation-cache-hits" / str(index)
        destination.mkdir(parents=True, exist_ok=True)
        shutil.copy2(cache / "evidence.json", destination / "evidence.json")


def publish_outputs(misses, hits):
    values = {"matrix": json.dumps({"include": misses}, separators=(",", ":")),
              "has-misses": str(bool(misses)).lower(), "has-hits": str(bool(hits)).lower()}
    output = os.environ.get("GITHUB_OUTPUT")
    if output:
        with Path(output).open("a") as stream:
            for name, value in values.items():
                stream.write(f"{name}={value}\n")
    print(json.dumps(values), flush=True)


def bootstrap_dependencies(directory):
    manifest = json.loads((verify.PROJECT / "lake-manifest.json").read_text())
    packages = verify.PROJECT / ".lake/packages"
    if all((packages / package["name"]).is_dir() for package in manifest["packages"]):
        return
    # Materialize pinned packages before inspecting Git revisions or building
    # translators. The normal baseline still runs after input fingerprints.
    verify.run(["lake", "build"], verify.PROJECT, "bootstrap.log", report=directory)


def main(config=None):
    entries = load_entries(config or verify.PROJECT / "ci_profiles.json")
    directory = verify.ROOT / "dist/verification/preflight"
    # A failed rerun must not expose successful results from an earlier run.
    shutil.rmtree(verify.ROOT / "dist/mutation-cache-hits", ignore_errors=True)
    shutil.rmtree(directory, ignore_errors=True)
    directory.mkdir(parents=True)
    profiles = prepare_profiles(entries, directory)
    validate_coverage(entries, profiles)
    bound = {name: verify.source_hashes(profile.sources) for name, profile in profiles.items()}
    bootstrap_dependencies(directory)
    perennial = verify.check_dependencies(directory)
    tools = verify.build_generators(perennial, directory)
    keys = fingerprints(entries, profiles, tools)
    modules = sorted({module for profile in profiles.values() for module in verify.baseline_modules(profile)})
    verify.run(["lake", "build", *modules], verify.PROJECT, "build.log", report=directory)
    check_sources(profiles, tools)
    with tempfile.TemporaryDirectory(prefix=".cache-verify-", dir=directory) as scratch:
        hits, misses = restore_candidates(entries, profiles, keys, Path(scratch))
        for name, profile in profiles.items():
            if verify.source_hashes(profile.sources) != bound[name]:
                raise RuntimeError("bound source changed during CI preflight")
        if fingerprints(entries, profiles, tools) != keys:
            raise RuntimeError("verification inputs changed during CI preflight")
        publish_hits(hits, profiles, bound, keys)
        publish_outputs(misses, hits)


if __name__ == "__main__":
    try:
        main()
    except (OSError, RuntimeError, ValueError, subprocess.SubprocessError) as error:
        raise SystemExit(str(error)) from error
