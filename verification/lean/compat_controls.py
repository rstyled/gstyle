"""Require the pinned external oracle to reject corrupt raw producer outputs."""
import argparse
import copy
import importlib
import json
from pathlib import Path
import subprocess
import sys

ORACLE_REVISION = "7366049160309bb0a764c57f7e07ae543a039048"


def select(records, **fields):
    return next(record for record in records
                if all(record.get(key) == value for key, value in fields.items()))


def remove_operation(doc):
    node = select(doc['nodes'], category='computation')
    doc['nodes'].remove(node)
    doc['edges'] = [edge for edge in doc['edges']
                    if node['id'] not in (edge['source'], edge['target'])]


def change_literal(doc):
    node = next(node for node in doc['nodes'] if 'literal' in node['attributes'])
    node['attributes']['literal']['value'] = '99'


def remove_edge(doc, kind):
    doc['edges'].remove(semantic_edge(doc, kind))


def semantic_edge(doc, kind):
    boundaries = {node['id'] for node in doc['nodes'] if node['category'] == 'formal_output'}
    return next(edge for edge in doc['edges'] if edge['kind'] == kind
                and edge['source'] not in boundaries and edge['target'] not in boundaries)


def reverse_edge(doc):
    edge = semantic_edge(doc, 'data')
    edge['source'], edge['target'] = edge['target'], edge['source']


def duplicate_edge(doc):
    edge = copy.deepcopy(semantic_edge(doc, 'data'))
    edge['id'] = 'fault-duplicate'
    doc['edges'].append(edge)


def killed_definition(doc):
    binding = select(doc['symbols'], role='local')['id']
    definitions = [item for item in doc['definitions'] if item['symbol_id'] == binding]
    initial = definitions[0]
    edge = next(edge for edge in doc['edges']
                if edge.get('symbol_id') == binding and 'definition_id' in edge)
    assert edge['definition_id'] != initial['id']
    edge.update(source=initial['node_id'], definition_id=initial['id'])


def change_binding(doc):
    edge = next(edge for edge in doc['edges'] if 'definition_id' in edge)
    other = next(item for item in doc['definitions'] if item['symbol_id'] != edge['symbol_id'])
    edge.update(source=other['node_id'], symbol_id=other['symbol_id'], definition_id=other['id'])


def eager_alternative(doc):
    edge = select(doc['edges'], subkind='value_alternative')
    edge['subkind'] = 'scalar-value' if doc['producer']['language'] == 'go' else 'value_operand'
    edge.pop('outcome', None)
    edge.pop('annotations', None)


def change_span(doc):
    node = select(doc['nodes'], category='read')
    other = select(doc['nodes'], category='computation')
    node['source_span'] = copy.deepcopy(other['source_span'])


def change_digest(doc):
    doc['sources'][0]['content_identity']['value'] = 'sha256:' + '0' * 64


def incomplete_nodes(doc):
    doc['extraction_evidence']['capabilities']['node_extraction']['status'] = 'approximated'


def incomplete_conformance(doc):
    doc['extraction_evidence']['conformance']['status'] = 'non_conformant'


FAULTS = [
    ('branch', 'missing-operation', remove_operation),
    ('branch', 'wrong-literal', change_literal),
    ('branch', 'missing-data', lambda doc: remove_edge(doc, 'data')),
    ('branch', 'missing-execution', lambda doc: remove_edge(doc, 'execution')),
    ('branch', 'missing-control', lambda doc: remove_edge(doc, 'control')),
    ('branch', 'reversed-data', reverse_edge),
    ('branch', 'duplicate-data', duplicate_edge),
    ('branch', 'killed-definition', killed_definition),
    ('branch', 'wrong-binding', change_binding),
    ('short_circuit', 'eager-alternative', eager_alternative),
    ('branch', 'wrong-span', change_span),
    ('branch', 'wrong-source-digest', change_digest),
    ('identity', 'incomplete-node-evidence', incomplete_nodes),
    ('identity', 'incomplete-conformance', incomplete_conformance),
]


def rejection(doc, name, language, directory, oracle):
    try:
        graph = oracle['adapter'].normalize(doc, language)
        if oracle['graph'].equivalent(oracle['expectations'].case(name), graph) is None:
            return 'graph'
        if oracle['report'].evidence_gaps(doc['extraction_evidence']):
            return 'evidence'
        source = (directory / ('f.go' if language == 'go' else 'f.rs')).read_bytes()
        bundle = None
        if language == 'rust':
            bundle = json.loads((directory / 'rust/unit.evidence.json').read_text())
        content, _ = oracle['source_correspondence'].resolve_source(doc, source, language, bundle)
        if name == 'branch':
            anchors = oracle['anchors']
            oracle['source_correspondence'].correspondence(doc, content, anchors, directory.name == 'renamed')
    except (ValueError, KeyError, IndexError, TypeError) as error:
        return str(error)
    return None


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--suite', required=True, type=Path)
    parser.add_argument('--results', required=True, type=Path)
    parser.add_argument('--out', required=True, type=Path)
    args = parser.parse_args()
    revision = subprocess.check_output(['git', '-C', str(args.suite), 'rev-parse', 'HEAD'], text=True).strip()
    if revision != ORACLE_REVISION:
        raise RuntimeError('The independent oracle revision differs from the reviewed pin')
    sys.path.insert(0, str(args.suite.resolve()))
    oracle = {name: importlib.import_module(name) for name in
              ('adapter', 'expectations', 'graph', 'report', 'source_correspondence')}
    oracle['anchors'] = json.loads((args.suite / 'cases/branch/correspondence.json').read_text())['anchors']
    documents = {}
    for name in oracle['expectations'].CASES:
        for variant in ('original', 'renamed'):
            directory = args.results / name / variant
            for language in ('go', 'rust'):
                doc = json.loads((directory / (language + '.raw.json')).read_text())
                reason = rejection(doc, name, language, directory, oracle)
                if reason is not None:
                    raise RuntimeError(f'Unmodified {name}/{variant}/{language} fails: {reason}')
                documents[name, variant, language] = doc
    results = []
    for name, fault, mutate in FAULTS:
        for variant in ('original', 'renamed'):
            for language in ('go', 'rust'):
                doc = copy.deepcopy(documents[name, variant, language])
                mutate(doc)
                reason = rejection(doc, name, language, args.results / name / variant, oracle)
                results.append(dict(case=name, variant=variant, language=language,
                                    fault=fault, rejected=reason is not None, reason=reason))
    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_text(json.dumps(dict(oracle_revision=revision, controls=results), indent=2) + '\n')
    failures = [item for item in results if not item['rejected']]
    if failures:
        raise RuntimeError(f'Corrupt producer output passes the oracle: {failures}')
    print(f'{len(documents)} original exports pass; {len(results)} corrupt exports are rejected')


if __name__ == '__main__':
    main()
