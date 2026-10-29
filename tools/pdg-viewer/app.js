// gstyle atlas: repository → package → function → dependence graph, on one canvas.
const $ = s => document.querySelector(s);
const esc = s => String(s ?? '').replace(/[&<>"']/g, c => ({'&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'}[c]));
const plural = (n, word) => `${n.toLocaleString()} ${word}${n === 1 ? '' : 's'}`;
const clamp = (v, lo, hi) => Math.max(lo, Math.min(hi, v));
const clip = (s, n) => s.length > n ? s.slice(0, n - 1) + '…' : s;
const base = path => path.split('/').at(-1);
const embedded = JSON.parse($('#pdg-data').textContent);
const standalone = Array.isArray(embedded);
const KINDS = {data: 'value dependency', control: 'lexical nesting', execution: 'must precede'};

const state = {
 model: null, view: null, selected: null, hover: null,
 pdg: {detail: 'statements', kinds: new Set(['data', 'control']), mode: 'direct'},
 sources: new Map(), sourceBlocks: new Map(), docs: new Map(), layouts: new Map(),
 imports: false, pendingDocs: new Map(),
 callScope: 'neighbors',
 allCalls: true,
};

// ── routing ────────────────────────────────────────────────────────────────
function parseRoute() {
 let parts;
 try { parts = location.hash.replace(/^#\/?/, '').split('/').map(decodeURIComponent); }
 catch { return {view: 'invalid'}; }
 const [view, a, b] = parts;
 if (view === 'pkg' && a) return {view: 'package', pkg: a, fn: b || null};
 if (view === 'pdg' && a) return {view: 'pdg', fn: a};
 if (standalone) return {view: 'pdg', fn: '0'};
 if (view === 'all') return {view: 'all', fn: a || null, pkg: b || null};
 return {view: 'repo', pkg: a || null};
}
const href = {
 repo: () => '#/',
 all: (fn, pkg) => '#/all' + (fn || pkg ? '/' + encodeURIComponent(fn || '') : '') + (pkg ? '/' + encodeURIComponent(pkg) : ''),
 pkg: (pkg, fn) => '#/pkg/' + encodeURIComponent(pkg) + (fn ? '/' + encodeURIComponent(fn) : ''),
 pdg: fn => '#/pdg/' + encodeURIComponent(fn),
};
function go(hash, replace = false) {
 if (location.hash === hash) return;
 if (replace) { history.replaceState(null, '', hash); route(); } else location.hash = hash;
}

// ── camera ─────────────────────────────────────────────────────────────────
const canvas = $('#canvas'), stage = $('#stage');
const cam = {x: 0, y: 0, k: 1, frame: 0};
let cameraFrame = 0, labelScale = null;
function applyCamera() {
 if (cameraFrame) return;
 cameraFrame = requestAnimationFrame(() => { cameraFrame = 0; renderCamera(); });
}
function renderCamera() {
 const w = canvas.clientWidth, h = canvas.clientHeight;
 canvas.setAttribute('viewBox', `${cam.x} ${cam.y} ${w / cam.k} ${h / cam.k}`);
 canvas.classList.toggle('far', cam.k < .5);
 $('#zoom-level').textContent = `${Math.round(cam.k * 100)}%`;
 if (!state.view?.overview && labelScale !== cam.k) {
  labelScale = cam.k;
  stage.style.setProperty('--map-label-size', `${12 / cam.k}px`);
  stage.style.setProperty('--map-heading-size', `${14 / cam.k}px`);
  for (const text of canvas.querySelectorAll('.package-frame text')) {
   const chars = Math.max(3, Math.floor((Number(text.dataset.width) - 32) * cam.k / 7));
   text.textContent = clip(text.dataset.label, chars);
  }
 }
 state.view?.overview?.camera();
 let grid = 24 * cam.k;
 while (grid < 14) grid *= 2;
 while (grid > 30) grid /= 2;
 stage.style.setProperty('--grid', grid + 'px');
 stage.style.setProperty('--gx', (-cam.x * cam.k) % grid + 'px');
 stage.style.setProperty('--gy', (-cam.y * cam.k) % grid + 'px');
}
function moveCamera(target, animate = true) {
 cancelAnimationFrame(cam.frame);
 if (!animate || matchMedia('(prefers-reduced-motion: reduce)').matches) { Object.assign(cam, target); applyCamera(); return; }
 const from = {x: cam.x, y: cam.y, k: cam.k}, start = performance.now();
 const step = now => {
  const t = Math.min(1, (now - start) / 260), e = 1 - (1 - t) ** 3;
  cam.k = from.k * (target.k / from.k) ** e;
  const w = canvas.clientWidth, h = canvas.clientHeight;
  const cx = (from.x + w / 2 / from.k) + ((target.x + w / 2 / target.k) - (from.x + w / 2 / from.k)) * e;
  const cy = (from.y + h / 2 / from.k) + ((target.y + h / 2 / target.k) - (from.y + h / 2 / from.k)) * e;
  cam.x = cx - w / 2 / cam.k; cam.y = cy - h / 2 / cam.k;
  applyCamera();
  if (t < 1) cam.frame = requestAnimationFrame(step);
 };
 cam.frame = requestAnimationFrame(step);
}
function fit(animate = true) {
 const layout = state.view?.layout;
 if (!layout) return;
 const w = canvas.clientWidth, h = canvas.clientHeight, pad = 40, top = state.view.key === 'all' ? 88 : 40, bottom = 80;
 const k = clamp(Math.min((w - pad * 2) / layout.width, (h - top - bottom) / layout.height), .04, 1.1);
 moveCamera({k, x: layout.width / 2 - w / 2 / k, y: layout.height / 2 - (h + top - bottom) / 2 / k}, animate);
}
function centerOn(id) {
 const p = state.view?.layout.positions.get(id);
 if (!p) return;
 const w = canvas.clientWidth, h = canvas.clientHeight, k = Math.max(cam.k, .8);
 const visible = p.x - p.w / 2 > cam.x && p.x + p.w / 2 < cam.x + w / cam.k && p.y - p.h / 2 > cam.y && p.y + p.h / 2 < cam.y + h / cam.k;
 if (!visible || cam.k < .5) moveCamera({k, x: p.x - w / 2 / k, y: p.y - h / 2 / k});
}
function zoomAt(factor, px = canvas.clientWidth / 2, py = canvas.clientHeight / 2) {
 cancelAnimationFrame(cam.frame);
 const k = clamp(cam.k * factor, .04, 3), wx = cam.x + px / cam.k, wy = cam.y + py / cam.k;
 Object.assign(cam, {k, x: wx - px / k, y: wy - py / k});
 applyCamera();
}
canvas.addEventListener('wheel', e => { e.preventDefault(); const r = canvas.getBoundingClientRect(); zoomAt(Math.exp(-e.deltaY * (e.ctrlKey ? .01 : .0015)), e.clientX - r.left, e.clientY - r.top); }, {passive: false});
let drag = null;
canvas.addEventListener('pointerdown', e => { if (e.button !== 0) return; cancelAnimationFrame(cam.frame); canvas.style.cursor = ''; drag = {x: e.clientX, y: e.clientY, cx: cam.x, cy: cam.y, moved: false}; });
addEventListener('pointermove', e => {
 if (!drag) return;
 const dx = e.clientX - drag.x, dy = e.clientY - drag.y;
 if (!drag.moved && Math.hypot(dx, dy) < 4) return;
 if (!drag.moved) { drag.moved = true; if (state.hover) { state.hover = null; paint(); } canvas.classList.add('dragging'); canvas.setPointerCapture?.(e.pointerId); }
 cam.x = drag.cx - dx / cam.k; cam.y = drag.cy - dy / cam.k; applyCamera();
});
addEventListener('pointerup', () => { if (drag?.moved) setTimeout(() => { drag = null; applyCamera(); }); else drag = null; canvas.classList.remove('dragging'); });
addEventListener('pointercancel', () => { drag = null; canvas.classList.remove('dragging'); applyCamera(); });
new ResizeObserver(applyCamera).observe(canvas);
$('#zoom-in').onclick = () => zoomAt(1.3);
$('#zoom-out').onclick = () => zoomAt(1 / 1.3);
$('#fit').onclick = () => fit();
$('#expand').onclick = () => {
 const expanded = $('.app').classList.toggle('canvas-wide');
 $('#expand').setAttribute('aria-pressed', expanded);
 $('#expand').setAttribute('aria-label', expanded ? 'Restore panels' : 'Expand canvas');
 requestAnimationFrame(() => fit());
};

// ── graph drawing ──────────────────────────────────────────────────────────
const MARKERS = ['edge', 'data', 'control', 'execution'];
function drawGraph(key, graph, spec) {
 if (!state.layouts.has(key)) state.layouts.set(key, spec.layout ? spec.layout(graph.nodes, graph.edges) : graphLayout(graph.nodes, graph.edges, {size: spec.size, rankGap: spec.rankGap}));
 const layout = state.layouts.get(key);
 if (spec.overview) return drawOverview(key, graph, layout, spec);
 let edges = '', labels = '', nodes = '';
 graph.edges.forEach((e, i) => {
  const kind = spec.edgeKind?.(e) || '', marker = MARKERS.includes(kind) ? kind : 'edge';
  const route = layout.routes[i], d = route ? routePath(route, layout.direction === 'down') : selfLoop(layout.positions.get(e.source));
  if (!d) return;
  const axis = layout.direction === 'down' ? 'y' : 'x';
  const back = route && route[0][axis] > route.at(-1)[axis];
  edges += `<path class="edge ${kind}${back ? ' back' : ''}" d="${d}" data-e="${i}" style="stroke-width:${spec.edgeWidth?.(e) || 1.2}" marker-end="url(#m-${marker})"/>`;
  const label = spec.edgeLabel?.(e);
  if (label && route) { const m = midpoint(route); labels += `<text class="edge-label" data-e="${i}" x="${m.x}" y="${m.y - 5}" text-anchor="middle">${esc(label)}</text>`; }
 });
 for (const n of graph.nodes) {
  const p = layout.positions.get(n.id);
  nodes += `<g class="node ${esc(spec.nodeClass?.(n) || '')}" data-id="${esc(n.id)}" transform="translate(${p.x - p.w / 2},${p.y - p.h / 2})" role="button" aria-label="${esc(n.label || n.code)}" tabindex="0"><title>${esc(spec.title?.(n) || n.label || n.code)}</title><rect class="box" width="${p.w}" height="${p.h}" rx="${spec.radius?.(n, p) ?? 7}"/>${spec.node(n, p)}</g>`;
 }
 const tray = layout.tray ? `<text class="tray-label" x="${layout.tray.x}" y="${layout.tray.y - 16}">${esc(spec.trayLabel)}</text>` : '';
 const defs = MARKERS.map(k => `<marker id="m-${k}" viewBox="0 0 8 8" refX="7.5" refY="4" markerWidth="7" markerHeight="7" markerUnits="userSpaceOnUse" orient="auto"><path d="M0,0.8 L8,4 L0,7.2z" style="fill:var(--${k})"/></marker>`).join('');
 const sections = (layout.sections || []).map(s => `<g class="map-section"><text x="${s.x}" y="${s.y + 10}" style="fill:${layerVar(s.layer)}">${esc(s.layer)}${s.continued ? " · continued" : ""}</text><text class="count" x="${s.x + s.w}" y="${s.y + 10}" text-anchor="end">${s.count}</text><path d="M${s.x},${s.y + 20}h${s.w}"/></g>`).join('');
 const frames = (layout.frames || []).map(p => `<g class="package-frame" data-package="${esc(p.id)}"><rect x="${p.x}" y="${p.y}" width="${p.w}" height="${p.h}" rx="10"/><title>${esc(p.label)} · ${p.count.toLocaleString()} functions</title><text x="${p.x + 16}" y="${p.y + 60}" data-label="${esc(p.label)} · ${p.count.toLocaleString()}" data-width="${p.w}">${esc(p.label)} · ${p.count.toLocaleString()}</text></g>`).join('');
 canvas.innerHTML = `<defs>${defs}</defs><g>${frames}${sections}${tray}${edges}${labels}${nodes}</g>`;
 const nodeEls = new Map([...canvas.querySelectorAll('.node')].map(el => [el.dataset.id, el]));
 const edgeEls = graph.edges.map(() => []);
 for (const el of canvas.querySelectorAll('[data-e]')) edgeEls[Number(el.dataset.e)].push(el);
 return {key, graph, layout, nodeEls, edgeEls, spec};
}
function selfLoop(p) {
 if (!p) return '';
 const x = p.x + p.w / 2 - 28, y = p.y - p.h / 2;
 return `M${x},${y} C${x},${y - 30} ${x + 30},${y - 30} ${x + 22},${y - 1}`;
}
function midpoint(route) {
 let best = route[0], gap = 0;
 for (let i = 1; i < route.length; i++) { const dx = Math.abs(route[i].x - route[i - 1].x); if (dx > gap) { gap = dx; best = {x: (route[i].x + route[i - 1].x) / 2, y: (route[i].y + route[i - 1].y) / 2}; } }
 return best;
}
// Highlights the hovered node, else the selection, with the view's notion of related nodes.
function paint() {
 const view = state.view;
 if (!view) return;
 if (view.overview) {
  const center = state.hover ?? state.selected;
  canvas.classList.toggle('focus', !!center);
  view.overview.highlight(center, state.selected);
  return;
 }
 const center = state.hover ?? state.selected;
 const related = center && view.nodeEls.has(center) ? view.related(center) : null;
 canvas.classList.toggle('focus', !!related);
 for (const [id, el] of view.nodeEls) { el.classList.toggle('on', !!related?.has(id)); el.classList.toggle('selected', id === state.selected); }
 view.graph.edges.forEach((e, i) => { const on = !!related && view.edgeOn(e, center, related); for (const el of view.edgeEls[i]) el.classList.toggle('on', on); });
}
canvas.addEventListener('pointerover', e => { if (state.view?.overview) return; const id = e.target.closest('.node')?.dataset.id; if (id !== undefined && !drag) { state.hover = id; paint(); } });
canvas.addEventListener('pointerout', e => { if (state.view?.overview) return; if (e.target.closest('.node') && !e.relatedTarget?.closest?.('.node')) { state.hover = null; paint(); } });
const hitAt = e => {
 const r = canvas.getBoundingClientRect();
 return state.view?.overview?.hit(cam.x + (e.clientX - r.left) / cam.k, cam.y + (e.clientY - r.top) / cam.k);
};
let hoverFrame = 0, pointer = null;
canvas.addEventListener('pointermove', e => {
 if (!state.view?.overview || drag) return;
 pointer = {clientX: e.clientX, clientY: e.clientY};
 if (hoverFrame) return;
 hoverFrame = requestAnimationFrame(() => {
  hoverFrame = 0;
  if (!state.view?.overview || !pointer || drag) return;
  const hit = hitAt(pointer), id = hit?.id || null;
  canvas.style.cursor = hit ? 'pointer' : '';
  if (id === state.hover) return;
  state.hover = id;
  canvas.querySelector('#overview-hover-title').textContent = id ? state.view.spec.title(state.view.overview.index.nodes.get(id)) : '';
  paint();
 });
});
canvas.addEventListener('pointerleave', () => {
 pointer = null;
 canvas.querySelector('#overview-hover-title')?.replaceChildren();
 if (state.view?.overview && state.hover) { state.hover = null; paint(); }
});
canvas.addEventListener('click', e => {
 if (drag?.moved) return;
 const hit = hitAt(e), id = e.target.closest('.node')?.dataset.id ?? hit?.id;
 const pkg = id === undefined ? e.target.closest('.package-frame')?.dataset.package || hit?.pkg : null;
 if (pkg && state.route.view === 'all') { go(href.all(null, pkg), true); return; }
 if (id === undefined) { if (state.selected) state.view.select(null); return; }
 state.view.select(id);
});
canvas.addEventListener('dblclick', e => {
 const hit = hitAt(e), id = e.target.closest('.node')?.dataset.id ?? hit?.id, pkg = e.target.closest('.package-frame')?.dataset.package || hit?.pkg;
 if (id !== undefined) state.view.open?.(id);
 else if (pkg) go(href.pkg(pkg));
});
canvas.addEventListener('keydown', e => {
 if (state.view?.overview) {
  const nodes = state.view.graph.nodes, index = nodes.findIndex(n => n.id === state.selected);
  let next;
  if (['ArrowRight', 'ArrowDown'].includes(e.key)) next = Math.min(nodes.length - 1, index + 1);
  if (['ArrowLeft', 'ArrowUp'].includes(e.key)) next = Math.max(0, index - 1);
  if (e.key === 'Home') next = 0;
  if (e.key === 'End') next = nodes.length - 1;
  if (next !== undefined && nodes[next]) { e.preventDefault(); e.stopPropagation(); state.view.select(nodes[next].id); return; }
  if (['Enter', ' '].includes(e.key) && !state.selected && nodes.length) { e.preventDefault(); e.stopPropagation(); state.view.select(nodes[0].id); return; }
 }
 const id = e.target.closest('.node')?.dataset.id;
 if (id === undefined || !['Enter', ' '].includes(e.key)) return;
 e.preventDefault(); e.stopPropagation();
 if (state.selected === id && e.key === 'Enter') state.view.open?.(id);
 else state.view.select(id);
});
const neighbors = (graph, id) => { const s = new Set([id]); for (const e of graph.edges) if (e.source === id || e.target === id) { s.add(e.source); s.add(e.target); } return s; };
const touches = (e, id) => e.source === id || e.target === id;

// ── views ──────────────────────────────────────────────────────────────────
const textWidth = (text, size, mono) => text.length * size * (mono ? .61 : .56);
const layerVar = layer => `var(--${layer})`;

function showRepo(route) {
 const model = state.model, graph = packageGraph(model);
 if (route.pkg && !model.packages.has(route.pkg)) return notFound('This package is not in the index.');
 const entry = new Set(model.index.symbols.filter(s => s.role === 'entrypoint').map(s => s.package));
 const view = drawGraph('repo', graph, {
  layout: packageLayout,
  size: n => ({w: clamp(48 + textWidth(n.label, 13), 150, 320), h: 52}),
  nodeClass: n => 'package' + (entry.has(n.id) ? ' entry' : ''),
  node: n => `<circle class="badge" cx="15" cy="16" r="3" style="fill:${layerVar(n.layer)}"/><text class="name" x="26" y="20">${esc(clip(n.label.replace(new RegExp('^' + n.layer + '/'), ''), 22))}</text><text class="meta" x="26" y="36">${plural(n.functions, 'function')}${entry.has(n.id) ? ' · entrypoint' : ''}</text>`,
  edgeWidth: e => 1 + Math.min(2.4, Math.log2(1 + e.count) * .35),
  edgeLabel: e => e.count ? plural(e.count, 'call') : 'imports',
  trayLabel: 'No imports to or from other packages in this repository',
 });
 Object.assign(view, {
  related: id => neighbors(graph, id), edgeOn: touches,
  select: id => go(id ? '#/' + encodeURIComponent(id) : '#/', true),
  open: id => go(href.pkg(id)),
 });
 setView(view, route.pkg);
 canvas.classList.add('package-map');
 canvas.classList.toggle('all-imports', state.imports);
 legend('<span>Select to trace imports</span><span>Double-click to open calls</span>');
 toolbar(`<div class="view-label">Package map <span>${graph.nodes.length}</span></div>${repositorySwitcher(false)}<div class="seg"><button id="all-imports" aria-pressed="${state.imports}">All imports</button></div>`);
 $('#all-imports').onclick = () => { state.imports = !state.imports; canvas.classList.toggle('all-imports', state.imports); $('#all-imports').setAttribute('aria-pressed', state.imports); };
 crumbs([[model.module, null]]);
 inspector(route.pkg ? packagePanel(route.pkg, true) : repoPanel(graph));
}

function repositorySwitcher(all) {
 return `<div class="seg"><a href="${href.repo()}" ${all ? '' : 'aria-current="page"'}>Packages</a><a href="${href.all()}" ${all ? 'aria-current="page"' : ''}>All functions</a></div>`;
}

function showAll(route) {
 const model = state.model;
 const graph = model.repositoryGraph ||= repositoryGraph(model), packages = model.packageNodes ||= packageGraph(model).nodes;
 const symbol = model.symbols.get(route.fn);
 if (route.fn && !symbol) return notFound('This function is not in the index.');
 if (route.pkg && !model.packages.has(route.pkg)) return notFound('This package is not in the index.');
 const view = state.view?.key === 'all' ? state.view : drawGraph('all', graph, {
  overview: true,
  layout: (nodes, edges) => repositoryLayout(nodes, edges, packages),
  title: n => `${n.label} · ${n.symbol.file}:${n.symbol.line}`,
 });
 Object.assign(view, {
  related: id => view.overview.index.adjacent.get(id), edgeOn: touches,
  select: id => go(href.all(id), true),
  open: id => go(href.pdg(id)),
 });
 setView(view, route.fn);
 canvas.classList.add('whole-map');
 canvas.setAttribute('tabindex', '0'); canvas.setAttribute('aria-describedby', 'overview-help');
 canvas.classList.toggle('all-calls', state.allCalls);
 for (const el of canvas.querySelectorAll('.package-frame')) el.classList.toggle('selected', el.dataset.package === (symbol?.package || route.pkg));
 if (route.pkg && !route.fn) {
  const frame = view.layout.frames.find(p => p.id === route.pkg);
  const k = Math.min(1, (canvas.clientWidth - 80) / frame.w, (canvas.clientHeight - 120) / frame.h);
  moveCamera({k, x: frame.x + frame.w / 2 - canvas.clientWidth / 2 / k, y: frame.y + frame.h / 2 - canvas.clientHeight / 2 / k});
 }
 toolbar(`<div class="view-label">All functions <span>${graph.nodes.length.toLocaleString()} · ${plural(graph.edges.length, 'call link')}</span></div>${repositorySwitcher(true)}<div class="seg"><button id="all-calls" aria-pressed="${state.allCalls}" title="Show or hide all resolved call links. Selected calls stay visible.">All calls: ${state.allCalls ? 'on' : 'off'}</button></div>`);
 $('#all-calls').onclick = () => {
  state.allCalls = !state.allCalls; canvas.classList.toggle('all-calls', state.allCalls);
  $('#all-calls').setAttribute('aria-pressed', state.allCalls); $('#all-calls').textContent = `All calls: ${state.allCalls ? 'on' : 'off'}`;
  allCallsLegend(); applyCamera();
 };
 allCallsLegend();
 crumbs([[model.module, href.repo()], ['all functions', symbol || route.pkg ? href.all() : null], ...(symbol ? [[symbol.name, null]] : route.pkg ? [[shortPackage(route.pkg, model.module), null]] : [])]);
 inspector(symbol ? functionPanel(symbol) : route.pkg ? packagePanel(route.pkg, true) : repoPanel(packageGraph(model)));
}

function allCallsLegend() {
 legend(`<span>Every tile is one function</span><span>${state.allCalls ? 'All resolved call links' : 'Selected calls only'}</span><span>Zoom to read names · select to trace calls</span>`);
}

function showPackage(route) {
 const model = state.model, pkg = model.packages.get(route.pkg);
 if (!pkg) return notFound(`No package ${route.pkg} in this index.`);
 if (route.fn && model.symbols.get(route.fn)?.package !== pkg.id) return notFound('This function is not in the selected package.');
 const full = callGraph(model, pkg.id), weight = new Map(), functions = model.byPackage.get(pkg.id) || [];
 for (const e of full.edges) for (const id of [e.source, e.target]) weight.set(id, (weight.get(id) || 0) + e.count);
 if (!route.fn && functions.length > 80) {
  const exported = functions.filter(s => /^[A-Z]/.test(s.name));
  const first = functions.find(s => s.role === 'entrypoint') || [...(exported.length ? exported : functions)].sort((a, b) => (weight.get(b.id) || 0) - (weight.get(a.id) || 0))[0];
  return go(href.pkg(pkg.id, first.id), true);
 }
 const visible = route.fn && state.callScope === 'neighbors' ? neighbors(full, route.fn) : new Set(full.nodes.map(n => n.id));
 const graph = {nodes: full.nodes.filter(n => visible.has(n.id)), edges: full.edges.filter(e => visible.has(e.source) && visible.has(e.target))};
 const visibleWeight = new Map();
 for (const e of graph.edges) for (const id of [e.source, e.target]) visibleWeight.set(id, (visibleWeight.get(id) || 0) + e.count);
 const key = 'pkg:' + pkg.id + ':' + (state.callScope === 'neighbors' ? route.fn || 'all' : 'all');
 const view = drawGraph(key, graph, {
  size: n => n.kind === 'function' ? {w: clamp(32 + Math.max(textWidth(n.label, 12.5, true), textWidth(fileLine(n.symbol), 11)), 130, 340), h: 44} : {w: clamp(50 + textWidth(n.label, 12), 140, 300), h: 44},
  nodeClass: n => n.kind === 'function' ? 'fn' + (n.symbol.role ? ' entry' : '') : 'stub',
  node: n => n.kind === 'function'
   ? `<text class="name" x="14" y="19">${esc(clip(n.label, 44))}</text><text class="meta" x="14" y="35">${esc(fileLine(n.symbol))}</text>`
   : `<circle cx="16" cy="16" r="3.5" style="fill:${layerVar(n.layer)}"/><text class="name" x="27" y="20">${esc(clip(n.label, 38))}</text><text class="meta" x="27" y="36">${n.kind === 'caller-package' ? 'calls in' : 'called'} · ${plural(visibleWeight.get(n.id) || 0, 'call')}</text>`,
  edgeWidth: e => 1 + Math.min(2, Math.log2(e.count) * .4),
  edgeLabel: e => e.count > 1 ? `${e.count}×` : '',
  trayLabel: 'No resolved calls to or from these functions within the repository',
 });
 Object.assign(view, {
  related: id => neighbors(graph, id), edgeOn: touches,
  select: id => {
   const node = graph.nodes.find(n => n.id === id);
   if (node && node.kind !== 'function') return go(href.pkg(node.package));
   go(href.pkg(pkg.id, id), true);
  },
  open: id => graph.nodes.find(n => n.id === id)?.kind === 'function' && go(href.pdg(id)),
 });
 setView(view, route.fn);
 legend(`<span>calls flow left → right</span><span><i class="dot" style="--c:transparent;border:1px dashed var(--ink-3)"></i>other packages</span><span>double-click a function for its dependence graph</span>`);
 const shown = graph.nodes.filter(n => n.kind === 'function').length;
 toolbar(`<div class="view-label">Call graph <span>${shown} of ${plural(functions.length, 'function')}</span></div>${route.fn ? `<div class="seg"><button data-scope="neighbors" aria-pressed="${state.callScope === 'neighbors'}">Nearby calls</button><button data-scope="all" aria-pressed="${state.callScope === 'all'}">All functions</button></div>` : ''}`);
 crumbs([[model.module, href.repo()], [shortPackage(pkg.id, model.module), route.fn ? href.pkg(pkg.id) : null], ...(route.fn ? [[model.symbols.get(route.fn)?.name || route.fn, null]] : [])]);
 const symbol = model.symbols.get(route.fn);
 inspector(symbol ? functionPanel(symbol) : packagePanel(pkg.id, false));
}
const fileLine = s => `${base(s.file)}:${s.line}`;

async function showPDG(route) {
 const doc = await loadPDG(route.fn);
 if (!doc || route !== state.route) return;
 const {detail, kinds, mode} = state.pdg;
 const graph = pdgView(doc, detail);
 graph.edges = graph.edges.filter(e => kinds.has(e.kind));
 const key = `pdg:${route.fn}:${detail}:${[...kinds]}`;
 const size = n => ({w: clamp(28 + textWidth(clip(n.code, 54), 12, true), 120, 430), h: 46});
 const view = drawGraph(key, graph, {
  layout: (nodes, edges) => dependenceLayout(nodes, edges, size),
  size,
  rankGap: 90,
  radius: (n, p) => n.kind === 'param' || n.kind === 'result' ? p.h / 2 : 7,
  nodeClass: n => [n.category, n.kind].join(' '),
  node: n => `<text class="kind" x="14" y="17">${n.line ? 'L' + n.line + ' · ' : ''}${esc(n.kind)}</text><text class="code" x="14" y="34">${esc(clip(n.code, 54))}</text>`,
  edgeKind: e => e.kind,
  edgeLabel: e => e.symbols.join(', '),
  trayLabel: 'No dependencies recorded for these operations',
 });
 Object.assign(view, {
  doc, related: id => trace(graph.edges, id, state.pdg.mode, state.pdg.kinds),
  edgeOn: (e, center, related) => related.has(e.source) && related.has(e.target) && (state.pdg.mode !== 'direct' || touches(e, center)),
  select: id => { state.selected = id; paint(); inspector(pdgPanel(doc, graph, id)); if (id) centerOn(id); },
 });
 const keep = state.view?.key?.startsWith(`pdg:${route.fn}:`) ? state.selected : null;
 setView(view, view.nodeEls.has(keep) ? keep : null, !keep);
 legend(Object.entries(KINDS).filter(([k]) => kinds.has(k)).map(([k, text]) => `<span><i class="dot" style="--c:var(--${k})"></i>${k} · ${text}</span>`).join(''));
 toolbar(pdgToolbar());
 const symbol = state.model?.symbols.get(route.fn), name = pdgName(doc);
 crumbs(standalone ? [['export', null], [name, null]] : [[state.model.module, href.repo()], [shortPackage(symbol.package, state.model.module), href.pkg(symbol.package)], [symbol.name, href.pkg(symbol.package, symbol.id)], ['dependence graph', null]]);
 inspector(pdgPanel(doc, graph, state.selected));
}
const pdgName = doc => fact(doc.function?.qualified_name)?.split('/').at(-1) || fact(doc.function?.name) || doc.function?.id || 'function';
function pdgToolbar() {
 const {detail, kinds, mode} = state.pdg;
 const seg = (name, options, current) => `<div class="seg" role="group">${options.map(([value, label]) => `<button data-${name}="${value}" aria-pressed="${current === value}">${label}</button>`).join('')}</div>`;
 return seg('detail', [['statements', 'Statements'], ['expressions', 'Expressions']], detail)
  + `<div class="seg" role="group">${Object.keys(KINDS).map(k => `<button data-kind="${k}" aria-pressed="${kinds.has(k)}"><i class="dot" style="--c:var(--${k})"></i>${k}</button>`).join('')}</div>`
  + seg('mode', [['direct', 'Neighbors'], ['upstream', 'Upstream'], ['downstream', 'Downstream']], mode);
}
$('#toolbar').addEventListener('click', e => {
 const b = e.target.closest('button[data-scope],button[data-detail],button[data-kind],button[data-mode]');
 if (!b) return;
 if (b.dataset.scope) { state.callScope = b.dataset.scope; route(); return; }
 const p = state.pdg;
 if (b.dataset.detail) { p.detail = b.dataset.detail; state.selected = null; }
 if (b.dataset.kind) p.kinds.has(b.dataset.kind) ? p.kinds.delete(b.dataset.kind) : p.kinds.add(b.dataset.kind);
 if (b.dataset.mode) { p.mode = b.dataset.mode; $('#toolbar').innerHTML = pdgToolbar(); paint(); inspector(pdgPanel(state.view.doc, state.view.graph, state.selected)); return; }
 route();
});

async function loadPDG(id) {
 if (standalone) return embedded[Number(id)] || notFound('This export has no function at that position.');
 if (state.docs.has(id)) return state.docs.get(id);
 const symbol = state.model.symbols.get(id);
 if (!symbol) return notFound('This function is not in the index.');
 canvas.innerHTML = ''; state.view = null; state.selected = null;
 toolbar(''); legend('');
 crumbs([[state.model.module, href.repo()], [symbol.name, href.pkg(symbol.package, id)], ['dependence graph', null]]);
 overlay(`<div class="notice"><h3>Building the dependence graph</h3><p>Type-checking <code>${esc(shortPackage(symbol.package, state.model.module))}</code> to extract <code>${esc(symbol.name)}</code>. The first graph from a package takes a few seconds.</p><div class="progress"></div></div>`);
 inspector(functionPanel(symbol));
 try {
  if (!state.pendingDocs.has(id)) state.pendingDocs.set(id, fetch('/api/pdg?id=' + encodeURIComponent(id)));
  const response = await state.pendingDocs.get(id);
  const copy = response.clone();
  if (!copy.ok) throw new Error(copy.status === 503 ? 'Another graph is still building. Try again in a moment.' : (await copy.text()).trim());
  const [doc] = await copy.json();
  state.docs.set(id, doc);
  if (state.route.view === 'pdg' && state.route.fn === id) overlay('');
  return doc;
 } catch (error) {
  if (state.route.view !== 'pdg' || state.route.fn !== id) return null;
  overlay(`<div class="notice"><h3>No dependence graph for ${esc(symbol.name)}</h3><p>${esc(error.message)}</p><div class="actions"><button class="primary" id="retry">Try again</button><a class="secondary" href="${href.pkg(symbol.package, id)}">Back to package</a></div></div>`);
  $('#retry').onclick = () => route();
  return null;
 } finally { state.pendingDocs.delete(id); }
}

function setView(view, selected, refit = true) {
 const changed = state.view?.key !== view.key;
 state.view = view; state.selected = selected ?? null; state.hover = null;
 paint();
 if (changed && refit) fit(false);
 if (state.selected) centerOn(state.selected);
}
function notFound(message) { state.view?.overview?.detach(); canvas.innerHTML = ''; state.view = null; toolbar(''); legend(''); inspector(''); overlay(`<div class="notice"><h3>Not found</h3><p>${esc(message)}</p><div class="actions"><a class="secondary" href="#/">Back to ${standalone ? 'export' : 'repository'}</a></div></div>`); return null; }
const overlay = html => $('#overlay').innerHTML = html;
const legend = html => $('#legend').innerHTML = html;
const toolbar = html => $('#toolbar').innerHTML = html;
const inspector = html => { $('#inspector').innerHTML = html; $('#inspector .ln.sel')?.scrollIntoView({block: 'center'}); };
function crumbs(parts) {
 $('#crumbs').innerHTML = parts.map(([label, link], i) => (i ? '<i>/</i>' : '') + (link ? `<a href="${link}">${esc(label)}</a>` : `<span class="${i === parts.length - 1 ? 'here' : ''}">${esc(label)}</span>`)).join('');
}

// ── inspector panels ───────────────────────────────────────────────────────
function repoPanel(graph) {
 const all = state.route.view === 'all';
 const {index, module} = state.model, layers = LAYERS.filter(l => graph.nodes.some(n => n.layer === l));
 const typed = index.symbols.filter(s => s.call_resolution === 'typed').length;
 const resolution = `${typed.toLocaleString()} / ${index.symbols.length.toLocaleString()} functions type-checked. Other calls are unavailable.`;
 return `<section><h2 class="sans">${esc(base(module))}</h2><div class="sub mono">${esc(module)}</div>
  <dl class="facts"><dt>Packages</dt><dd>${index.packages.length}</dd><dt>Functions</dt><dd>${index.symbols.length.toLocaleString()}</dd><dt>Go files</dt><dd>${index.packages.reduce((n, p) => n + p.files, 0)}</dd><dt>Lines</dt><dd>${index.lines.toLocaleString()}</dd><dt>Calls</dt><dd>${resolution}</dd></dl><p class="note">Production Go only. Excludes tests, testdata, dist, nested modules, vendor, and dependencies. ${all ? 'Each tile is a function; frames group packages. Zoom to read names.' : 'Each box is a package; open it to see the functions inside.'}</p></section>
  <section><h3>Layers</h3><div class="list">${layers.map(l => { const pk = graph.nodes.filter(n => n.layer === l); return `<div class="link"><i class="dot" style="--c:${layerVar(l)}"></i><span>${l}</span><small>${plural(pk.length, 'package')} · ${pk.reduce((s, n) => s + n.functions, 0).toLocaleString()} fn</small></div>`; }).join('')}</div>
  <p class="note">${all ? 'Search or select a function to trace its calls. Every function stays on the map.' : 'Select a package to trace its imports. Open it to explore function calls.'}</p></section>
  <section><h3>Indexed checkout</h3><div class="sub mono">${esc(index.root)}</div></section>
  <section><h3>Read the map</h3><p class="note">Columns group packages by their path. Position does not imply execution order.</p><p class="note">Call graphs show static links. Interface dispatch, function values, tests, and callers in other modules can add calls. A missing link is not an unused-code finding.</p></section>`;
}
function packagePanel(id, fromRepo) {
 const model = state.model, p = model.packages.get(id), functions = model.byPackage.get(id) || [];
 const internal = p.imports.filter(i => model.packages.has(i)), external = p.imports.filter(i => !model.packages.has(i));
 const importers = model.index.packages.filter(q => q.imports.includes(id));
 const pkgLink = q => `<a class="link" href="${href.pkg(q)}"><i class="dot" style="--c:${layerVar(layerOf(q, model.module))}"></i><code>${esc(shortPackage(q, model.module))}</code></a>`;
 return `<section><h2 class="sans">${esc(shortPackage(id, model.module))}</h2><div class="sub mono">${esc(id)}</div>
  <dl class="facts"><dt>Layer</dt><dd><i class="dot" style="--c:${layerVar(layerOf(id, model.module))};display:inline-block;margin-right:6px"></i>${layerOf(id, model.module)}</dd><dt>Files</dt><dd>${p.files}</dd><dt>Lines</dt><dd>${p.lines.toLocaleString()}</dd><dt>Functions</dt><dd>${functions.length}</dd></dl>
  ${fromRepo ? `<div class="actions"><a class="primary" href="${href.pkg(id)}">Open call graph</a></div>` : ''}</section>
  <section><h3>Imports <span>${internal.length}</span></h3><div class="list">${internal.map(pkgLink).join('') || '<span class="empty">No packages from this repository.</span>'}</div>
  ${external.length ? `<p class="note">Also: ${external.map(esc).join(', ')}</p>` : ''}</section>
  <section><h3>Imported by <span>${importers.length}</span></h3><div class="list">${importers.map(q => pkgLink(q.id)).join('') || '<span class="empty">No package in this repository imports it.</span>'}</div></section>
  ${fromRepo ? '' : `<section><h3>Functions <span>${functions.length}</span></h3><div class="list">${functions.map(s => `<a class="link" href="${href.pkg(id, s.id)}"><code>${esc(s.name)}</code><small>${esc(fileLine(s))}</small></a>`).join('')}</div></section>`}`;
}
function functionPanel(s) {
 const model = state.model, callers = [...(model.callers.get(s.id) || [])].map(([id, count]) => [model.symbols.get(id), count]);
 const calls = new Map(), external = new Map(), dynamic = new Set();
 for (const c of s.calls) {
  if (model.symbols.has(c.target)) calls.set(c.target, (calls.get(c.target) || 0) + 1);
  else if (c.kind === 'dynamic') dynamic.add(c.spelling);
  else if (c.kind === 'external') { const pkg = c.package || '?'; if (!external.has(pkg)) external.set(pkg, new Set()); external.get(pkg).add(c.spelling.split('.').at(-1)); }
 }
 const fnLink = (t, count) => `<a class="link" href="${state.route.view === 'all' ? href.all(t.id) : href.pkg(t.package, t.id)}"><code>${esc(t.name)}</code><small>${t.package === s.package ? '' : esc(shortPackage(t.package, model.module)) + ' · '}${esc(fileLine(t))}${count > 1 ? ' · ' + count + '×' : ''}</small></a>`;
 const source = state.sources.get(s.id);
 if (source === undefined) loadSource(s);
 return `<section><h2>${esc(s.name)}</h2><div class="sub mono">${esc(s.file)}:${s.line}–${s.end_line}</div>
  ${s.call_resolution !== 'typed' ? '<p class="note">Calls unavailable: this declaration is outside the active build or its package has type errors.</p>' : ''}
  <div class="actions"><a class="primary" href="${href.pdg(s.id)}">Dependence graph</a></div></section>
  <section id="source">${source === undefined ? '<p class="empty">Loading source…</p>' : source == null ? '<p class="empty">Source is unavailable. Restart the atlas after changing files.</p>' : functionSource(s)}</section>
  <section><h3>Called by <span>${callers.length}</span></h3><div class="list">${callers.map(([t, n]) => fnLink(t, n)).join('') || `<span class="empty">${s.role ? 'Started by the Go runtime.' : 'No calls found in this repository. It may be called through an interface, by tests, or by another module.'}</span>`}</div></section>
  <section><h3>Calls <span>${calls.size}</span></h3><div class="list">${[...calls].map(([id, n]) => fnLink(model.symbols.get(id), n)).join('') || '<span class="empty">No functions from this repository.</span>'}</div>
  ${external.size ? `<h3 style="margin-top:16px">Outside this repository</h3><div class="chips">${[...external].map(([pkg, names]) => `<span class="chip"><b>${esc(base(pkg))}</b> ${esc([...names].join(', '))}</span>`).join('')}</div>` : ''}
  ${dynamic.size ? `<h3 style="margin-top:16px">Dynamic <span>interface or function value</span></h3><div class="chips">${[...dynamic].map(d => `<span class="chip">${esc(clip(d, 48))}</span>`).join('')}</div>` : ''}</section>`;
}
async function loadSource(s) {
 try {
  const response = await fetch('/api/source?id=' + encodeURIComponent(s.id));
  state.sources.set(s.id, response.ok ? (await response.json()).content : null);
 } catch { state.sources.set(s.id, null); }
 const box = $('#source');
 if (box && state.route.fn === s.id) box.innerHTML = state.sources.get(s.id) == null ? '<p class="empty">Source changed on disk since indexing. Restart the atlas to refresh it.</p>' : functionSource(s);
}
function functionSource(s) {
 if(!state.sourceBlocks.has(s.id))state.sourceBlocks.set(s.id,sourceBlock(state.sources.get(s.id),s.line));
 return state.sourceBlocks.get(s.id);
}
function sourceBlock(text, firstLine, marks = new Map()) {
 return `<div class="source">${highlightGo(text).map((html, i) => { const n = firstLine + i, m = marks.get(n) || ''; return `<div class="ln ${m}" data-n="${n}"><pre>${html || ' '}</pre></div>`; }).join('')}</div>`;
}
function pdgPanel(doc, graph, selected) {
 const src = doc.sources?.find(s => s.id === doc.function?.source_span?.source_id) || doc.sources?.[0];
 const node = graph.nodes.find(n => n.id === selected);
 let source = '<p class="empty">Source text is not included. Export with <code>-source-root</code> to read code here.</p>';
 if (typeof src?.content === 'string') {
  const text = sourceText(src.content), span = doc.function?.source_span || {start: 0, end: text.bytes.length};
  const first = text.line(span.start), last = text.line(Math.max(span.start, span.end - 1));
  const marks = new Map(), related = node ? trace(graph.edges, node.id, state.pdg.mode, state.pdg.kinds) : new Set();
  for (const n of graph.nodes) if (n.line) marks.set(n.line, (marks.get(n.line) || 'hit') + (related.has(n.id) && n !== node ? ' rel' : ''));
  if (node?.span) { const end = node.category === 'control' ? headerEnd(text, node.span.start, node.span.end) : node.span.end; for (let l = node.line; l <= text.line(Math.max(node.span.start, end - 1)); l++) marks.set(l, (marks.get(l) || '') + ' sel'); }
  source = sourceBlock(text.slice(text.starts[first - 1], text.starts[last] ?? text.bytes.length).replace(/\n$/, ''), first, marks);
 }
 const symbol = state.model?.symbols.get(state.route.fn);
 return `<section><h2>${esc(pdgName(doc))}</h2><div class="sub mono">${esc(src?.path || '')}</div>
  <dl class="facts"><dt>Operations</dt><dd>${doc.nodes.length} · shown as ${plural(graph.nodes.length, state.pdg.detail === 'statements' ? 'statement' : 'expression')}</dd><dt>Dependencies</dt><dd>${graph.edges.length} shown · ${doc.edges.length} recorded</dd></dl><p class="note">Dependencies are approximate. Missing links do not prove independence. Open the extraction limits below for details.</p>
  ${symbol ? `<div class="actions"><a class="secondary" href="${href.pkg(symbol.package, symbol.id)}">← Call graph</a></div>` : ''}</section>
  ${node ? operationSection(node, graph) : `<section><p class="note" style="margin:0">Click a statement in the graph or the code to trace what it depends on and what it affects.</p></section>`}
  <section id="pdg-source">${source}</section>
  <section>${evidenceDetails(doc)}</section>`;
}
function operationSection(node, graph) {
 const label = id => graph.nodes.find(n => n.id === id);
 const row = (e, other) => { const o = label(other); return `<button class="link" data-select="${esc(other)}"><span class="tag" style="--c:var(--${e.kind})">${e.kind}</span><code>${esc(clip(o.code, 46))}</code><small>${e.symbols.length ? esc(e.symbols.join(', ')) + ' · ' : ''}${o.line ? 'L' + o.line : o.kind}</small></button>`; };
 const incoming = graph.edges.filter(e => e.target === node.id && e.source !== node.id), outgoing = graph.edges.filter(e => e.source === node.id && e.target !== node.id);
 return `<section><div class="sub mono" style="margin:0 0 4px">${node.line ? 'L' + node.line + ' · ' : ''}${esc(node.kind)}${node.members.length > 1 ? ` · ${node.members.length} operations` : ''}</div><h2 style="font-size:14px;font-weight:500">${esc(node.code)}</h2></section>
  <section><h3>Depends on <span>${incoming.length}</span></h3><div class="list">${incoming.map(e => row(e, e.source)).join('') || '<span class="empty">Nothing, among the enabled edge kinds.</span>'}</div>
  <h3 style="margin-top:16px">Affects <span>${outgoing.length}</span></h3><div class="list">${outgoing.map(e => row(e, e.target)).join('') || '<span class="empty">Nothing, among the enabled edge kinds.</span>'}</div></section>`;
}
function evidenceDetails(doc) {
 const caps = Object.entries(doc.extraction_evidence?.capabilities || {});
 if (!caps.length) return '<p class="empty">The export records no extraction limits.</p>';
 return `<details><summary>What this graph does and does not model</summary>${caps.map(([k, v]) => `<div class="cap"><b>${esc(k.replaceAll('_', ' '))}</b><span class="status ${esc(v.status)}">${esc(v.status)}</span><span></span><span>${esc(v.description || '')}</span></div>`).join('')}</details>`;
}
$('#inspector').addEventListener('click', e => {
 const target = e.target.closest('[data-select]');
 if (target) { state.view.select(target.dataset.select); return; }
 const line = e.target.closest('#pdg-source .ln.hit');
 if (!line) return;
 const hits = state.view.graph.nodes.filter(n => n.line === Number(line.dataset.n));
 const next = hits[(hits.findIndex(n => n.id === state.selected) + 1) % hits.length];
 if (next) state.view.select(next.id);
});

// ── Go highlighting ────────────────────────────────────────────────────────
const GO_TOKENS = /(\/\/[^\n]*|\/\*[\s\S]*?\*\/)|("(?:[^"\\\n]|\\.)*"|`[^`]*`|'(?:[^'\\\n]|\\.)*')|\b(break|case|chan|const|continue|default|defer|else|fallthrough|for|func|go|goto|if|import|interface|map|package|range|return|select|struct|switch|type|var)\b|\b(true|false|nil|iota)\b|\b(0[xX][\da-fA-F_]+|\d[\d_]*(?:\.\d+)?(?:[eE][+-]?\d+)?)\b|\b(string|int|int8|int16|int32|int64|uint|uint8|uint16|uint32|uint64|uintptr|float32|float64|bool|byte|rune|error|any)\b|\b([A-Za-z_]\w*)(?=\()/g;
const TOKEN_CLASS = [, 'c', 's', 'k', 'b', 'n', 't', 'f'];
function highlightGo(text) {
 const lines = [''];
 const push = (chunk, cls) => chunk.split('\n').forEach((part, i) => { if (i) lines.push(''); if (part) lines[lines.length - 1] += cls ? `<span class="tk-${cls}">${esc(part)}</span>` : esc(part); });
 let last = 0;
 for (const m of text.matchAll(GO_TOKENS)) {
  push(text.slice(last, m.index));
  push(m[0], TOKEN_CLASS[m.findIndex((g, i) => i && g !== undefined)]);
  last = m.index + m[0].length;
 }
 push(text.slice(last));
 return lines;
}

let treeKey = null, treeVirtual = null, treeScrollFrame = 0;
const functionRowHeight = 28;
function syncTreeWindow(focus) {
 const data=treeVirtual, tree=$('#tree');
 if(!data?.node.isConnected)return;
 const top=data.node.offsetTop, height=tree.clientHeight;
 if(focus&&data.indices.has(focus)){
  const y=top+data.indices.get(focus)*functionRowHeight;
  if(y<tree.scrollTop||y+functionRowHeight>tree.scrollTop+height)tree.scrollTop=Math.max(0,y-height/2);
 }
 const first=clamp(Math.floor((tree.scrollTop-top)/functionRowHeight)-8,0,data.list.length);
 const last=clamp(Math.ceil((tree.scrollTop-top+height)/functionRowHeight)+8,first,data.list.length);
 const range=`${first}:${last}`;
 if(range!==data.range){
  data.range=range;data.window.style.top=first*functionRowHeight+'px';
  data.window.innerHTML=data.list.slice(first,last).map((s,i)=>`<div role="listitem" aria-posinset="${first+i+1}" aria-setsize="${data.list.length}">${data.row(s)}</div>`).join('');
 }
 const active=tree.querySelector('.row.fn.on'), next=state.route.fn?data.window.querySelector(`a[data-function="${encodeURIComponent(state.route.fn)}"]`):null;
 if(active!==next){active?.classList.remove('on');next?.classList.add('on');}
}
$('#tree').addEventListener('scroll',()=>{
 if(treeScrollFrame||!treeVirtual)return;
 treeScrollFrame=requestAnimationFrame(()=>{treeScrollFrame=0;syncTreeWindow();});
});
new ResizeObserver(()=>syncTreeWindow(state.route?.fn)).observe($('#tree'));
// ── sidebar ────────────────────────────────────────────────────────────────
function mark(text, query) {
 const i = query ? text.toLowerCase().indexOf(query) : -1;
 return i < 0 ? esc(text) : esc(text.slice(0, i)) + '<mark>' + esc(text.slice(i, i + query.length)) + '</mark>' + esc(text.slice(i + query.length));
}
function functionRows(list,row) {
 if(list.length<=200)return list.map(s=>row(s)).join('');
 return `<div class="virtual-functions" role="list" aria-label="${list.length} functions" style="height:${list.length*functionRowHeight}px"><div class="virtual-window"></div></div>`;
}
function renderTree() {
 const query = $('#search').value.trim().toLowerCase(), route = state.route;
 if (standalone) {
  $('#tree').innerHTML = `<div class="group">Exported functions</div>` + embedded.map((doc, i) => [doc, i]).filter(([doc]) => pdgName(doc).toLowerCase().includes(query)).map(([doc, i]) => `<a class="row fn${route.fn === String(i) ? ' on' : ''}" href="${href.pdg(String(i))}"><span class="name">${mark(pdgName(doc), query)}</span><span class="count">${doc.nodes.length}</span></a>`).join('');
  return;
 }
 const model = state.model, current = route.pkg || model.symbols.get(route.fn)?.package;
 const key = `${route.view}:${current || ''}:${query}`;
 if (treeKey === key) {
  if(treeVirtual){syncTreeWindow(route.fn);return;}
  const active = $('#tree .row.fn.on');
  const url = route.view === 'all' ? href.all(route.fn) : href.pkg(current, route.fn);
  const next = route.fn ? $('#tree').querySelector(`a[href="${url}"]`) : null;
  if (active !== next) { active?.classList.remove('on'); next?.classList.add('on'); next?.scrollIntoView({block:'nearest'}); }
  return;
 }
 treeKey = key;treeVirtual=null;

 const pkgRow = p => `<a class="row${p.id === current ? ' on' : ''}" href="${route.view === 'all' ? href.all(null, p.id) : href.pkg(p.id)}"><i class="dot" style="--c:${layerVar(layerOf(p.id, model.module))}"></i><span class="name">${mark(shortPackage(p.id, model.module), query)}</span><span class="count">${model.byPackage.get(p.id)?.length || 0}</span></a>`;
 const fnRow = (s, extra = '') => `<a class="row fn${s.id === state.route.fn ? ' on' : ''}" href="${route.view === 'all' ? href.all(s.id) : href.pkg(s.package, s.id)}" data-function="${encodeURIComponent(s.id)}"><span class="name">${mark(s.name, query)}</span><span class="count">${extra}</span></a>`;
 if (query) {
  const packages = model.index.packages.filter(p => shortPackage(p.id, model.module).toLowerCase().includes(query));
  const functions = model.index.symbols.filter(s => `${s.name} ${s.file}`.toLowerCase().includes(query));
  $('#tree').innerHTML = (packages.length ? `<div class="group">Packages</div>${packages.slice(0, 40).map(pkgRow).join('')}` : '')
   + (functions.length ? `<div class="group">Functions <span style="margin-left:auto">${functions.length}</span></div>${functions.slice(0, 120).map(s => fnRow(s, esc(base(shortPackage(s.package, model.module))))).join('')}` : '')
   + (packages.length + functions.length ? '' : `<p class="hint">Nothing matches “${esc(query)}”.</p>`);
  return;
 }
 $('#tree').innerHTML = LAYERS.map(layer => {
  const packages = model.index.packages.filter(p => layerOf(p.id, model.module) === layer);
  if (!packages.length) return '';
  return `<div class="group"><i class="dot" style="--c:${layerVar(layer)}"></i>${layer}</div>` + packages.map(p => pkgRow(p) + (p.id === current ? functionRows(model.byPackage.get(p.id) || [], fnRow) : '')).join('');
 }).join('');
 const block=$('#tree .virtual-functions');
 if(block){
  const list=model.byPackage.get(current)||[];
  treeVirtual={node:block,window:block.firstElementChild,list,row:fnRow,indices:new Map(list.map((s,i)=>[s.id,i])),range:null};
  syncTreeWindow(route.fn);
 }else $('#tree .row.fn.on, #tree .row.on')?.scrollIntoView({block:'nearest'});
}
$('#search').addEventListener('input', renderTree);
$('#search').addEventListener('keydown', e => { if (e.key === 'Enter') $('#tree a')?.click(); if (e.key === 'Escape') { e.target.value = ''; renderTree(); e.target.blur(); } });

// ── keyboard ───────────────────────────────────────────────────────────────
addEventListener('keydown', e => {
 if (e.target.closest('input, textarea, select, [contenteditable=true]')) return;
 if (e.key === 'Enter' && e.target.closest('button, a, summary')) return;
 if (e.key === '/') { e.preventDefault(); $('#search').focus(); }
 if (e.key === 'f') fit();
 if (e.key === '+' || e.key === '=') zoomAt(1.3);
 if (e.key === '-') zoomAt(1 / 1.3);
 if (e.key === 'Enter' && state.selected) state.view.open?.(state.selected);
 if (e.key === 'Escape') {
  const r = state.route;
  if (state.selected) state.view.select(null);
  else if (r.view === 'pdg' && !standalone) go(href.pkg(state.model.symbols.get(r.fn).package, r.fn));
  else if (r.view === 'all' && r.pkg) go(href.all());
  else if (r.view === 'package') go(href.repo());
 }
});

// ── boot ───────────────────────────────────────────────────────────────────
function route() {
 if (!standalone && !state.model) return;
 state.route = parseRoute();
 overlay('');
 canvas.classList.remove('package-map', 'all-imports', 'whole-map', 'all-calls');
 canvas.removeAttribute('tabindex'); canvas.removeAttribute('aria-describedby'); canvas.removeAttribute('aria-activedescendant');
 canvas.style.cursor = ''; labelScale = null;
 const r = state.route;
 if (r.view !== 'all') state.view?.overview?.detach();
 if (r.view === 'invalid') { notFound('This link has an invalid address.'); return; }
 if (r.view === 'all') showAll(r); else if (r.view === 'pdg') showPDG(r); else if (r.view === 'package') showPackage(r); else showRepo(r);
 renderTree();
}
addEventListener('hashchange', route);
(async () => {
 if (standalone) { $('#stats').textContent = plural(embedded.length, 'function'); route(); return; }
 try {
  const response = await fetch('/api/index', {cache:'no-store'});
  if (!response.ok) throw new Error(await response.text());
  state.model = indexModel(await response.json());
  const {index} = state.model;
  $('#stats').textContent = `${index.packages.length} packages · ${index.symbols.length.toLocaleString()} functions · ${index.lines.toLocaleString()} lines`;
  route();
 } catch (error) { overlay(`<div class="notice"><h3>The repository index did not load</h3><p>${esc(error.message)}</p></div>`); }
})();
