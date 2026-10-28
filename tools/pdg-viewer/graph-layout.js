// Layered (Sugiyama) layout, left to right. Reversing DFS back edges breaks
// cycles for ranking only; routes keep each edge's real direction. Long edges
// pass through virtual points and run straight across the columns they cross.
// Nodes without edges pack into a tray below. Ranks describe layout, not time.
// The repository map gives each architectural layer a stable section. Import
// links do not move packages when selection changes.
function packageLayout(nodes, edges) {
 const columns = [['cmd', 'internal', 'tools', 'root'], ['application'], ['adapters'], ['domain']];
 const positions = new Map(), sections = [];
 let column = 0, height = 0;
 for (const layers of columns) {
  let y = 44, used = false;
  for (const layer of layers) {
   const group = nodes.filter(n => n.layer === layer);
   for (let start = 0; start < group.length; start += 12) {
    if (start) { column++; y = 44; }
    const chunk = group.slice(start, start + 12), x = 32 + column * 220, top = y;
    y += 30;
    for (const n of chunk) {
     positions.set(n.id, {x: x + 92, y: y + 23, w: 184, h: 46});
     y += 54;
    }
    sections.push({layer, x, y: top, w: 184, h: y - top, count: group.length, continued: start > 0});
    y += 22; used = true;
    height = Math.max(height, y);
   }
  }
  if (used) column++;
 }
 const routes = edges.map(e => {
  const a = positions.get(e.source), b = positions.get(e.target);
  if (!a || !b) return null;
  if (a.x === b.x) {
   const x = a.x + a.w / 2, bend = x + 20;
   return [{x, y: a.y}, {x: bend, y: a.y}, {x: bend, y: b.y}, {x, y: b.y}];
  }
  const sign = Math.sign(b.x - a.x);
  return [{x: a.x + sign * a.w / 2, y: a.y}, {x: b.x - sign * b.w / 2, y: b.y}];
 });
 return {positions, routes, sections, width: column ? 64 + column * 220 - 36 : 200, height};
}

// Transpose the same dependency layout to read from top to bottom. Node text
// keeps its normal dimensions; only ranking and routes change orientation.
function dependenceLayout(nodes, edges, size) {
 const layout = graphLayout(nodes, edges, {size: n => {const s = size(n);return {w: s.h, h: s.w};}, rankGap: 54, nodeGap: 28});
 const positions = new Map([...layout.positions].map(([id, p]) => [id, {...p, x: p.y, y: p.x, w: p.h, h: p.w}]));
 const routes = layout.routes.map(points => points?.map(p => ({x: p.y, y: p.x})) || null);
 return {...layout, positions, routes, tray: null, width: layout.height, height: layout.width, direction: 'down'};
}

// Expand package groups into individual function tiles. Architectural columns
// stay stable regardless of calls, selection, search, or disconnected nodes.
function repositoryLayout(nodes, edges, packages) {
 const columns = [['cmd', 'internal', 'tools', 'root'], ['application'], ['adapters'], ['domain']];
 const byPackage = new Map(packages.map(p => [p.id, []]));
 for (const n of nodes) byPackage.get(n.package).push(n);
 const positions = new Map(), frames = [], sections = [];
 let x = 48, height = 0;
 for (const layers of columns) {
  const own = packages.filter(p => layers.includes(p.layer));
  if (!own.length) continue;
  const count = own.reduce((sum, p) => sum + byPackage.get(p.id).length, 0);
  const cols = Math.max(2, Math.min(18, Math.ceil(Math.sqrt(count) / 3)));
  const width = cols * 132 + 32;
  let y = 160;
  for (const layer of layers) {
   const group = own.filter(p => p.layer === layer);
   if (!group.length) continue;
   sections.push({layer, x, y: y - 80, w: width, count: group.length});
   for (const p of group) {
    const functions = byPackage.get(p.id), top = y;
    functions.sort((a, b) => a.symbol.file.localeCompare(b.symbol.file) || a.symbol.start - b.symbol.start);
    y += 88;
    functions.forEach((n, i) => positions.set(n.id, {x: x + 16 + (i % cols) * 132 + 60, y: y + Math.floor(i / cols) * 28 + 10, w: 120, h: 20}));
    y += Math.max(1, Math.ceil(functions.length / cols)) * 28 + 24;
    frames.push({id: p.id, label: p.label, layer, count: functions.length, x, y: top, w: width, h: y - top});
    y += 24;
   }
   y += 120;
  }
  x += width + 160;
  height = Math.max(height, y);
 }
 const routes = edges.map(e => {
  const a = positions.get(e.source), b = positions.get(e.target);
  if (a === b) return null;
  const sign = b.x >= a.x ? 1 : -1;
  return [{x: a.x + sign * a.w / 2, y: a.y}, {x: b.x - sign * b.w / 2, y: b.y}];
 });
 return {positions, routes, frames, sections, width: Math.max(200, x - 112), height: Math.max(200, height)};
}

function graphLayout(nodes, edges, options = {}) {
 const size = options.size || (() => ({w: 180, h: 40}));
 const rankGap = options.rankGap ?? 110, nodeGap = options.nodeGap ?? 16, pad = options.pad ?? 48;
 const index = new Map(nodes.map((n, i) => [n.id, i]));
 const links = [];
 edges.forEach((e, i) => { const a = index.get(e.source), b = index.get(e.target); if (a !== undefined && b !== undefined && a !== b) links.push({i, a, b}); });
 const connected = new Uint8Array(nodes.length);
 for (const l of links) connected[l.a] = connected[l.b] = 1;
 markBackEdges(nodes.length, links, connected);
 const rank = rankNodes(nodes.length, links, connected);
 const group = components(nodes.length, links);
 const items = nodes.map((n, i) => ({node: n, rank: rank[i], group: group[i], seq: i, ...size(n)}));
 const chains = routeChains(items, links);
 const layers = orderLayers(items, connected);
 const columns = placeColumns(layers, rankGap, pad);
 placeRows(layers, nodeGap);
 const bottom = normalize(layers, pad);
 const positions = new Map();
 items.forEach((it, i) => { if (connected[i]) positions.set(it.node.id, {x: columns.center[it.rank], y: it.y, w: it.w, h: it.h, rank: it.rank}); });
 const routes = edges.map(() => null);
 for (const l of links) routes[l.i] = edgePoints(chains.get(l.key), columns, l.reversed);
 const tray = packTray(nodes.filter((_, i) => !connected[i]), size, positions, Math.max(columns.width, 960), bottom, pad);
 return {positions, routes, tray, width: Math.max(columns.width, tray ? tray.x + tray.w + pad : 0), height: tray ? tray.y + tray.h + pad : bottom + pad};
}
// Iterative DFS so 10k-deep chains stay off the JavaScript call stack.
function markBackEdges(count, links, connected) {
 const out = Array.from({length: count}, () => []), state = new Uint8Array(count);
 for (const l of links) out[l.a].push(l);
 for (let start = 0; start < count; start++) {
  if (state[start] || !connected[start]) continue;
  const stack = [[start, 0]]; state[start] = 1;
  while (stack.length) {
   const top = stack[stack.length - 1], v = top[0];
   if (top[1] === out[v].length) { state[v] = 2; stack.pop(); continue; }
   const l = out[v][top[1]++];
   if (state[l.b] === 1) l.reversed = true;
   else if (!state[l.b]) { state[l.b] = 1; stack.push([l.b, 0]); }
  }
 }
 for (const l of links) { [l.u, l.v] = l.reversed ? [l.b, l.a] : [l.a, l.b]; l.key = l.u + ':' + l.v; }
}
// Longest path from sources, then sources move right next to their nearest successor.
function rankNodes(count, links, connected) {
 const succ = Array.from({length: count}, () => []), indegree = new Array(count).fill(0), rank = new Array(count).fill(0);
 for (const l of links) { succ[l.u].push(l.v); indegree[l.v]++; }
 const queue = [];
 for (let i = 0; i < count; i++) if (connected[i] && !indegree[i]) queue.push(i);
 const sources = queue.length;
 for (let q = 0; q < queue.length; q++) for (const v of succ[queue[q]]) { rank[v] = Math.max(rank[v], rank[queue[q]] + 1); if (--indegree[v] === 0) queue.push(v); }
 for (let q = 0; q < sources; q++) { const u = queue[q]; rank[u] = succ[u].reduce((m, v) => Math.min(m, rank[v]), Infinity) - 1; }
 return rank;
}
// Connected components keep their own vertical band; each takes its first node's index.
function components(count, links) {
 const parent = Array.from({length: count}, (_, i) => i);
 const find = i => { while (parent[i] !== i) i = parent[i] = parent[parent[i]]; return i; };
 for (const l of links) { const a = find(l.a), b = find(l.b); if (a !== b) parent[Math.max(a, b)] = Math.min(a, b); }
 return parent.map((_, i) => find(i));
}
// One chain of items per distinct ranked pair: [source, ...virtual, target].
function routeChains(items, links) {
 const chains = new Map();
 for (const l of links) {
  if (chains.has(l.key)) continue;
  const chain = [items[l.u]];
  for (let r = items[l.u].rank + 1; r < items[l.v].rank; r++) { const virtual = {rank: r, w: 0, h: 0, virtual: true, group: items[l.u].group, seq: items.length}; items.push(virtual); chain.push(virtual); }
  chain.push(items[l.v]);
  for (let k = 1; k < chain.length; k++) { (chain[k - 1].down ||= []).push(chain[k]); (chain[k].up ||= []).push(chain[k - 1]); }
  chains.set(l.key, chain);
 }
 return chains;
}
// Barycenter sweeps reduce crossings; input order breaks ties deterministically.
function orderLayers(items, connected) {
 const layers = [];
 items.forEach((it, i) => { if (it.virtual || connected[i]) (layers[it.rank] ||= []).push(it); });
 for (let r = 0; r < layers.length; r++) (layers[r] ||= []).sort((a, b) => a.group - b.group || a.seq - b.seq).forEach((it, i) => it.order = i);
 for (let sweep = 0; sweep < 8; sweep++) {
  const down = sweep % 2 === 0, range = down ? layers.map((_, r) => r).slice(1) : layers.map((_, r) => r).reverse().slice(1);
  for (const r of range) {
   const score = it => { const near = (down ? it.up : it.down) || []; return near.length ? near.reduce((s, n) => s + n.order, 0) / near.length : it.order; };
   layers[r].forEach(it => it.score = score(it));
   layers[r].sort((a, b) => a.score - b.score || a.order - b.order).forEach((it, i) => it.order = i);
  }
 }
 return layers;
}
function placeColumns(layers, rankGap, pad) {
 const center = [], left = [], right = [];
 let x = pad;
 layers.forEach((layer, r) => { const w = layer.reduce((m, it) => Math.max(m, it.w), 0); left[r] = x; center[r] = x + w / 2; right[r] = x + w; x += w + rankGap; });
 return {center, left, right, width: layers.length ? x - rankGap + pad : 0};
}
// Each node seeks the mean height of its neighbors without breaking layer order.
function placeRows(layers, nodeGap) {
 const gap = (a, b) => (a.h + b.h) / 2 + (a.virtual || b.virtual ? 6 : nodeGap);
 for (const layer of layers) { let y = 0; layer.forEach((it, i) => { y += i ? gap(layer[i - 1], it) : 0; it.y = y; }); const shift = y / 2; layer.forEach(it => it.y -= shift); }
 for (let pass = 0; pass < 8; pass++) {
  const down = pass % 2 === 0, order = down ? layers : [...layers].reverse();
  for (const layer of order) {
   const desired = layer.map(it => { const near = (down ? it.up : it.down) || []; return near.length ? near.reduce((s, n) => s + n.y, 0) / near.length : it.y; });
   settle(layer, desired, gap);
  }
 }
}
function settle(layer, desired, gap) {
 const n = layer.length, forward = [], backward = [];
 for (let i = 0; i < n; i++) forward[i] = i ? Math.max(desired[i], forward[i - 1] + gap(layer[i - 1], layer[i])) : desired[i];
 for (let i = n - 1; i >= 0; i--) backward[i] = i < n - 1 ? Math.min(desired[i], backward[i + 1] - gap(layer[i], layer[i + 1])) : desired[i];
 for (let i = 0; i < n; i++) layer[i].y = Math.max((forward[i] + backward[i]) / 2, i ? layer[i - 1].y + gap(layer[i - 1], layer[i]) : -Infinity);
}
function normalize(layers, pad) {
 let top = Infinity, bottom = -Infinity;
 for (const layer of layers) for (const it of layer) { top = Math.min(top, it.y - it.h / 2); bottom = Math.max(bottom, it.y + it.h / 2); }
 if (top === Infinity) return 0;
 for (const layer of layers) for (const it of layer) it.y += pad - top;
 return bottom - top + pad;
}
function edgePoints(chain, columns, reversed) {
 const [first, last] = [chain[0], chain[chain.length - 1]];
 const points = [{x: columns.center[first.rank] + first.w / 2, y: first.y}];
 for (const it of chain.slice(1, -1)) points.push({x: columns.left[it.rank], y: it.y}, {x: columns.right[it.rank], y: it.y});
 points.push({x: columns.center[last.rank] - last.w / 2, y: last.y});
 return reversed ? points.reverse() : points;
}
function packTray(isolated, size, positions, width, bottom, pad) {
 if (!isolated.length) return null;
 const sizes = isolated.map(size), cell = Math.max(...sizes.map(s => s.w)) + 16, rowH = Math.max(...sizes.map(s => s.h)) + 12;
 const columns = Math.max(1, Math.floor((width - pad * 2 + 16) / cell)), top = positions.size ? bottom + 72 : pad + 28;
 isolated.forEach((n, i) => positions.set(n.id, {x: pad + (i % columns) * cell + sizes[i].w / 2, y: top + Math.floor(i / columns) * rowH + sizes[i].h / 2, ...sizes[i]}));
 return {x: pad, y: top, w: Math.min(isolated.length, columns) * cell - 16, h: Math.ceil(isolated.length / columns) * rowH - 12};
}
// Smooth path through route points: horizontal runs stay straight, gaps curve.
function routePath(points, vertical = false) {
 let d = `M${points[0].x},${points[0].y}`;
 for (let i = 1; i < points.length; i++) {
  const p = points[i - 1], q = points[i], dx = (q.x - p.x) / 2;
  if (vertical) {
   const dy = (q.y - p.y) / 2;
   d += Math.abs(q.x - p.x) < .5 ? ` L${q.x},${q.y}` : ` C${p.x},${p.y + dy} ${q.x},${q.y - dy} ${q.x},${q.y}`;
   continue;
  }
  d += Math.abs(q.y - p.y) < .5 ? ` L${q.x},${q.y}` : ` C${p.x + dx},${p.y} ${q.x - dx},${q.y} ${q.x},${q.y}`;
 }
 return d;
}
