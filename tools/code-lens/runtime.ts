import {explanationSteps} from './explanation';
import {investigationLayout,directedRoutes,routePath} from './directed-layout';
import {bytes,lineAt,lineNodes,snippet,validate,upstream,proofRegion,fitBox,zoomBox,resizeBox,type Graph,type Node,type Edge,type Box,type Witness} from './model';
const $=<T extends HTMLElement=HTMLElement>(id:string)=>document.getElementById(id) as T;
const svg=$('graph') as unknown as SVGSVGElement;
const mini=$('minimap') as unknown as SVGSVGElement;
const data=JSON.parse($('evidence').textContent!);
const g:Graph=data.graph,witness:Witness=data.detector.witness;
validate(g,data.source);
const nodes=new Map(g.nodes.map(n=>[n.id,n]));
const layout=investigationLayout(g,witness);
const positions:Map<string,{x:number;y:number;w:number;h:number}>=layout.positions;
const full:Box={x:0,y:0,w:layout.width,h:layout.height};
let box:Box={...full},whole:Box={...full};
let selected=witness.comparison,selectedEdge='',highlight:'none'|'proof'|'slice'='none';
let region=new Set<string>(),panel='',proofStep='',guideStep=-1;
const guide=explanationSteps(g,witness);
const enabled=new Set(['data','control','execution']);
const esc=(s:unknown)=>String(s).replaceAll('&','&amp;').replaceAll('<','&lt;').replaceAll('>','&gt;').replaceAll('"','&quot;');
const firstLine=lineAt(data.source,g.function.source_span.start);
const functionSource=bytes(data.source,g.function.source_span.start,g.function.source_span.end);
const sourceLines=functionSource.split('\n');
const line=(n:Node)=>n.source_span?lineAt(data.source,n.source_span.start):firstLine;
const short=(n:Node)=>n.category==='merge'?'merge(boxed)':snippet(data.source,n).split('\n')[0].trim().split('//')[0].trim();
const category=(n:Node)=>n.category.replaceAll('_',' ');
const proof=proofRegion(g,witness);
const seeds=new Set(witness.steps.filter(s=>s.seed).map(s=>s.node));
const routes=directedRoutes(g,witness,layout);
const semantic=(n:Node)=>({merge:'Join both reaching values',control:'Choose a branch',read:'Read the binding',assignment:'Assign the value',computation:'Compute the expression',declaration:'Declare the initial value',formal_input:'Receive the input'}[n.category]||category(n));
const title=(n:Node)=>short(n).length>26?short(n).slice(0,25)+'…':short(n);

function graphMarkup() {
  const edges=g.edges.map((e,i)=>{
    const a=positions.get(e.source)!,b=positions.get(e.target)!;
    const points=routes.get(e.id)!,path=routePath(points);
    const segments=points.slice(1).map((p,j)=>({a:points[j],b:p,length:Math.abs(p.x-points[j].x)+Math.abs(p.y-points[j].y)})).sort((a,b)=>b.length-a.length);
    const middle=segments[0];
    const label=e.kind==='control'&&seeds.has(e.target)?(e.operand_position===1?'false':'true'):proof.edges.has(e.id)?(seeds.has(e.source)?'non-nil':e.subkind==='reaching-definition'?'boxed':'value'):'';
    const labelX=(middle.a.x+middle.b.x)/2+(middle.a.x===middle.b.x?8:0),labelY=(middle.a.y+middle.b.y)/2-7;
    return `<g data-edge-group="${e.id}" data-kind="${e.kind}"><path class="edge-hit" data-edge="${e.id}" d="${path}"/><path class="edge ${e.kind}" data-edge="${e.id}" d="${path}" marker-end="url(#arrow-${e.kind})"><title>${esc(e.id+' · '+e.source+' → '+e.target+' · '+e.subkind+' · '+e.evidence.status)}</title></path><text class="edge-label" x="${labelX}" y="${labelY}" text-anchor="${middle.a.x===middle.b.x?'start':'middle'}">${label}</text></g>`;
  }).join('');
  const items=g.nodes.map(n=>{
    const p=positions.get(n.id)!;
    return `<g class="graph-node ${n.category==='control'?'control-node':''} ${n.synthetic?'synthetic-node':''}" data-node="${n.id}" transform="translate(${p.x} ${p.y})" tabindex="0" role="button" aria-label="${esc(n.id+' '+category(n)+' source line '+line(n)+': '+short(n))}"><title>${esc(short(n))}</title><rect x="-110" y="-44" width="220" height="88" rx="8"/><text class="node-id" x="-94" y="-25">${n.id}</text><text class="node-meta" x="-63" y="-25">${esc(category(n))} · L${line(n)}</text><text class="node-role" x="-94" y="0">${esc(layout.roles.get(n.id))}</text><text class="node-label" x="-94" y="24">${esc(title(n))}</text></g>`;
  }).join('');
  const defs=['data','control','execution'].map(k=>`<marker id="arrow-${k}" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse"><path d="M0 0L10 5L0 10" fill="var(--${k==='data'?'accent':k})"/></marker>`).join('');
  return `<defs>${defs}</defs><g class="layout-guides"><rect x="10" y="205" width="245" height="465" rx="14"/><text x="28" y="232">SUPPORTING OPERATIONS</text><text x="325" y="210">TRUE BRANCH</text><text x="870" y="210">FALSE BRANCH</text><text x="560" y="545">BOTH PATHS REACH THE JOIN</text></g><g id="edges">${edges}</g><g id="nodes">${items}</g>`;
}
svg.innerHTML=graphMarkup();
mini.setAttribute('viewBox',`0 0 ${layout.width} ${layout.height}`);
mini.innerHTML=g.edges.map(e=>{const a=positions.get(e.source)!,b=positions.get(e.target)!;return `<path class="map-edge" d="M${a.x} ${a.y}L${b.x} ${b.y}"/>`;}).join('')+g.nodes.map(n=>{const p=positions.get(n.id)!;return `<rect data-map-node="${n.id}" class="map-node" x="${p.x-65}" y="${p.y-22}" width="130" height="44" rx="7"/>`;}).join('')+'<rect class="map-window" id="map-window"/>';
let lastAspect=1.5;
function aspect() {
  const r=svg.getBoundingClientRect();
  if(r.width>0&&r.height>0)lastAspect=r.width/r.height;
  return lastAspect;
}
function camera() {
  svg.setAttribute('viewBox',`${box.x} ${box.y} ${box.w} ${box.h}`);
  const scale=svg.getBoundingClientRect().width/box.w;
  svg.classList.toggle('overview',scale<.48);
  $('zoom-label').textContent=`${Math.round(whole.w/box.w*100)}%`;
  const view=$('map-window');
  for(const key of ['x','y','w','h'] as const)view.setAttribute(key==='w'?'width':key==='h'?'height':key,String(box[key]));
}
function fitAll() {whole=fitBox(full,aspect(),30);box={...whole};camera();}
function wholeGraph() {guideStep=-1;highlight='none';region.clear();update();guideReadout();fitAll();}
function bounds(ids:Set<string>):Box {
  const ps=[...ids].map(id=>positions.get(id)!).filter(Boolean);
  const xs=ps.map(p=>p.x),ys=ps.map(p=>p.y);
  return {x:Math.min(...xs)-120,y:Math.min(...ys)-54,w:Math.max(...xs)-Math.min(...xs)+240,h:Math.max(...ys)-Math.min(...ys)+108};
}
function focus(ids:Set<string>) {box=fitBox(bounds(ids),aspect(),70);camera();}
function zoom(factor:number,x=.5,y=.5) {
  if(box.w*factor<260||box.w*factor>whole.w*4)return;
  box=zoomBox(box,factor,x,y);camera();
}
function refreshGraph() {
    svg.querySelectorAll<SVGGElement>('.graph-node').forEach(el=>{
    const id=el.dataset.node!;
    el.classList.toggle('active',id===selected);
    el.classList.toggle('proof-node',highlight==='proof'&&proof.ids.has(id));
    el.classList.toggle('context',highlight!=='none'&&!region.has(id)&&id!==selected);
  });
  svg.querySelectorAll<SVGGElement>('[data-edge-group]').forEach(el=>{
    const e=g.edges.find(e=>e.id===el.dataset.edgeGroup)!;
    el.style.display=enabled.has(e.kind)?'':'none';
    el.classList.toggle('context-edge',guideStep>=0&&!(region.has(e.source)&&region.has(e.target)));
    const edge=el.querySelector('.edge')!;
    edge.classList.toggle('active',e.id===selectedEdge||(highlight==='none'&&(e.source===selected||e.target===selected)));
    edge.classList.toggle('proof',proof.edges.has(e.id));
    edge.classList.toggle('supporting-edge',!proof.edges.has(e.id)&&!(e.kind==='control'&&seeds.has(e.target)));
    edge.classList.toggle('context',highlight!=='none'&&!(highlight==='proof'?proof.edges.has(e.id):region.has(e.source)&&region.has(e.target)));
  });
  mini.querySelectorAll<SVGElement>('[data-map-node]').forEach(el=>el.classList.toggle('selected',el.dataset.mapNode===selected));
  const visible=g.edges.filter(e=>enabled.has(e.kind)).length;
  $('counts').textContent=`${g.nodes.length} NODES · ${visible}/${g.edges.length} EDGES`;
  $('graph-state').textContent=highlight==='proof'?'DETECTOR PROOF / CONTEXT RETAINED':highlight==='slice'?'BACKWARD VALUE SLICE / CONTEXT RETAINED':'FULL EXPORT / ALL NODES RETAINED';
  $('selection-readout').textContent=`${selected} · source L${line(nodes.get(selected)!)}`;
}
function sourceMarkup() {
  let offset=g.function.source_span.start;
  const n=nodes.get(selected)!,s=n.source_span;
  const proofLines=new Set([...(guideStep>=0?region:proof.ids)].map(id=>line(nodes.get(id)!)));
  return sourceLines.map((text,i)=>{
    const start=offset,end=start+new TextEncoder().encode(text).length;
    offset=end+1;
    let code=esc(text);
    if(s&&s.start<end&&s.end>start){
      const a=Math.max(start,s.start),b=Math.min(end,s.end);
      code=esc(bytes(data.source,start,a))+`<mark>${esc(bytes(data.source,a,b))}</mark>`+esc(bytes(data.source,b,end));
    }
    return `<div class="source-row ${line(n)===firstLine+i?'active':''} ${(highlight==='proof'||guideStep>=0)&&proofLines.has(firstLine+i)?'proof':''}" data-line="${firstLine+i}" tabindex="0" role="button" aria-label="Source line ${firstLine+i}: ${esc(text.trim())}"><span class="ln">${firstLine+i}</span><code>${code}</code></div>`;
  }).join('');
}
function edgeDescription(e:Edge) {
  if(e.kind==='control')return `Lexical ${e.operand_position===1?'false':'true'} branch`;
  if(e.kind==='execution')return 'Recorded order constraint';
  return e.subkind==='reaching-definition'?'Reaching definition':e.subkind==='binding-value'?'Value supplied to operation':e.subkind;
}
function nodeExplanation(n:Node) {
  if(n.id===witness.comparison)return 'This is the comparison reported by the actual detector. Its interface operand is proven non-nil through the solver witness; therefore == nil is false.';
  if(n.category==='merge')return 'An analysis operation joins the recorded reaching definitions of boxed. Both branch assignments must be non-nil for this join to become known. This is not a Go statement.';
  if(n.category==='declaration')return 'boxed starts nil here. Neither branch leaves that value unchanged, so this declaration does not reach the later comparison.';
  const step=witness.steps.find(s=>s.node===n.id);
  if(step?.seed)return 'The detector marks this concrete interface value as a non-nil seed. It does not need this expression’s syntax operands to establish non-nil.';
  if(step)return `The detector propagates non-nil from ${step.dependencies.join(', ')} into this operation. Select “Detector proof” to inspect its exact transfer rule.`;
  if(n.category==='control')return 'This branch selects which assignment runs. Its control edges provide navigation context; the non-nil detector does not consume them as proof edges.';
  return n.evidence.description;
}
function inspect() {
  const n=nodes.get(selected)!;
  $('source').innerHTML=sourceMarkup();
  $('source-status').textContent=`L${line(n)} · ${n.source_span?.start}–${n.source_span?.end} bytes`;
  $('detail-label').textContent=highlight==='proof'&&!proof.ids.has(selected)?'SELECTED CONTEXT / NOT IN PROOF':'SELECTED OPERATION';
  $('title').textContent=semantic(n);
  $('badge').textContent=`${n.id} · ${category(n)} · ${n.synthetic?'synthetic':'source'} · L${line(n)}`;
  $('snippet').textContent=snippet(data.source,n);
  $('explanation').textContent=guideStep>=0?guide[guideStep].body:nodeExplanation(n);
  const edges=g.edges.filter(e=>e.source===selected||e.target===selected);
  $('connections-title').textContent=`${edges.length} direct connections · select twice to follow`;
  $('connections').innerHTML=edges.map(e=>{
    const other=nodes.get(e.source===selected?e.target:e.source)!;
    return `<button class="connection ${selectedEdge===e.id?'active':''}" data-connection="${e.id}"><b>${esc(e.source+' → '+e.target+' · '+short(other))}</b><small>${esc(edgeDescription(e))} · ${esc(e.evidence.status)}${enabled.has(e.kind)?'':' · hidden by filter'}</small></button>`;
  }).join('')||'<span class="muted">No direct exported connections.</span>';
  const e=g.edges.find(e=>e.id===selectedEdge);
  $('edge-evidence').textContent=e?e.evidence.description:'Select a connection to read its evidence. Selection survives zooming and view changes.';
  $('raw').textContent=JSON.stringify(e?{edge:e,source:nodes.get(e.source),target:nodes.get(e.target)}:n,null,2);
  $('node-select').value=selected;
}
function update() {refreshGraph();inspect();}
function select(id:string,reveal=false) {
  if(!nodes.has(id))return;
  if(guideStep>=0){guideStep=-1;highlight='none';region.clear();guideReadout();}
  selected=id;selectedEdge='';update();
  if(reveal)focus(new Set([id]));
}
function chooseEdge(id:string) {
  const e=g.edges.find(e=>e.id===id)!;
  if(selectedEdge===id){select(e.source===selected?e.target:e.source,true);return;}
  if(e.source!==selected&&e.target!==selected)selected=e.target;
  selectedEdge=id;update();
}
function showPanel(name:string) {
  panel=panel===name?'':name;
  for(const id of ['proof','runs'])$(id+'-panel').classList.toggle('visible',panel===id);
  document.querySelectorAll<HTMLButtonElement>('[data-panel]').forEach(b=>b.setAttribute('aria-pressed',String(b.dataset.panel===panel)));
}
function guideReadout() {
  document.body.classList.toggle('guided',guideStep>=0);
  const step=guide[guideStep];
  $('guide-title').textContent=step?.title||'Can boxed be nil at the final check?';
  $('guide-body').textContent=step?.body||'Both branches give boxed a value. Follow these four steps to see why the nil check is always false.';
  document.querySelectorAll<HTMLButtonElement>('[data-guide-step]').forEach(b=>b.setAttribute('aria-pressed',String(Number(b.dataset.guideStep)===guideStep)));
  ($('guide-prev') as HTMLButtonElement).disabled=guideStep<=0;
  ($('guide-next') as HTMLButtonElement).disabled=guideStep===guide.length-1;
  $('guide-next').textContent=guideStep<0?'Start →':'Next →';
}
function startGuide(index:number) {
  if(index<0||index>=guide.length)return;
  guideStep=index;const step=guide[index];
  highlight='slice';region=new Set(step.nodes);selected=step.selected;selectedEdge='';
  update();guideReadout();focus(region);
  $('graph-state').textContent=`EXPLANATION / STEP ${index+1} OF ${guide.length}`;
}
function finding() {startGuide(0);}
function detectorRule(n:Node) {
  const step=witness.steps.find(s=>s.node===n.id);
  if(n.id===witness.comparison)return {name:'operandKnown',reason:'The comparison is reported only when a matching supported operand exists and every matching source is known.'};
  if(step?.seed)return {name:'operation',reason:'The concrete/boxed interface-value annotation seeds this node as non-nil.'};
  if(n.category==='merge')return {name:'solve',reason:'The worklist marks the join known only after all recorded dependencies have become known.'};
  return {name:'transfer',reason:'Only eligible transfer categories consume supported proof dependencies. The witness records the actual dependencies used.'};
}
function methodSource(name:string) {
  const source=data.detector.source as string;
  const match=new RegExp('^func (?:\\([^)]*\\) )?'+name+'\\(', 'm').exec(source);
  if(!match)throw new Error('Detector rule is missing: '+name);
  const start=match.index;
  const next=source.indexOf('\nfunc ',start+1);
  return {text:source.slice(start,next<0?source.length:next),line:source.slice(0,start).split('\n').length};
}
function showProofStep(id:string,reveal=true) {
  guideStep=-1;guideReadout();
  proofStep=id;
  if(highlight!=='proof'){highlight='proof';region=new Set(proof.ids);}
  select(id,reveal);
  const n=nodes.get(id)!,rule=detectorRule(n),source=methodSource(rule.name);
  $('detector-path').textContent=`${data.detector.path}:${source.line} · ${rule.name}`;
  $('detector-explanation').textContent=rule.reason;
  $('detector-code').textContent=source.text;
  $('proof-step-status').textContent=`${id}: ${nodeExplanation(n)}`;
  document.querySelectorAll<HTMLButtonElement>('[data-proof-node]').forEach(b=>b.setAttribute('aria-pressed',String(b.dataset.proofNode===id)));
}
const ordered=[...witness.steps].reverse();
$('proof-steps').innerHTML=ordered.map((s,i)=>`<button data-proof-node="${s.node}" aria-pressed="false">${i+1}. ${esc(short(nodes.get(s.node)!))}<small>${s.node} · ${s.seed?'concrete seed':'needs '+s.dependencies.join(' + ')}</small></button>`).join('')+`<button data-proof-node="${witness.comparison}" aria-pressed="false">${ordered.length+1}. Report comparison<small>${witness.comparison} · operand ${witness.operand}</small></button>`;
function runsMarkup(runs:any) {
  return ['true','false'].map(input=>{const e=runs[input].at(-1);return `<div class="run"><span class="mono">enabled = ${input}</span><strong>== nil: ${e.result}</strong><small>${esc(e.type)} · ${esc(JSON.stringify(e.value))}</small></div>`;}).join('');
}
$('original-runs').innerHTML=runsMarkup(data.replay);
$('repair-runs').innerHTML=runsMarkup(data.repair.runs);
$('repair-contract').textContent=data.repair.contract;
$('repair-result').textContent=`Go confirms the contract for both inputs. The real detector reports ${data.repair.remainingFindings} typednil findings for this experimental function. This is a reviewed source experiment, not a safe automatic fix.`;
$('repair-source').textContent=data.repair.after;
$('detector-hash').textContent=`DETECTOR SOURCE SHA-256 ${data.detector.sha256.slice(0,16)}…`;
$('coverage').textContent=JSON.stringify(g.extraction_evidence,null,2);
$('source-path').textContent=data.path;
$('finding-location').textContent=`L${data.finding.Sites[0].Line} · ${witness.comparison}`;
$('provenance').textContent=`${data.path} · source revision ${data.revision.slice(0,12)}`;
$('integrity').textContent=`${g.nodes.length}/${g.nodes.length} nodes · ${g.edges.length}/${g.edges.length} edges loaded · source hash verified`;
$('node-select').innerHTML=g.nodes.map(n=>`<option value="${n.id}">${n.id} · ${category(n)} · L${line(n)}</option>`).join('');
$('node-select').onchange=()=>select($<HTMLSelectElement>('node-select').value,true);
$('connections').onclick=event=>{const b=(event.target as Element).closest<HTMLButtonElement>('[data-connection]');if(b)chooseEdge(b.dataset.connection!);};
function sourceSelect(event:Event) {
  const el=(event.target as Element).closest<HTMLElement>('[data-line]');
  if(!el)return;
  const candidates=lineNodes(g,data.source,Number(el.dataset.line));
  if(!candidates.length)return;
  const next=candidates.some(n=>n.id===selected)?candidates[(candidates.findIndex(n=>n.id===selected)+1)%candidates.length]:candidates[0];
  select(next.id,true);
}
$('source').onclick=sourceSelect;
$('source').onkeydown=event=>{if(event.key==='Enter'||event.key===' '){event.preventDefault();sourceSelect(event);}};
svg.addEventListener('click',event=>{
  if(dragged){dragged=false;return;}
  const el=event.target as Element;
  const n=el.closest<SVGElement>('[data-node]');
  if(n){select(n.dataset.node!);return;}
  const e=el.closest<SVGElement>('[data-edge]');
  if(e)chooseEdge(e.dataset.edge!);
});
svg.addEventListener('keydown',event=>{
  const id=(event.target as Element).closest<SVGElement>('[data-node]')?.dataset.node;
  if((event.key==='Enter'||event.key===' ')&&id){event.preventDefault();select(id);return;}
  if(event.key==='+'||event.key==='='){event.preventDefault();zoom(.8);return;}
  if(event.key==='-'){event.preventDefault();zoom(1.25);return;}
  if(event.key==='Home'){event.preventDefault();wholeGraph();return;}
  if(!['ArrowRight','ArrowLeft'].includes(event.key))return;
  event.preventDefault();const i=g.nodes.findIndex(n=>n.id===selected),d=event.key==='ArrowRight'?1:-1;
  select(g.nodes[(i+d+g.nodes.length)%g.nodes.length].id,true);
});
let drag:{x:number;y:number;box:Box}|null=null,dragged=false;
svg.addEventListener('pointerdown',event=>{
  if(event.button!==0)return;
  drag={x:event.clientX,y:event.clientY,box:{...box}};dragged=false;
  if(!(event.target as Element).closest('[data-node],[data-edge]'))svg.setPointerCapture(event.pointerId);
});
svg.addEventListener('pointermove',event=>{
  if(!drag)return;
  const dx=event.clientX-drag.x,dy=event.clientY-drag.y;
  if(Math.abs(dx)+Math.abs(dy)<5)return;
  dragged=true;const r=svg.getBoundingClientRect();
  box={...drag.box,x:drag.box.x-dx/r.width*drag.box.w,y:drag.box.y-dy/r.height*drag.box.h};camera();
});
svg.addEventListener('pointerup',()=>{drag=null;});svg.addEventListener('pointercancel',()=>{drag=null;});
svg.addEventListener('wheel',event=>{event.preventDefault();const r=svg.getBoundingClientRect();zoom(event.deltaY>0?1.14:1/1.14,(event.clientX-r.left)/r.width,(event.clientY-r.top)/r.height);},{passive:false});
mini.addEventListener('click',event=>{
  const p=mini.createSVGPoint();p.x=event.clientX;p.y=event.clientY;
  const at=p.matrixTransform(mini.getScreenCTM()!.inverse());
  box={...box,x:at.x-box.w/2,y:at.y-box.h/2};camera();
});
$('zoom-in').onclick=()=>zoom(.8);$('zoom-out').onclick=()=>zoom(1.25);
$('focus').onclick=()=>focus(new Set([selected]));
$('fit').onclick=wholeGraph;
$('slice').onclick=()=>{guideStep=-1;guideReadout();highlight='slice';region=upstream(g,selected);update();focus(region);};
$('finding').onclick=finding;
$('guide-prev').onclick=()=>startGuide(guideStep-1);
$('guide-next').onclick=()=>startGuide(guideStep+1);
$('guide-steps').onclick=event=>{const b=(event.target as Element).closest<HTMLButtonElement>('[data-guide-step]');if(b)startGuide(Number(b.dataset.guideStep));};
$('guide-steps').innerHTML=guide.map((s,i)=>`<button data-guide-step="${i}" aria-pressed="false">${['Choose branch','Assign boxed','Follow boxed','Check nil'][i]}</button>`).join('');
guideReadout();
$('theme').onclick=()=>{document.body.classList.toggle('dark');$('theme').setAttribute('aria-pressed',String(document.body.classList.contains('dark')));};
$('proof-steps').onclick=event=>{const b=(event.target as Element).closest<HTMLButtonElement>('[data-proof-node]');if(b)showProofStep(b.dataset.proofNode!);};
for(const input of document.querySelectorAll<HTMLInputElement>('input[data-kind]'))input.onchange=()=>{if(input.checked)enabled.add(input.dataset.kind!);else enabled.delete(input.dataset.kind!);refreshGraph();inspect();};
for(const b of document.querySelectorAll<HTMLButtonElement>('[data-panel]'))b.onclick=()=>{showPanel(b.dataset.panel!);if(panel==='proof')showProofStep(proofStep||witness.comparison,false);};
new ResizeObserver(()=>{const r=svg.getBoundingClientRect();if(r.width<=0||r.height<=0)return;const ratio=aspect();whole=fitBox(full,ratio,30);box=resizeBox(box,ratio);camera();}).observe(svg);
fitAll();update();startGuide(0);
Object.defineProperty(window,'codeLens',{value:{getState:()=>({selected,selectedEdge,highlight,panel,guideStep,box:{...box},nodeIDs:g.nodes.map(n=>n.id),edgeIDs:g.edges.map(e=>e.id),visibleEdges:g.edges.filter(e=>enabled.has(e.kind)).map(e=>e.id),proofNodes:[...proof.ids],proofEdges:[...proof.edges],region:[...region],sourceLine:line(nodes.get(selected)!),zoom:whole.w/box.w})}});
