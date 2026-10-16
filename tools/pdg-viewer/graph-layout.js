// Collapse dependency cycles before assigning layers. Self and feedback edges
// remain in the drawing; ranks describe layout, not execution time.
function graphComponents(ids, edges) {
 const adjacency=new Map(ids.map(id=>[id,[]]));
 for(const e of edges) if(adjacency.has(e.source)&&adjacency.has(e.target)) adjacency.get(e.source).push(e.target);
 const visited=new Set(),order=[];
 // Iterative DFS keeps large functions off the JavaScript call stack.
 for(const start of ids){
  if(visited.has(start))continue;
  const stack=[[start,false]];
  while(stack.length){const [id,done]=stack.pop();if(done){order.push(id);continue;}if(visited.has(id))continue;visited.add(id);stack.push([id,true]);for(const next of adjacency.get(id))if(!visited.has(next))stack.push([next,false]);}
 }
 const reverse=new Map(ids.map(id=>[id,[]]));
 for(const [id,targets]of adjacency)for(const target of targets)reverse.get(target).push(id);
 const components=new Map();let component=0;
 for(const start of order.reverse()){
  if(components.has(start))continue;
  const stack=[start];components.set(start,component);
  while(stack.length){for(const next of reverse.get(stack.pop()))if(!components.has(next)){components.set(next,component);stack.push(next);}}
  component++;
 }
 return {components,count:component};
}
function graphLayout(nodes,edges,options={}) {
 const ids=nodes.map(n=>n.id),{components,count}=graphComponents(ids,edges);
 const next=Array.from({length:count},()=>new Set()),degree=Array(count).fill(0),rank=Array(count).fill(0);
 for(const e of edges){const a=components.get(e.source),b=components.get(e.target);if(a===undefined||b===undefined||a===b||next[a].has(b))continue;next[a].add(b);degree[b]++;}
 const queue=degree.map((n,i)=>n===0?i:-1).filter(i=>i>=0);
 for(let i=0;i<queue.length;i++)for(const b of next[queue[i]]){rank[b]=Math.max(rank[b],rank[queue[i]]+1);if(--degree[b]===0)queue.push(b);}
 const layers=[];
 for(const n of nodes){const r=rank[components.get(n.id)];if(!layers[r])layers[r]=[];layers[r].push(n);}
 // Barycentric sweeps shorten edges while preserving deterministic tie order.
 const parents=new Map(ids.map(id=>[id,[]]));for(const e of edges)if(parents.has(e.target))parents.get(e.target).push(e.source);
 const order=new Map();
 for(let sweep=0;sweep<3;sweep++)for(const layer of layers){if(!layer)continue;layer.forEach((n,i)=>order.set(n.id,i));layer.sort((a,b)=>{const score=n=>{const p=parents.get(n.id).filter(id=>order.has(id));return p.length?p.reduce((sum,id)=>sum+order.get(id),0)/p.length:order.get(n.id);};return score(a)-score(b);});layer.forEach((n,i)=>order.set(n.id,i));}
 const columns=options.columns||7,stepX=260,stepY=125;
 const width=Math.max(900,...layers.map(l=>Math.min(columns,l?.length||0)*stepX+100)),positions=new Map();
 let y=90;
 for(const layer of layers){
  if(!layer)continue;
  layer.forEach((n,i)=>{const row=Math.floor(i/columns),size=Math.min(columns,layer.length-row*columns);positions.set(n.id,{x:width/2+((i%columns)-(size-1)/2)*stepX,y:y+row*stepY,w:220,h:78});});
  y+=Math.ceil(layer.length/columns)*stepY;
 }
 const height=Math.max(500,y+70);
 return {positions,width,height};
}
function graphEdgePath(a,b,index) {
 if(a===b)return `M${a.x+100},${a.y} C${a.x+170},${a.y-90} ${a.x+170},${a.y+90} ${a.x+100},${a.y+10}`;
 const forward=b.y>a.y,sy=a.y+(forward?39:-39),ty=b.y+(forward?-39:39);
 if(forward)return `M${a.x},${sy} C${a.x},${(sy+ty)/2} ${b.x},${(sy+ty)/2} ${b.x},${ty}`;
 const offset=145+(index%5)*15;
 return `M${a.x+110},${a.y} C${a.x+offset},${a.y-100} ${b.x+offset},${b.y-100} ${b.x+110},${b.y}`;
}
function enableGraphCamera(svg) {
 let box={x:0,y:0,w:1000,h:620},initial={...box},drag=null,moved=false;
 const apply=()=>svg.setAttribute('viewBox',`${box.x} ${box.y} ${box.w} ${box.h}`);
 const reset=(w,h)=>{box={x:0,y:0,w,h};initial={...box};apply();};
 const zoom=(factor,px=.5,py=.5)=>{const w=Math.max(200,Math.min(initial.w*3,box.w*factor)),h=w*box.h/box.w;box={x:box.x+(box.w-w)*px,y:box.y+(box.h-h)*py,w,h};apply();};
 svg.addEventListener('wheel',e=>{e.preventDefault();const r=svg.getBoundingClientRect();zoom(e.deltaY>0?1.12:1/1.12,(e.clientX-r.left)/r.width,(e.clientY-r.top)/r.height);},{passive:false});
 svg.addEventListener('pointerdown',e=>{if(e.button!==0)return;drag={x:e.clientX,y:e.clientY,b:{...box}};moved=false;if(!e.target.closest('[data-node],[data-package]'))svg.setPointerCapture(e.pointerId);});
 svg.addEventListener('pointermove',e=>{if(!drag)return;const dx=e.clientX-drag.x,dy=e.clientY-drag.y;if(Math.abs(dx)+Math.abs(dy)>5)moved=true;const r=svg.getBoundingClientRect();box={...drag.b,x:drag.b.x-dx*r.width**-1*box.w,y:drag.b.y-dy*r.height**-1*box.h};apply();});
 svg.addEventListener('pointerup',()=>{drag=null;});svg.addEventListener('pointercancel',()=>{drag=null;});
 svg.addEventListener('click',e=>{if(moved){e.stopImmediatePropagation();moved=false;}},true);
 return {reset,fit:()=>reset(initial.w,initial.h),zoom,focus:(x,y)=>{const r=svg.getBoundingClientRect(),w=950,h=w*r.height/r.width;box={x:x-w/2,y:y-h/2,w,h};apply();}};
}
