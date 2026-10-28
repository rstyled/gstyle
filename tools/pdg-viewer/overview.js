// Geometry lookup stays separate from rendering. The full graph remains loaded;
// spatial buckets select only the vector detail needed for the current viewport.
function overviewIndex(graph, layout, cell = 256) {
 const buckets = new Map(), nodes = new Map(), incident = new Map(), adjacent = new Map();
 const key = (x, y) => `${x}:${y}`;
 for (const n of graph.nodes) {
  const p = layout.positions.get(n.id);
  nodes.set(n.id, n); incident.set(n.id, []); adjacent.set(n.id, new Set([n.id]));
  for (let x = Math.floor((p.x - p.w / 2) / cell); x <= Math.floor((p.x + p.w / 2) / cell); x++) {
   for (let y = Math.floor((p.y - p.h / 2) / cell); y <= Math.floor((p.y + p.h / 2) / cell); y++) {
    const k = key(x, y);
    if (!buckets.has(k)) buckets.set(k, new Set());
    buckets.get(k).add(n.id);
   }
  }
 }
 graph.edges.forEach((e, i) => {
  incident.get(e.source).push(i);
  if (e.target !== e.source) incident.get(e.target).push(i);
  adjacent.get(e.source).add(e.target); adjacent.get(e.target).add(e.source);
 });
 const query = bounds => {
  const found = new Set();
  for (let x = Math.floor(bounds.x / cell); x <= Math.floor((bounds.x + bounds.w) / cell); x++) {
   for (let y = Math.floor(bounds.y / cell); y <= Math.floor((bounds.y + bounds.h) / cell); y++) {
    for (const id of buckets.get(key(x, y)) || []) {
     const p = layout.positions.get(id);
     if (p.x + p.w / 2 >= bounds.x && p.x - p.w / 2 <= bounds.x + bounds.w && p.y + p.h / 2 >= bounds.y && p.y - p.h / 2 <= bounds.y + bounds.h) found.add(id);
    }
   }
  }
  return found;
 };
 const hit = (x, y) => {
  const id = query({x, y, w: 0, h: 0}).values().next().value;
  if (id !== undefined) return {id};
  const frame = layout.frames.find(p => x >= p.x && x <= p.x + p.w && y >= p.y && y <= p.y + p.h);
  return frame ? {pkg: frame.id} : null;
 };
 return {nodes, incident, adjacent, query, hit};
}

// At most 2048² pixels per layer, independent of repository size or device DPR.
// SVG still supplies captions, crisp zoomed-in detail, and selected call paths.
function overviewRasters(graph, layout, paths) {
 const scratch = document.createElement('canvas'), scale = Math.min(1, 2048 / Math.max(layout.width, layout.height));
 scratch.width = Math.ceil(layout.width * scale); scratch.height = Math.ceil(layout.height * scale);
 const ctx = scratch.getContext('2d'), colors = getComputedStyle(document.documentElement);
 ctx.scale(scale, scale);
 ctx.strokeStyle = colors.getPropertyValue('--edge').trim(); ctx.lineWidth = 1;
 ctx.stroke(new Path2D(paths.join(' ')));
 const calls = scratch.toDataURL();
 ctx.clearRect(0, 0, layout.width, layout.height);
 const mix = (a, b, weight) => {
  const rgb = hex => hex.trim().replace('#', '').match(/../g).map(s => parseInt(s, 16));
  const left = rgb(a), right = rgb(b);
  return `rgb(${left.map((v, i) => Math.round(v * weight + right[i] * (1 - weight))).join(',')})`;
 };
 for (const layer of LAYERS) {
  const path = new Path2D(), tile = colors.getPropertyValue('--' + layer), raised = colors.getPropertyValue('--raised'), line = colors.getPropertyValue('--line');
  for (const n of graph.nodes) if (n.layer === layer) {
   const p = layout.positions.get(n.id); path.rect(p.x - p.w / 2, p.y - p.h / 2, p.w, p.h);
  }
  ctx.fillStyle = mix(tile, raised, .38); ctx.strokeStyle = mix(tile, line, .7);
  ctx.fill(path); ctx.stroke(path);
 }
 const tiles = scratch.toDataURL();
 scratch.width = scratch.height = 0;
 return {calls, tiles};
}

function drawOverview(key, graph, layout, spec) {
 const index = layout.overviewIndex ||= overviewIndex(graph, layout);
 const paths = layout.overviewPaths ||= layout.routes.map((route, i) => route ? routePath(route) : selfLoop(layout.positions.get(graph.edges[i].source)));
 const raster = layout.overviewRaster ||= overviewRasters(graph, layout, paths);
 const frames = layout.frames.map(p => `<g class="package-frame" data-package="${esc(p.id)}"><rect x="${p.x}" y="${p.y}" width="${p.w}" height="${p.h}" rx="10"/><title>${esc(p.label)} · ${p.count.toLocaleString()} functions</title><text x="${p.x + 16}" y="${p.y + 60}" data-label="${esc(p.label)} · ${p.count.toLocaleString()}" data-width="${p.w}">${esc(p.label)}</text></g>`).join('');
 const sections = layout.sections.map(s => `<g class="map-section"><text x="${s.x}" y="${s.y + 10}" style="fill:${layerVar(s.layer)}">${esc(s.layer)}</text></g>`).join('');
 canvas.innerHTML = `<title id="overview-hover-title"></title><defs><marker id="overview-arrow" viewBox="0 0 8 8" refX="7.5" refY="4" markerWidth="7" markerHeight="7" markerUnits="userSpaceOnUse" orient="auto"><path d="M0,.8 8,4 0,7.2z" fill="var(--ink-2)"/></marker></defs><desc id="overview-help">Every indexed function is on this map. Use search or the arrow keys to select a function. Press Enter to open its dependence graph.</desc>${frames}<image class="overview-calls" href="${raster.calls}" width="${layout.width}" height="${layout.height}" aria-hidden="true"/><image class="overview-tiles" href="${raster.tiles}" width="${layout.width}" height="${layout.height}" aria-hidden="true"/>${sections}<g class="overview-focus"></g><g class="overview-detail"></g><g id="overview-selection" role="button"></g>`;
 canvas.setAttribute('tabindex', '0'); canvas.setAttribute('aria-describedby', 'overview-help');
 const focus = canvas.querySelector('.overview-focus'), detail = canvas.querySelector('.overview-detail'), selection = canvas.querySelector('#overview-selection');
 let center = null, selected = null, detailKey = null;
 const cell = 512;
 const glyph = n => {
  const p = layout.positions.get(n.id), on = !center || index.adjacent.get(center)?.has(n.id);
  return `<g class="node whole-function layer-${n.layer}${on ? ' on' : ''}${n.id === selected ? ' selected' : ''}" data-id="${esc(n.id)}" transform="translate(${p.x - p.w / 2},${p.y - p.h / 2})"><rect class="box" width="${p.w}" height="${p.h}" rx="2"/>${spec.node(n)}<title>${esc(spec.title(n))}</title></g>`;
 };
 const camera = () => {
  if (cam.k < .5) {
   if (detailKey !== 'far') { detail.innerHTML = ''; detailKey = 'far'; }
   return;
  }
  const x = Math.floor(cam.x / cell) * cell, y = Math.floor(cam.y / cell) * cell;
  const w = Math.ceil((canvas.clientWidth / cam.k + cam.x - x) / cell) * cell, h = Math.ceil((canvas.clientHeight / cam.k + cam.y - y) / cell) * cell;
  const next = `${x}:${y}:${w}:${h}:${center}:${selected}`;
  if (next === detailKey) return;
  detailKey = next;
  detail.innerHTML = [...index.query({x, y, w, h})].map(id => glyph(index.nodes.get(id))).join('');
 };
 const highlight = (id, chosen) => {
  if (id === center && chosen === selected) return;
  center = id; selected = chosen;
  const related = index.adjacent.get(center) || new Set();
  let html = '';
  if (center) {
   html += index.incident.get(center).map(i => `<path class="edge on" d="${paths[i]}" marker-end="url(#overview-arrow)"/>`).join('');
   const byLayer = new Map();
   for (const id of related) {
    const n = index.nodes.get(id), p = layout.positions.get(id);
    byLayer.set(n.layer, (byLayer.get(n.layer) || '') + `M${p.x - p.w / 2},${p.y - p.h / 2}h${p.w}v${p.h}h-${p.w}z`);
   }
   html += [...byLayer].map(([layer, d]) => `<g class="whole-function layer-${layer}"><path class="box" d="${d}"/></g>`).join('');
  }
  if (selected) {
   const p = layout.positions.get(selected);
   html += `<rect x="${p.x - p.w / 2}" y="${p.y - p.h / 2}" width="${p.w}" height="${p.h}" rx="2" fill="none" stroke="var(--ink)" stroke-width="1.2" vector-effect="non-scaling-stroke"/>`;
  }
  focus.innerHTML = html;
  if (selected) {
   selection.setAttribute('aria-label', spec.title(index.nodes.get(selected)));
   const p = layout.positions.get(selected);
   selection.innerHTML = `<rect x="${p.x - p.w / 2}" y="${p.y - p.h / 2}" width="${p.w}" height="${p.h}" fill="transparent" pointer-events="none"/>`;
   canvas.setAttribute('aria-activedescendant', 'overview-selection');
  } else { selection.innerHTML = ''; selection.removeAttribute('aria-label'); canvas.removeAttribute('aria-activedescendant'); }
  detailKey = null; camera();
 };
 return {key, graph, layout, spec, nodeEls: new Map(), edgeEls: [], overview: {index, hit: index.hit, camera, highlight, rasterCount: graph.nodes.length}};
}
