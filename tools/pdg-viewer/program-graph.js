// The program graph contains every indexed declaration, including disconnected
// functions, and a visible endpoint for every recorded unresolved call spelling.
function buildProgramGraph(index) {
 const nodes=index.symbols.map(s=>({id:s.id,name:s.name,package:s.package,file:s.file,line:s.line,role:s.role||'',kind:'function'}));
 const known=new Set(nodes.map(n=>n.id)),unknown=new Map(),links=new Map();
 for(const s of index.symbols)for(const call of s.calls){
  let target=call.target;
  if(!known.has(target)){
   target='unresolved:'+JSON.stringify([s.package,call.spelling]);
   if(!unknown.has(target))unknown.set(target,{id:target,name:call.spelling,package:s.package,kind:'unresolved'});
  }
  const key=JSON.stringify([s.id,target]);
  if(links.has(key)){links.get(key).count++;continue;}
  links.set(key,{source:s.id,target,count:1,kind:known.has(target)?'candidate':'unresolved'});
 }
 return {nodes:[...nodes,...unknown.values()],edges:[...links.values()],entrypoints:nodes.filter(n=>n.role==='entrypoint'),packages:index.packages||[]};
}
function entrypointLayout(graph,entryID) {
 const entry=graph.nodes.find(n=>n.id===entryID&&n.role==='entrypoint');
 if(!entry)return {...graphLayout(graph.nodes,graph.edges,{columns:Math.max(12,Math.ceil(Math.sqrt(graph.nodes.length)))}),groups:[],reached:new Set(),initializers:[]};
 const packageMap=new Map(graph.packages.map(p=>[p.id,p])),scope=new Set([entry.package]),packageQueue=[entry.package];
 for(let i=0;i<packageQueue.length;i++)for(const id of packageMap.get(packageQueue[i])?.imports||[]){if(!packageMap.has(id)||scope.has(id))continue;scope.add(id);packageQueue.push(id);}
 const initializers=graph.nodes.filter(n=>n.role==='initializer'&&scope.has(n.package));
 const adjacency=new Map();for(const e of graph.edges){if(!adjacency.has(e.source))adjacency.set(e.source,[]);adjacency.get(e.source).push(e.target);}
 const distance=new Map([[entry.id,0],...initializers.map(n=>[n.id,0])]),queue=[...distance.keys()];
 for(let i=0;i<queue.length;i++)for(const id of adjacency.get(queue[i])||[]){if(distance.has(id))continue;distance.set(id,distance.get(queue[i])+1);queue.push(id);}
 const reached=new Set(distance.keys()),positions=new Map(),groups=[];
 const reachedFunctions=graph.nodes.filter(n=>n.kind==='function'&&reached.has(n.id)&&n.role!=='initializer');
 const columns=Math.max(5,Math.min(12,Math.ceil(Math.sqrt(reachedFunctions.length)))),leftWidth=columns*260+360;
 reachedFunctions.sort((a,b)=>a.package.localeCompare(b.package)||a.name.localeCompare(b.name));
 const layers=[];for(const n of reachedFunctions){const d=distance.get(n.id);if(!layers[d])layers[d]=[];layers[d].push(n);}
 let y=130;
 for(const layer of layers){if(!layer)continue;layer.forEach((n,i)=>{const row=Math.floor(i/columns),size=Math.min(columns,layer.length-row*columns);positions.set(n.id,{x:330+(leftWidth-360)/2+((i%columns)-(size-1)/2)*260,y:y+row*125});});y+=Math.ceil(layer.length/columns)*125;}
 groups.push({x:310,y:35,w:leftWidth-280,h:y-5,title:'FROM main.main — recorded call reachability'});
 initializers.forEach((n,i)=>positions.set(n.id,{x:160,y:130+i*125}));
 groups.push({x:25,y:35,w:260,h:Math.max(160,initializers.length*125+60),title:'PACKAGE init()'});
 const boundaries=graph.nodes.filter(n=>n.kind==='unresolved'&&reached.has(n.id));
 const placeGrid=(nodes,x,top,cols)=>{nodes.forEach((n,i)=>positions.set(n.id,{x:x+135+(i%cols)*260,y:top+100+Math.floor(i/cols)*125}));return {w:Math.max(1,Math.min(cols,nodes.length))*260+20,h:Math.max(160,Math.ceil(nodes.length/cols)*125+80)};};
 const boundaryBox=placeGrid(boundaries,310,y+80,columns);groups.push({x:310,y:y+80,...boundaryBox,title:'CALL BOUNDARIES — unresolved / external / builtin'});
 const other=graph.nodes.filter(n=>n.kind==='function'&&!reached.has(n.id)),otherCols=Math.max(12,Math.ceil(Math.sqrt(other.length))),rightX=leftWidth+650;
 other.sort((a,b)=>a.package.localeCompare(b.package)||a.name.localeCompare(b.name));
 const otherBox=placeGrid(other,rightX,35,otherCols);groups.push({x:rightX,y:35,...otherBox,title:'NOT REACHED BY RECORDED CALLS — not proof of unused code'});
 const otherBoundaries=graph.nodes.filter(n=>n.kind==='unresolved'&&!reached.has(n.id));
 const otherBoundaryBox=placeGrid(otherBoundaries,rightX,otherBox.h+180,otherCols);groups.push({x:rightX,y:otherBox.h+180,...otherBoundaryBox,title:'OTHER CALL BOUNDARIES'});
 const width=Math.max(...groups.map(g=>g.x+g.w))+70,height=Math.max(...groups.map(g=>g.y+g.h))+70;
 groups.push(...packageBands(graph.nodes,positions));
 return {positions,width,height,groups,reached,initializers,entry};
}
// Keep call data complete; select its presentation independently.
function programEdgePlan(graph,selected,allEdges=false) {
 const detailed=graph.edges.filter(e=>allEdges||e.source===selected||e.target===selected);
 const byID=new Map(graph.nodes.map(n=>[n.id,n])),bundles=new Map();
 for(const e of graph.edges){const a=byID.get(e.source),b=byID.get(e.target);if(!a||!b||a.package===b.package)continue;
  const key=JSON.stringify([a.package,b.package]);if(!bundles.has(key))bundles.set(key,{source:a.package,target:b.package,count:0});bundles.get(key).count+=e.count;
 }
 return {detailed,bundles:[...bundles.values()]};
}
function packageBands(nodes,positions) {
 const bands=new Map();
 for(const n of nodes){if(n.kind!=='function')continue;const p=positions.get(n.id),key=JSON.stringify([n.package,p.y]);if(!bands.has(key))bands.set(key,{x:p.x-125,y:p.y-70,w:250,h:115,title:n.package,zone:'package'});const band=bands.get(key),right=Math.max(band.x+band.w,p.x+125);band.x=Math.min(band.x,p.x-125);band.w=right-band.x;}
 return [...bands.values()];
}
function programCanvas(canvas,graph,onSelect) {
 let layout=entrypointLayout(graph,graph.entrypoints[0]?.id);
 const nodeByID=new Map(graph.nodes.map(n=>[n.id,n])),neighbors=new Map();
 for(const e of graph.edges){for(const id of [e.source,e.target]){if(!neighbors.has(id))neighbors.set(id,new Set());neighbors.get(id).add(e.source);neighbors.get(id).add(e.target);}}
 const packages=[...new Set(graph.nodes.map(n=>n.package))],palette=['#37796b','#687f9c','#9b745c','#85805b','#957c9e','#699274'];
 const colors=new Map(packages.map((p,i)=>[p,palette[i%palette.length]]));
 let view={x:0,y:0,w:layout.width,h:layout.height},selected='',matches=null,drag=null,moved=false,frame=0,allEdges=false;
 const ctx=canvas.getContext('2d');
 let cachedEdgeKey='',cachedEdgePlan=null;
 const fit=()=>{const ratio=canvas.clientWidth/canvas.clientHeight,w=Math.max(layout.width,layout.height*ratio),h=w/ratio;view={x:(layout.width-w)/2,y:(layout.height-h)/2,w,h};schedule();};
 const world=(x,y)=>({x:view.x+x/canvas.clientWidth*view.w,y:view.y+y/canvas.clientHeight*view.h});
 function render(){
  frame=0;const dpr=Math.min(2,devicePixelRatio||1),width=canvas.clientWidth,height=canvas.clientHeight;
  if(!width||!height)return;
  canvas.width=Math.round(width*dpr);canvas.height=Math.round(height*dpr);ctx.setTransform(dpr,0,0,dpr,0,0);ctx.fillStyle='#fdfdf9';ctx.fillRect(0,0,width,height);
  const scale=width/view.w;ctx.translate(-view.x*scale,-view.y*scale);ctx.scale(scale,scale);
  const atEntry=selected===layout.entry?.id;
  const active=atEntry?layout.reached:neighbors.get(selected)||new Set([selected]);
  for(const group of layout.groups){if(group.zone==='package'&&scale<.14)continue;ctx.globalAlpha=group.zone==='package'?.55:1;ctx.strokeStyle=group.zone==='package'?'#dee4d6':'#c0cbbc';ctx.lineWidth=1/scale;ctx.strokeRect(group.x,group.y,group.w,group.h);if(scale>.06){ctx.fillStyle='#526a48';ctx.font=group.zone==='package'?'13px monospace':'bold 18px monospace';ctx.fillText(group.title.split('/').slice(-2).join('/'),group.x+12,group.y+25);}}
  const edgeKey=JSON.stringify([selected,allEdges]);
  if(edgeKey!==cachedEdgeKey){cachedEdgePlan=programEdgePlan(graph,selected,allEdges);cachedEdgeKey=edgeKey;}
  const edgePlan=cachedEdgePlan;
  canvas.dataset.edgeMode=allEdges?'all':'selected';canvas.dataset.drawnEdges=String(edgePlan.detailed.length);canvas.dataset.totalEdges=String(graph.edges.length);
  if(scale<.18&&!allEdges){
   const centers=new Map();for(const n of graph.nodes){if(n.kind!=='function')continue;const p=layout.positions.get(n.id);if(!centers.has(n.package))centers.set(n.package,{x:0,y:0,count:0});const c=centers.get(n.package);c.x+=p.x;c.y+=p.y;c.count++;}
   for(const c of centers.values()){c.x/=c.count;c.y/=c.count;}
   ctx.strokeStyle='#7d9888';ctx.globalAlpha=.2;ctx.lineWidth=.7/scale;
   for(const e of edgePlan.bundles){const a=centers.get(e.source),b=centers.get(e.target);if(!a||!b)continue;ctx.beginPath();ctx.moveTo(a.x,a.y);ctx.bezierCurveTo(a.x,(a.y+b.y)/2,b.x,(a.y+b.y)/2,b.x,b.y);ctx.stroke();}
  }
  for(const e of edgePlan.detailed){const a=layout.positions.get(e.source),b=layout.positions.get(e.target);if(!a||!b)continue;
   const relevant=selected&&(e.source===selected||e.target===selected);
   ctx.strokeStyle=relevant?'#246a57':e.kind==='unresolved'?'#c4b49a':'#b2c3b5';ctx.globalAlpha=relevant?.85:.035;ctx.lineWidth=(relevant?1.3:.45)/scale;
   ctx.beginPath();const end=b.y-39;ctx.moveTo(a.x,a.y+39);const y=(a.y+39+end)/2;if(a===b){ctx.bezierCurveTo(a.x+180,a.y+120,b.x+180,b.y-120,b.x,end);}else{ctx.bezierCurveTo(a.x,y,b.x,y,b.x,end);}ctx.stroke();
   if(scale>.12){ctx.fillStyle=ctx.strokeStyle;ctx.beginPath();ctx.moveTo(b.x,end);const direction=a===b||end>=y?1:-1;ctx.lineTo(b.x-7,end-13*direction);ctx.lineTo(b.x+7,end-13*direction);ctx.closePath();ctx.fill();}
  }
  ctx.globalAlpha=1;
  for(const n of graph.nodes){const p=layout.positions.get(n.id);if(p.x+115<view.x||p.x-115>view.x+view.w||p.y+42<view.y||p.y-42>view.y+view.h)continue;
   const chosen=n.id===selected,matched=matches?.has(n.id);ctx.globalAlpha=selected&&!active.has(n.id) ? .65 : matches&&!matched ? .25 : 1;
   ctx.fillStyle=chosen?'#255f4d':n.role==='entrypoint'?'#193e2e':n.role==='initializer'?'#80638d':matched?'#e0b65c':n.kind==='unresolved'?'#e5d9bd':colors.get(n.package);
   ctx.fillRect(p.x-110,p.y-39,220,78);if(n.role==='entrypoint'){ctx.strokeStyle='#b89b3d';ctx.lineWidth=3/scale;ctx.strokeRect(p.x-110,p.y-39,220,78);}
   if(scale>.18){ctx.fillStyle=chosen?'#fff':'#fffdf4';ctx.font='14px monospace';ctx.fillText((n.role==='entrypoint'?'ENTRYPOINT · ':n.role==='initializer'?'STARTUP · ':'')+n.name.slice(0,25),p.x-100,p.y-6);ctx.font='11px monospace';ctx.fillText(n.kind==='unresolved'?'unresolved call':n.file.split('/').slice(-2).join('/')+':'+n.line,p.x-100,p.y+17);}
  }
  ctx.globalAlpha=1;
 }
 function schedule(){if(!frame)frame=requestAnimationFrame(render);}
 function zoom(factor,px=.5,py=.5){const w=Math.min(layout.width*5,Math.max(450,view.w*factor)),h=w*view.h/view.w;view={x:view.x+(view.w-w)*px,y:view.y+(view.h-h)*py,w,h};schedule();}
 function focus(id){const p=layout.positions.get(id);if(!p)return;selected=id;const w=1000,h=w*canvas.clientHeight/canvas.clientWidth;view={x:p.x-w/2,y:nodeByID.get(id)?.role==='entrypoint'?Math.max(0,p.y-100):p.y-h/2,w,h};schedule();}
 canvas.addEventListener('wheel',e=>{e.preventDefault();const r=canvas.getBoundingClientRect();zoom(e.deltaY>0?1.18:1/1.18,(e.clientX-r.left)/r.width,(e.clientY-r.top)/r.height);},{passive:false});
 canvas.addEventListener('pointerdown',e=>{if(e.button!==0)return;drag={x:e.clientX,y:e.clientY,view:{...view}};moved=false;canvas.setPointerCapture(e.pointerId);});
 canvas.addEventListener('pointermove',e=>{if(!drag)return;const dx=e.clientX-drag.x,dy=e.clientY-drag.y;if(Math.abs(dx)+Math.abs(dy)>5)moved=true;view={...drag.view,x:drag.view.x-dx/canvas.clientWidth*drag.view.w,y:drag.view.y-dy/canvas.clientHeight*drag.view.h};schedule();});
 canvas.addEventListener('pointerup',e=>{if(!drag)return;drag=null;if(moved)return;const r=canvas.getBoundingClientRect(),p=world(e.clientX-r.left,e.clientY-r.top);
  const n=graph.nodes.find(n=>{const b=layout.positions.get(n.id);return Math.abs(p.x-b.x)<=110&&Math.abs(p.y-b.y)<=39;});
  if(!n)return;selected=n.id;schedule();onSelect(n);
 });canvas.addEventListener('pointercancel',()=>{drag=null;});
 new ResizeObserver(()=>{view.h=view.w*canvas.clientHeight/canvas.clientWidth;schedule();}).observe(canvas);
 fit();return {fit,zoom,focus,showAllEdges:value=>{allEdges=value;schedule();},setEntry:id=>{layout=entrypointLayout(graph,id);const entry=layout.entry;if(entry)focus(entry.id);else fit();return {reached:[...layout.reached].filter(id=>nodeByID.get(id)?.kind==='function').length,initializers:layout.initializers.length,total:graph.nodes.filter(n=>n.kind==='function').length};},select:id=>{selected=id;schedule();},search:ids=>{matches=ids.size?ids:null;schedule();},nodeByID};
}
