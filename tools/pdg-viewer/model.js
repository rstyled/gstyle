// Domain layers come from the first path segment below the module root.
const LAYERS = ['domain', 'application', 'adapters', 'internal', 'cmd', 'tools', 'root'];
function layerOf(pkg, module) {
 const head = pkg === module ? '' : pkg.slice(module.length + 1).split('/')[0];
 return LAYERS.includes(head) ? head : 'root';
}
const shortPackage = (pkg, module) => pkg === module ? module.split('/').at(-1) : pkg.slice(module.length + 1);

function indexModel(index) {
 const symbols = new Map(index.symbols.map(s => [s.id, s])), byPackage = new Map(index.packages.map(p => [p.id, []])), callers = new Map();
 for (const s of index.symbols) {
  if (!byPackage.has(s.package)) byPackage.set(s.package, []);
  byPackage.get(s.package).push(s);
  for (const c of s.calls) {
   if (!symbols.has(c.target)) continue;
   if (!callers.has(c.target)) callers.set(c.target, new Map());
   callers.get(c.target).set(s.id, (callers.get(c.target).get(s.id) || 0) + 1);
  }
 }
 for (const list of byPackage.values()) list.sort((a, b) => a.file.localeCompare(b.file) || a.line - b.line);
 return {index, module: index.module, symbols, byPackage, callers, packages: new Map(index.packages.map(p => [p.id, p]))};
}

// Packages and their repository-internal imports, weighted by resolved calls.
function packageGraph(model) {
 const calls = new Map();
 for (const s of model.index.symbols) for (const c of s.calls) {
  const target = model.symbols.get(c.target);
  if (target && target.package !== s.package) { const key = s.package + '\n' + target.package; calls.set(key, (calls.get(key) || 0) + 1); }
 }
 const nodes = model.index.packages.map(p => ({id: p.id, kind: 'package', label: shortPackage(p.id, model.module), layer: layerOf(p.id, model.module), functions: model.byPackage.get(p.id)?.length || 0, lines: p.lines}))
  .sort((a, b) => LAYERS.indexOf(a.layer) - LAYERS.indexOf(b.layer) || a.id.localeCompare(b.id));
 const edges = model.index.packages.flatMap(p => p.imports.filter(id => model.packages.has(id)).map(id => ({source: p.id, target: id, count: calls.get(p.id + '\n' + id) || 0})));
 return {nodes, edges};
}

// A package's functions and calls between them. Calls that cross the package
// boundary collapse into one node per caller package (left) and callee package (right).
function callGraph(model, pkg) {
 const own = model.byPackage.get(pkg) || [], ownIDs = new Set(own.map(s => s.id)), stubs = new Map(), edges = new Map();
 const stub = (side, other) => {
  const id = side + ':' + other;
  if (!stubs.has(id)) stubs.set(id, {id, kind: side === 'in' ? 'caller-package' : 'callee-package', package: other, label: shortPackage(other, model.module), layer: layerOf(other, model.module)});
  return id;
 };
 const link = (source, target, count = 1) => { const key = source + '\n' + target; const e = edges.get(key); if (e) e.count += count; else edges.set(key, {source, target, count}); };
 for (const s of own) {
  for (const c of s.calls) {
   const target = model.symbols.get(c.target);
   if (target) link(s.id, ownIDs.has(target.id) ? target.id : stub('out', target.package));
  }
  for (const [caller, count] of model.callers.get(s.id) || []) {
   const from = model.symbols.get(caller);
   if (from.package !== pkg) link(stub('in', from.package), s.id, count);
  }
 }
 const nodes = own.map(s => ({id: s.id, kind: 'function', label: s.name, symbol: s, layer: layerOf(pkg, model.module)}));
 return {nodes: [...[...stubs.values()].filter(n => n.kind === 'caller-package'), ...nodes, ...[...stubs.values()].filter(n => n.kind === 'callee-package')], edges: [...edges.values()]};
}

// Byte-offset helpers. Spans are UTF-8 byte offsets.
function sourceText(content) {
 const bytes = new TextEncoder().encode(content), starts = [0], decoder = new TextDecoder();
 for (let i = 0; i < bytes.length; i++) if (bytes[i] === 10) starts.push(i + 1);
 const line = offset => { let lo = 0, hi = starts.length - 1; while (lo < hi) { const mid = (lo + hi + 1) >> 1; if (starts[mid] <= offset) lo = mid; else hi = mid - 1; } return lo + 1; };
 return {bytes, starts, line, slice: (a, b) => decoder.decode(bytes.subarray(a, b))};
}

const STATEMENTS = new Set(['control', 'assignment', 'return', 'declaration', 'formal_input', 'formal_output', 'entry', 'exit']);
const fact = f => f?.status === 'known' ? String(f.value) : '';
const spanOf = n => n.source_span && n.source_span.end > n.source_span.start ? n.source_span : null;

// The opening brace of a control statement's body ends its header. Go requires
// composite literals in headers to be parenthesized, so depth-0 "{" is the body.
function headerEnd(text, start, end) {
 let depth = 0, quote = '', comment = '';
 for (let i = start; i < end; i++) {
  const c = text.bytes[i], next = text.bytes[i + 1];
  if (comment) {
   if (comment === 'line' && c === 10) comment = '';
   else if (comment === 'block' && c === 42 && next === 47) { comment = ''; i++; }
   continue;
  }
  if (quote) { if (c === 92 && quote !== '`') i++; else if (String.fromCharCode(c) === quote) quote = ''; continue; }
  if (c === 47 && (next === 47 || next === 42)) { comment = next === 47 ? 'line' : 'block'; i++; continue; }
  if (c === 34 || c === 39 || c === 96) quote = String.fromCharCode(c);
  else if (c === 40 || c === 91) depth++;
  else if (c === 41 || c === 93) depth--;
  else if (c === 123 && depth === 0) return i;
 }
 return end;
}

// Statement view folds each expression into the statement that contains it, the
// way a reader sees code. Dependencies between folded expressions become
// dependencies between statements; edges inside one statement disappear.
function pdgView(doc, detail = 'statements') {
 const texts = new Map((doc.sources || []).filter(s => typeof s.content === 'string').map(s => [s.id, sourceText(s.content)]));
 const textOf = n => texts.get(spanOf(n)?.source_id);
 const owner = detail === 'statements' ? statementOwners(doc.nodes, textOf) : new Map(doc.nodes.map(n => [n.id, n]));
 const members = new Map();
 for (const n of doc.nodes) { const o = owner.get(n.id); if (!members.has(o.id)) members.set(o.id, []); members.get(o.id).push(n); }
 const nodes = doc.nodes.filter(n => owner.get(n.id) === n).map(n => describe(n, members.get(n.id), textOf(n), doc));
 const edges = new Map();
 for (const e of doc.edges) {
  const source = owner.get(e.source).id, target = owner.get(e.target).id;
  if (source === target && detail === 'statements') continue;
  const key = source + '\n' + target + '\n' + e.kind;
  if (!edges.has(key)) edges.set(key, {source, target, kind: e.kind, symbols: new Set(), raw: []});
  const symbol = symbolName(doc, e.symbol_id);
  if (symbol) edges.get(key).symbols.add(symbol);
  edges.get(key).raw.push(e);
 }
 return {nodes, edges: [...edges.values()].map(e => ({...e, symbols: [...e.symbols]}))};
}
function statementOwners(nodes, textOf) {
 const spanned = nodes.filter(spanOf).sort((a, b) => (spanOf(a).end - spanOf(a).start) - (spanOf(b).end - spanOf(b).start));
 const parent = new Map();
 spanned.forEach((n, i) => {
  const s = spanOf(n);
  for (let j = i + 1; j < spanned.length; j++) {
   const q = spanOf(spanned[j]);
   if (q.source_id === s.source_id && q.start <= s.start && q.end >= s.end && q.end - q.start > s.end - s.start) { parent.set(n.id, spanned[j]); break; }
  }
 });
 const owner = new Map();
 for (const n of nodes) {
  let current = n;
  for (;;) {
   if (STATEMENTS.has(current.category)) break;
   const p = parent.get(current.id);
   if (!p) break;
   const text = textOf(p);
   if (p.category === 'control' && (!text || spanOf(current).start >= headerEnd(text, spanOf(p).start, spanOf(p).end))) break;
   current = p;
  }
  owner.set(n.id, current);
 }
 return owner;
}
function describe(n, members, text, doc) {
 const kind = {formal_input: 'param', formal_output: 'result'}[n.category] || (fact(n.original_kind).replace(/^\*?ast\./, '').replace('BasicLit', 'Literal').replace(/(Stmt|Expr|Lit|Spec|Decl)$/, '') || String(n.category || 'operation').replaceAll('_', ' ')).toLowerCase();
 const span = spanOf(n);
 let code = '';
 if (span && text) {
  const end = n.category === 'control' ? headerEnd(text, span.start, span.end) : span.end;
  code = text.slice(span.start, end).split('\n')[0].trim().replace(/\s+/g, ' ');
 }
 if (!code) code = fact(n.attributes?.callee) || symbolName(doc, n.attributes?.symbol_ids?.[0]) || kind.replaceAll('_', ' ');
 return {id: n.id, node: n, category: n.category, kind, code, line: span && text ? text.line(span.start) : null, span, members};
}
function symbolName(doc, id) {
 if (!id) return '';
 return fact(doc.symbols?.find(s => s.id === id)?.name);
}

// Nodes reached from id along enabled edge kinds: direct neighbors, or transitive upstream/downstream.
function trace(edges, id, mode, kinds) {
 const seen = new Set([id]), usable = edges.filter(e => kinds.has(e.kind));
 if (mode === 'direct') { for (const e of usable) if (e.source === id || e.target === id) { seen.add(e.source); seen.add(e.target); } return seen; }
 const next = new Map();
 for (const e of usable) { const [a, b] = mode === 'upstream' ? [e.target, e.source] : [e.source, e.target]; if (!next.has(a)) next.set(a, []); next.get(a).push(b); }
 const queue = [id];
 for (let i = 0; i < queue.length; i++) for (const v of next.get(queue[i]) || []) if (!seen.has(v)) { seen.add(v); queue.push(v); }
 return seen;
}
