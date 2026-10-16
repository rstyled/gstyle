const graphs = JSON.parse(document.querySelector('#pdg-data').textContent);
const $ = (s) => document.querySelector(s);
const esc = (s) => String(s ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
const fact = (f) => f?.status === 'known' ? String(f.value) : '';
const names = (g) => fact(g.function?.qualified_name) || fact(g.function?.name) || g.function?.id || 'Unnamed function';
const colors = {data:'#397c78', control:'#ad7143', execution:'#6c65a7'};
let g = graphs[0], selected = null, mode = 'direct', pageIndex = 0;
let P, sockets = new Map(), activeNodes = new Set(), angle = 45, targetAngle = 45, frame = 0;
const pageSize = 6;
const viewNodes = () => selected && $('#focus').checked ? g.nodes.filter(n=>activeNodes.has(n.id)) : g.nodes;
const enabled = () => new Set([...document.querySelectorAll('[data-kind]:checked')].map(e => e.dataset.kind));
const nodeLabel = n => {
 const callee=fact(n.attributes?.callee);

 const src=g.sources?.find(s=>s.id===n.source_span?.source_id);
 if(src?.content && n.source_span.end>n.source_span.start) {
  const snippet=utf8slice(src.content,n.source_span.start,n.source_span.end).trim().replace(/\s+/g,' ');
  if(snippet) return snippet.length>70 ? snippet.slice(0,67)+'…' : snippet;
 }
 const symbol=n.attributes?.symbol_ids?.[0];
 return callee || (symbol ? symbolName(symbol) : n.attributes?.description || n.category || fact(n.original_kind) || n.id);
};
const symbolName = id => { const symbol=g.symbols?.find(s=>s.id===id); return fact(symbol?.name)||id||'unresolved'; };
const accessNames = accesses => (accesses||[]).map(a=>symbolName(a.symbol_id)||a.memory_location_id||a.unresolved_reason||'memory').join(', ');
const sourceFor = n => g.sources?.find(s => s.id === n?.source_span?.source_id);
const utf8slice = (text, start, end) => new TextDecoder().decode(new TextEncoder().encode(text).slice(start, end));
function related() {
 if (!selected) return new Set();
 const seen = new Set([selected]), types = enabled();
 if (mode === 'direct') {
  for (const e of g.edges) if (types.has(e.kind) && (e.source === selected || e.target === selected)) { seen.add(e.source); seen.add(e.target); }
  return seen;
 }
 const adjacency = new Map();
 for (const e of g.edges) {
  if (!types.has(e.kind)) continue;
  const from = mode === 'upstream' ? e.target : e.source;
  const to = mode === 'upstream' ? e.source : e.target;
  if (!adjacency.has(from)) adjacency.set(from, []);
  adjacency.get(from).push(to);
 }
 const queue = [selected];
 for (let i = 0; i < queue.length; i++) for (const id of adjacency.get(queue[i]) || []) {
  if (seen.has(id)) continue;
  seen.add(id); queue.push(id);
 }
 return seen;
}
function selectNode(id) {
 selected = id || null;
 activeNodes = related();
 const index = viewNodes().findIndex(n => n.id === selected);
 if (index >= 0) pageIndex = Math.floor(index / pageSize);
 $('#node-list').value = selected || '';
 activeNodes = related(); inspect(); draw();
 if(selected){const p=lastLayout?.positions.get(selected);if(p)camera.focus(p.x,p.y);}
}
function showSource(n) {
 const src = sourceFor(n) || g.sources?.find(s => s.id === g.function?.source_span?.source_id);
 const text = src?.content;
 if (typeof text !== 'string') { $('#code').textContent = 'Source text is not included. Generate with -source-root /path/to/repo to show code.'; return; }
 const bounds = g.function?.source_span || {start:0,end:new TextEncoder().encode(text).length};
 const start = bounds.start, end = bounds.end;
 const span = n?.source_span;
 const valid = span && span.source_id === src.id && span.start >= start && span.end <= end;
 const bytes = new TextEncoder().encode(text);
 const slice = (a,b) => new TextDecoder().decode(bytes.slice(a,b));
 const spans = g.nodes.filter(v => v.source_span?.source_id === src.id && v.source_span.start >= start && v.source_span.end <= end && v.source_span.end > v.source_span.start);
 const cuts = [...new Set([start,end,...spans.flatMap(v=>[v.source_span.start,v.source_span.end])])].sort((a,b)=>a-b);
 const html = [];
 for(let i=0;i<cuts.length-1;i++) {
  const a=cuts[i], b=cuts[i+1];
  const owners=spans.filter(v=>v.source_span.start<=a&&v.source_span.end>=b).sort((x,y)=>(x.source_span.end-x.source_span.start)-(y.source_span.end-y.source_span.start));
  let part=esc(slice(a,b));
  if(owners.length) part=`<span data-source-node="${esc(owners[0].id)}" title="${esc(nodeLabel(owners[0]))}">${part}</span>`;
  if(valid && a>=span.start && b<=span.end) part='<mark>'+part+'</mark>';
  html.push(part);
 }
 $('#code').innerHTML=html.join('');
 const line = utf8slice(text,0,valid ? span.start : start).split('\n').length;
 $('#source-location').textContent = `${src.path} · line ${line} · byte offsets ${valid ? span.start : start}–${valid ? span.end : end}`;
 $('#code mark')?.scrollIntoView({block:'nearest'});
}
function inspect() {
 const n = g.nodes.find(n => n.id === selected);
 $('#node-title').textContent = n ? nodeLabel(n) : 'Choose a node';
 $('#raw').textContent = n ? JSON.stringify(n,null,2) : '';
 $('#links').replaceChildren(); showSource(n);
 $('#explanation').textContent=n ? `This ${n.category.replaceAll('_',' ')} reads ${accessNames(n.attributes?.reads)||'no recorded values'} and writes ${accessNames(n.attributes?.writes)||'no recorded values'}. Select a link below to follow where its inputs come from or where its result goes.` : 'Click code to see its recorded inputs and uses.';
 if (!n) { $('#detail').textContent = 'Select a socket, source code, or node from the list.'; return; }
 const incoming = g.edges.filter(e => e.target === n.id), outgoing = g.edges.filter(e => e.source === n.id);
 $('#detail').textContent = `${n.id}\n${n.category} · ${n.synthetic ? 'synthetic' : 'source'}\n${incoming.length} incoming / ${outgoing.length} outgoing\nEvidence: ${n.evidence?.status || 'not supplied'}\n${n.evidence?.description || ''}\n${activeNodes.size} nodes in ${mode} trace\nReads: ${accessNames(n.attributes?.reads)||'none recorded'}\nWrites: ${accessNames(n.attributes?.writes)||'none recorded'}`;
 for (const e of [...incoming,...outgoing]) {
  const id = e.source === n.id ? e.target : e.source;
  const other = g.nodes.find(v => v.id === id);
  const button = document.createElement('button');
  const forward=e.source===n.id;
  const relation={data:forward?'Result used by':'Gets value from',control:forward?'Controls':'Controlled by',execution:forward?'Must run before':'Must run after'}[e.kind];
  button.textContent = `${relation}${e.symbol_id ? ' ['+symbolName(e.symbol_id)+']' : ''} · ${nodeLabel(other || {id})}`;
  button.title = JSON.stringify(e,null,2); button.onclick = () => selectNode(id); $('#links').append(button);
 }
}
const camera=enableGraphCamera($('#stage'));
let graphSignature='',lastLayout=null;
const shortCode=n=>nodeLabel(n).replace(/\s+/g,' ').trim();
function diagramNode(n,p) {
 const src=sourceFor(n),line=src?.content&&n.source_span?utf8slice(src.content,0,n.source_span.start).split('\n').length:null;
 const kind=fact(n.original_kind).replace(/^\*?ast\./,'')||n.category;
 const code=shortCode(n),parts=[code.slice(0,25),code.length>25?code.slice(25,49)+(code.length>49?'…':''):''];
 const rectangular=['control','assignment','return','declaration','entry','exit'].includes(n.category);
 const shape=rectangular?`<rect x="${p.x-110}" y="${p.y-39}" width="220" height="78" rx="5"/>`:`<ellipse cx="${p.x}" cy="${p.y}" rx="110" ry="39"/>`;
 const classes=['graph-node',n.category==='control'?'control-node':'',['formal_input','formal_output'].includes(n.category)?'input-node':'',selected===n.id?'chosen':'',selected&&!activeNodes.has(n.id)?'muted':''].join(' ');
 return `<g class="${classes}" data-node="${esc(n.id)}" role="button" tabindex="0" aria-label="${esc(kind+': '+code)}" aria-pressed="${selected===n.id}"><title>${esc(n.id+' · '+kind+' · '+code)}</title>${shape}<text x="${p.x}" y="${p.y-18}" text-anchor="middle" class="node-kind">${esc(kind)}${line?' · L'+line:''}</text><text x="${p.x}" y="${p.y+1}" text-anchor="middle" class="node-code"><tspan x="${p.x}">${esc(parts[0])}</tspan><tspan x="${p.x}" dy="17">${esc(parts[1])}</tspan></text></g>`;
}
function diagramEdge(e,i,positions) {
 const a=positions.get(e.source),b=positions.get(e.target);if(!a||!b)return '';
 const relevant=!selected||(mode==='direct'?e.source===selected||e.target===selected:activeNodes.has(e.source)&&activeNodes.has(e.target));
 const label=e.kind+(e.symbol_id?' · '+symbolName(e.symbol_id):e.outcome?' · '+e.outcome:'');
 const dash={data:'6 4',control:'2 4',execution:''}[e.kind];
 return `<g class="${relevant?'selected':'muted'}"><path id="edge-${i}" d="${graphEdgePath(a,b,i)}" fill="none" stroke="${colors[e.kind]}" stroke-width="${selected&&relevant?2.4:1.2}" stroke-dasharray="${dash}" marker-end="url(#arrow-${e.kind})"><title>${esc(e.source+' → '+e.target+' / '+e.kind+' / '+e.subkind)}</title></path><text class="edge-text" dy="-4"><textPath href="#edge-${i}" startOffset="45%">${esc(label)}</textPath></text></g>`;
}
function draw() {
 const nodes=viewNodes(),types=enabled(),edges=g.edges.filter(e=>types.has(e.kind));
 const layout=graphLayout(nodes,edges),signature=nodes.map(n=>n.id).join('|')+g.function?.id+[...types].join(',');
 let html=`<defs>${Object.entries(colors).map(([kind,color])=>`<marker id="arrow-${kind}" viewBox="0 0 6 6" refX="5" refY="3" markerWidth="6" markerHeight="6" orient="auto"><path d="M0 0L6 3L0 6Z" fill="${color}"/></marker>`).join('')}</defs>`;
 html+=edges.map((e,i)=>diagramEdge(e,i,layout.positions)).join('');
 html+=nodes.map(n=>diagramNode(n,layout.positions.get(n.id))).join('');
 lastLayout=layout;
 $('#stage').innerHTML=html;
 if(signature!==graphSignature){camera.reset(layout.width,layout.height);graphSignature=signature;}
 $('#counts').textContent=`${g.nodes.length} NODES / ${g.edges.length} EDGES`;
 $('#readout').textContent=`${nodes.length} visible operations · ${edges.filter(e=>layout.positions.has(e.source)&&layout.positions.has(e.target)).length} enabled dependencies`;
 $('#page-label').textContent='Whole function';$('#prev').disabled=true;$('#next').disabled=true;
}
function load(index) {
 g=graphs[index]; selected=null; pageIndex=0; activeNodes=new Set();
 $('#source').textContent=g.sources?.map(s=>s.path).join(' · ') || 'No source path supplied';
 $('#node-list').innerHTML='<option value="">Choose a socket</option>'+g.nodes.map(n=>`<option value="${esc(n.id)}">${esc((g.nodes.indexOf(n)+1)+'. '+n.id+' · '+nodeLabel(n))}</option>`).join('');
 const evidence=g.extraction_evidence;
 const caps=evidence?.capabilities || {};
 $('#confidence').textContent = ['control_dependencies','scalar_reaching_definitions','memory_aliases'].map(key=>`${key.replaceAll('_',' ')}: ${caps[key]?.status||'not supplied'}`).join(' · ');
 $('#coverage').textContent=evidence ? JSON.stringify(evidence,null,2) : 'Extraction evidence is not supplied.';
 selectNode(null);
}
function filterFunctions() {
 const search=$('#search').value.toLowerCase();
 $('#functions').innerHTML=graphs.map((v,i)=>({v,i})).filter(({v})=>(names(v)+' '+(v.sources||[]).map(s=>s.path).join(' ')).toLowerCase().includes(search)).map(({v,i})=>`<option value="${i}">${esc(names(v))} · ${v.nodes.length} nodes</option>`).join('');
 $('#search-status').textContent = $('#functions').options.length ? `${$('#functions').options.length} functions` : 'No matching functions. The current graph remains visible.';
 if ($('#functions').value !== '') load(Number($('#functions').value));
}
$('#functions').onchange=e=>load(Number(e.target.value));
$('#search').oninput=filterFunctions;
$('#node-list').onchange=e=>selectNode(e.target.value);
$('#stage').onclick=e=>{const node=e.target.closest('[data-node]');if(node) selectNode(node.dataset.node);};
$('#stage').onkeydown=e=>{const node=e.target.closest('[data-node]');if(node && ['Enter',' '].includes(e.key)){e.preventDefault();selectNode(node.dataset.node); $('#node-list').focus();}};
$('#clear').onclick=()=>selectNode(null);
$('#trace').onchange=e=>{mode=e.target.value;activeNodes=related();pageIndex=0;inspect();draw();};
for(const checkbox of document.querySelectorAll('[data-kind]')) checkbox.onchange=()=>{activeNodes=related();pageIndex=0;inspect();draw();};
$('#prev').onclick=()=>{pageIndex--;draw();}; $('#next').onclick=()=>{pageIndex++;draw();};
$('#theme').onclick=()=>{document.body.classList.toggle('dark');draw();};
$('#zoom-in').onclick=()=>camera.zoom(1/1.25);
$('#zoom-out').onclick=()=>camera.zoom(1.25);
$('#fit').onclick=()=>camera.fit();
$('#fullscreen').onclick=()=>$('#stage').closest('.instrument').requestFullscreen();
filterFunctions();

$('#code').onclick=e=>{const node=e.target.closest('[data-source-node]');if(node)selectNode(node.dataset.sourceNode);};

$('#focus').onchange=()=>{pageIndex=0;draw();};
