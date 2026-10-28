import {expect, test} from 'bun:test';
import {readFileSync} from 'node:fs';
const m = new Function(readFileSync(new URL('./model.js', import.meta.url), 'utf8') + ';return {indexModel,packageGraph,callGraph,pdgView,trace,sourceText,headerEnd}')();
const pkg = (id: string, imports: string[] = []) => ({id, imports, lines: 10, files: 1});
const symbol = (id: string, p: string, targets: string[] = []) => ({id, package: p, name: id, file: p + '.go', line: 1, calls: targets.map(target => ({target}))});
test('every production function stays in the index; cross-package calls aggregate without losing counts', () => {
 const model = m.indexModel({module: 'repo', packages: [pkg('repo/a', ['repo/b']), pkg('repo/b'), pkg('repo/empty')], symbols: [symbol('a', 'repo/a', ['b', 'b']), symbol('alone', 'repo/a'), symbol('b', 'repo/b', ['a'])]});
 const packages = m.packageGraph(model);
 expect(packages.nodes).toHaveLength(3);
 expect(packages.nodes.find((n: any) => n.id === 'repo/a').functions).toBe(2);
 expect(packages.edges[0].count).toBe(2);
 const calls = m.callGraph(model, 'repo/a');
 expect(calls.nodes.filter((n: any) => n.kind === 'function').map((n: any) => n.id)).toEqual(['a', 'alone']);
 expect(calls.edges).toContainEqual({source: 'a', target: 'out:repo/b', count: 2});
 expect(calls.edges).toContainEqual({source: 'in:repo/b', target: 'a', count: 1});
 expect(model.symbols.size).toBe(3);
});
const known = (value: string) => ({status: 'known', value});
const content = 'func f(é int) {\n if é > 0 {\n  return é\n }\n}\n';
const bytes = (s: string) => new TextEncoder().encode(s).length;
const span = (part: string) => { const i = content.indexOf(part); return {source_id: 's', start: bytes(content.slice(0, i)), end: bytes(content.slice(0, i + part.length))}; };
const node = (id: string, category: string, part: string, kind: string) => ({id, category, source_span: span(part), original_kind: known('*ast.' + kind)});
const doc = {
 sources: [{id: 's', content}], symbols: [],
 nodes: [node('p', 'formal_input', 'é int', 'Field'), node('if', 'control', 'if é > 0 {\n  return é\n }', 'IfStmt'), node('test', 'expression', 'é > 0', 'BinaryExpr'), node('ret', 'return', 'return é', 'ReturnStmt'), {...node('use', 'expression', 'return é', 'Ident'), source_span: (() => {const s=span('return é');return {...s,start:s.end-bytes('é')};})()}],
 edges: [{source: 'p', target: 'test', kind: 'data'}, {source: 'test', target: 'if', kind: 'data'}, {source: 'if', target: 'ret', kind: 'control'}, {source: 'p', target: 'use', kind: 'data'}],
};
test('statement view keeps the body distinct from its control header and preserves external dependencies', () => {
 const graph = m.pdgView(doc);
 expect(graph.nodes.map((n: any) => n.id)).toEqual(['p', 'if', 'ret']);
 expect(graph.nodes.find((n: any) => n.id === 'if').code).toBe('if é > 0');
 expect(graph.nodes.find((n: any) => n.id === 'ret').line).toBe(3);
 expect(graph.edges.map((e: any) => [e.source,e.target,e.kind])).toEqual([['p','if','data'],['if','ret','control'],['p','ret','data']]);
 expect(m.pdgView(doc, 'expressions').nodes).toHaveLength(5);
});
test('source-free exports do not fold operations into unknown control headers', () => {
 const graph = m.pdgView({...doc, sources: []});
 expect(graph.nodes.map((n: any) => n.id)).toEqual(['p', 'if', 'test', 'ret']);
});
test('tracing terminates on cycles and follows only enabled kinds', () => {
 const edges = [{source:'a',target:'b',kind:'data'}, {source:'b',target:'a',kind:'data'}, {source:'b',target:'c',kind:'control'}];
 expect([...m.trace(edges,'a','downstream',new Set(['data']))].sort()).toEqual(['a','b']);
 expect([...m.trace(edges,'c','upstream',new Set(['data','control']))].sort()).toEqual(['a','b','c']);
});
test('control header braces in quoted strings and composite literals stay in the header', () => {
 const text = m.sourceText('if f("{", (T{a: 1})) { body() }');
 expect(text.slice(0, m.headerEnd(text, 0, text.bytes.length))).toBe('if f("{", (T{a: 1})) ');
});
test('braces in control-header comments are not treated as the body', () => {
 const text = m.sourceText('if /* { } */ ready // {\n { body() }');
 expect(text.slice(0, m.headerEnd(text, 0, text.bytes.length))).toBe('if /* { } */ ready // {\n ');
});
