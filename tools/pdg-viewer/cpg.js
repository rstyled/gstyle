// rstyle owns analysis. This boundary adapts its node IDs and recorded source spans
// to the existing inspector without deriving dependencies from syntax.
const CPG_STATEMENTS = new Set(['local_declaration','assignment','compound_assignment','return','if','while','for','loop','match','break','continue','call','closure','defer','go','select','switch']);
function normalizeCPG(input) {
 if (input.format !== 'rstyle.cpg-view/1') throw new Error('Unsupported rstyle viewer format');
 const f=input.function, files=input.files;
 if(!f || !Array.isArray(f.nodes) || !Array.isArray(files))throw new Error('Missing CPG function or source files');
 if(!['known','approximated','unknown'].includes(input.flow?.standing))throw new Error('Missing CPG flow standing');
 const sources=files.map(file=>({id:file.id,path:file.path,content:file.content}));
 const texts=new Map(sources.map(s=>[s.id,sourceText(s.content)]));
 const span=s=>{
  if(!s)return null;
  const text=texts.get(s.file);
  if(!text||!Number.isInteger(s.start)||!Number.isInteger(s.end)||s.start<0||s.end<s.start||s.end>text.bytes.length)throw new Error('Invalid CPG source span');
  return {source_id:s.file,start:s.start,end:s.end};
 };
 const known=value=>({status:'known',value});
 const nodes=f.nodes.map(n=>({id:n.id,category:CPG_STATEMENTS.has(n.kind)?n.kind==='if'||['while','for','loop','match','select','switch'].includes(n.kind)?'control':'statement':'expression',original_kind:known(n.kind),source_span:span(n.span),cpg:n}));
 nodes.push({id:f.cfg.entry,category:'entry',original_kind:known('entry')},...f.cfg.exits.map(n=>({id:n.id,category:'exit',original_kind:known(n.kind+' exit')})));
 const ids=new Set();for(const n of nodes){if(!n.id||ids.has(n.id))throw new Error('Duplicate CPG node ID');ids.add(n.id);}
 const edges=[];
 const add=(source,target,kind,extra={})=>{if(!ids.has(source)||!ids.has(target))throw new Error('CPG edge has a missing endpoint');edges.push({source,target,kind,...extra});};
 for(const n of f.nodes)for(const child of n.children)add(n.id,child,'ast');
 for(const e of f.cfg.edges)add(e.source,e.target,'cfg',{label:e.kind,loop_role:e.loop_role});
 for(const kind of ['data','control','execution'])for(const e of input.flow?.[kind]||[])add(e.source,e.target,kind,{symbol_id:e.symbol,label:e.kind,approximate:e.approximate,termination:e.termination});
 return {cpg:input,function:{id:f.id,name:known(f.name),source_span:span(f.span)},sources,nodes,edges,symbols:f.symbols.map(s=>({id:s.id,name:known(s.name)}))};
}

// Source ownership is only a display fold. All original endpoints stay in raw.
function cpgOwnership(doc, detail) {
 const raw=new Map(doc.nodes.map(n=>[n.id,n])),parents=new Map();
 for(const n of doc.nodes)for(const child of n.cpg?.children||[])parents.set(child,n.id);
 const visible=n=>!['block','expression_statement'].includes(n.cpg?.kind) && (detail==='expressions'||CPG_STATEMENTS.has(n.cpg?.kind)||['entry','exit'].includes(n.category));
 const owners=new Map();
 for(const n of doc.nodes){
  let current=n;const seen=new Set();
  while(current&&!visible(current)&&!seen.has(current.id)){
   seen.add(current.id);
   const child=current.cpg?.kind==='expression_statement'?current.cpg.children[0]:null;
   current=raw.get(child||parents.get(current.id));
  }
  owners.set(n.id,current&&visible(current)?current.id:null);
 }
 return {raw,parents,owners};
}
function cpgFindings(doc) {
 const checks=(doc.cpg.diagnostics||[]).map((d,i)=>({id:'check:'+i,type:'check',title:d.rule_id,message:d.message,actual:d.actual,limit:d.limit,lines:[...new Set((d.details||[]).flatMap(detail=>detail.lines))],details:d.details||[],scope:(d.details||[]).some(detail=>detail.lines.length)?'source lines':'function'}));
 const patterns=(doc.cpg.patterns||[]).map((p,i)=>({id:'pattern:'+i,type:'pattern',title:p.kind,message:p.observation,proposal:p.possible_refactor,inference:p.inference,counter:p.counter_evidence||[],sites:p.sites,lines:[],scope:'function'}));
 return [...checks,...patterns];
}
function cpgView(doc,detail) {
 const {raw,parents,owners}=cpgOwnership(doc,detail),texts=new Map(doc.sources.map(s=>[s.id,sourceText(s.content)]));
 const members=new Map();for(const n of doc.nodes){const id=owners.get(n.id);if(!id)continue;if(!members.has(id))members.set(id,[]);members.get(id).push(n);}
 const findings=cpgFindings(doc);
 const nodes=doc.nodes.filter(n=>owners.get(n.id)===n.id).map(n=>{
  const text=texts.get(n.source_span?.source_id),d=describe(n,members.get(n.id),text,doc);
  d.kind=n.cpg?.kind||n.category;d.cpg=n.cpg;d.spine=CPG_STATEMENTS.has(d.kind)||['entry','exit'].includes(n.category);
  d.findings=findings.filter(f=>f.lines.length&&d.members.some(m=>{const t=texts.get(m.source_span?.source_id);return t&&f.lines.includes(t.line(m.source_span.start));})).map(f=>f.id);
  return d;
 });
 const visibleIDs=new Set(nodes.map(n=>n.id)),anchor=id=>{
  let at=raw.get(id);const seen=new Set();while(at&&!seen.has(at.id)){seen.add(at.id);if(CPG_STATEMENTS.has(at.cpg?.kind)||['entry','exit'].includes(at.category))return owners.get(at.id);at=raw.get(parents.get(at.id));}return null;
 };
 for(const n of nodes){n.anchor=anchor(n.id);n.parent=owners.get(parents.get(n.id));if(n.parent===n.id)n.parent=null;}
 const edges=new Map();
 for(const e of doc.edges){const source=owners.get(e.source),target=owners.get(e.target);if(!visibleIDs.has(source)||!visibleIDs.has(target)||source===target)continue;const key=[source,target,e.kind,e.label||'',e.symbol_id||''].join('\n');if(!edges.has(key))edges.set(key,{...e,source,target,symbols:e.symbol_id?[symbolName(doc,e.symbol_id)]:[],raw:[]});edges.get(key).raw.push(e);}
 return {nodes,edges:[...edges.values()],findings};
}

// Statements form the source-order spine. Syntax grows below it. Dependencies
// arc above it; CFG edges remain distinct even when they share endpoints.
function cpgLayout(nodes,edges,size) {
 const spine=nodes.filter(n=>n.spine).sort((a,b)=>a.category==='entry'?-1:b.category==='entry'?1:a.category==='exit'?1:b.category==='exit'?-1:(a.span?.start||0)-(b.span?.start||0));
 const positions=new Map(),groups=new Map(spine.map(n=>[n.id,[]]));
 for(const n of nodes)if(!n.spine){const id=groups.has(n.anchor)?n.anchor:spine[0]?.id;if(id)groups.get(id).push(n);}
 let x=70;const baseline=200;
 for(const s of spine){const group=groups.get(s.id),width=Math.max(170,Math.min(group.length,4)*110),center=x+width/2;
  positions.set(s.id,{x:center,y:baseline,...size(s)});
  group.sort((a,b)=>(a.span?.start||0)-(b.span?.start||0));
  group.forEach((n,i)=>positions.set(n.id,{x:x+55+(i%4)*110,y:baseline+100+Math.floor(i/4)*75,...size(n)}));
  x+=width+45;
 }
 const paths=[],labelPoints=[],routes=[];
 for(const e of edges){const a=positions.get(e.source),b=positions.get(e.target);if(!a||!b){paths.push('');routes.push(null);labelPoints.push(null);continue;}
  let d,label;
  if(['data','control','execution'].includes(e.kind)){
   const lift=65+Math.min(100,Math.abs(b.x-a.x)*.12)+(e.kind==='control'?32:0),y=baseline-lift;
   d=`M ${a.x} ${a.y-a.h/2} C ${a.x} ${y}, ${b.x} ${y}, ${b.x} ${b.y-b.h/2}`;label={x:(a.x+b.x)/2,y:y+12};
  }else if(e.kind==='ast'){
   d=`M ${a.x} ${a.y+a.h/2} C ${a.x} ${a.y+65}, ${b.x} ${b.y-65}, ${b.x} ${b.y-b.h/2}`;label=null;
  }else{
   const sign=b.x>=a.x?1:-1,ax=a.x+sign*a.w/2,bx=b.x-sign*b.w/2,bend=e.label==='false'?60:0;
   d=`M ${ax} ${a.y+8} C ${ax+sign*35} ${a.y+8+bend}, ${bx-sign*35} ${b.y+8+bend}, ${bx} ${b.y+8}`;label={x:(ax+bx)/2,y:(a.y+b.y)/2+3+bend*.75};
  }
  paths.push(d);labelPoints.push(label);routes.push([{x:a.x,y:a.y},{x:b.x,y:b.y}]);
 }
 return {positions,paths,labelPoints,routes,width:x,height:Math.max(400,...[...positions.values()].map(p=>p.y+p.h/2+50)),tray:null};
}

function cpgStanding(doc) {
 const flow=doc.cpg.flow;
 return `rstyle flow: ${flow.standing}. ${[...(flow.reasons||[]),...(flow.assumptions||[])].join(' ')} Missing links do not prove independence.`;
}
function cpgFindingsPanel(doc,graph,node) {
 const findings=graph.findings||cpgFindings(doc),row=f=>{
  const selected=state.view?.findingID===f.id;
  const evidence=f.lines.length?'Recorded source lines '+f.lines.join(', '):'Function-wide evidence; no exact operation link is supplied';
  const measure=f.type==='check'?`actual ${f.actual} / limit ${f.limit}`:'candidate';
  const link=f.type==='check'?'check':'pattern';
  return `<div class="finding-row"><button class="link" data-finding="${esc(f.id)}" aria-pressed="${selected}"><strong>${esc(f.title)}</strong><small>${esc(measure)} · ${link}</small></button><p>${esc(f.message)}</p><p class="note">${esc(evidence)}</p>${f.proposal?`<p>${esc(f.proposal)}</p>`:''}${f.sites?cpgPatternSites(f.sites):''}${f.inference?`<details><summary>Inference and counter-evidence</summary><p>${esc(f.inference)}</p>${f.counter.map(s=>`<p>${esc(s)}</p>`).join('')}</details>`:''}</div>`;
 };
 const relevant=node?findings.filter(f=>!f.lines.length||node.findings?.includes(f.id)):findings;
 return `<section id="cpg-findings"><h3>rstyle checks and patterns <span>${findings.length}</span></h3><p class="note">${node?'Evidence matching this operation’s source line, plus function-wide findings.':'Select a finding to highlight its recorded source lines. Function-wide findings highlight the whole function.'}</p>${relevant.map(row).join('')||'<p class="empty">No matching findings reported.</p>'}<p class="note">Go CPG library-effect classification is incomplete. Side-effect density is not a reliable health result here.</p></section>`;
}
function selectCPGFinding(id) {
 const view=state.view,f=view.graph.findings?.find(f=>f.id===id);
 if(!f)return;
 view.findingID=view.findingID===id?null:id;
 view.evidenceIDs=view.findingID?new Set(view.graph.nodes.filter(n=>!f.lines.length||n.findings?.includes(id)).map(n=>n.id)):null;
 state.selected=null;state.hover=null;paint();inspector(pdgPanel(view.doc,view.graph,null));
 if(view.evidenceIDs?.size){const first=view.evidenceIDs.values().next().value;centerOn(first);}
 if(f.lines.length){for(const row of document.querySelectorAll('#pdg-source .ln'))row.classList.toggle('sel',!!view.findingID&&f.lines.includes(Number(row.dataset.n)));}
}
function cpgEvidenceDetails(doc) {
 const flow=doc.cpg.flow,capabilities={...doc.cpg.coverage.capabilities,...doc.cpg.function.coverage?.capabilities};
 return `<details><summary>rstyle producer, coverage and assumptions</summary><p class="note">${esc(doc.cpg.producer.name)} ${esc(doc.cpg.producer.version)} · compat.cpg/0.1.0 · flow ${esc(flow.standing)}</p>${[...(flow.reasons||[]),...(flow.assumptions||[])].map(s=>`<p class="note">${esc(s)}</p>`).join('')}${Object.entries(capabilities).map(([k,v])=>`<div class="cap"><b>${esc(k)}</b><span class="status ${esc(v.status)}">${esc(v.status)}</span><span></span><span>${esc(v.description)}</span></div>`).join('')}<p class="note">Check details identify source lines, not exact operations. AST wrappers fold for display; raw nodes and edge endpoints remain in the response.</p></details>`;
}

function cpgPatternSites(sites) {
 const row=site=>{
  const matches=(state.model?.index.symbols||[]).filter(s=>s.file===site.path&&s.line===site.line);
  const label=esc((site.name||site.id)+' · '+site.path+':'+site.line);
  return matches.length===1?`<a class="link" href="${href.pdg(matches[0].id)}">${label}</a>`:`<p>${label}</p>`;
 };
 return `<details><summary>Pattern member sites (${sites.length})</summary>${sites.map(row).join('')}</details>`;
}
