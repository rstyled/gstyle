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
// The viewport surface paints crisp text, zoomed-in tiles, and selected calls.
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
 const scene = document.createElement('canvas');
 scene.className = 'overview-scene'; scene.setAttribute('aria-hidden', 'true');
 stage.insertBefore(scene, canvas);
 const ctx = scene.getContext('2d'), root = getComputedStyle(document.documentElement);
 let pose=null,dirty=true;
 const color = name => root.getPropertyValue('--' + name).trim();
 const mix = (a, b, ratio) => {
  const rgb = hex => hex.replace('#','').match(/../g).map(s => parseInt(s,16));
  const left=rgb(a), right=rgb(b);
  return `rgb(${left.map((v,i)=>Math.round(v*ratio+right[i]*(1-ratio))).join(',')})`;
 };
 const palette = new Map(LAYERS.map(layer => [layer, {tile:color(layer),fill:mix(color(layer),color('raised'),.38),line:mix(color(layer),color('line'),.7)}]));
 const images = raster.images ||= Object.fromEntries(['calls','tiles'].map(name => {const img = new Image();img.src=raster[name];return [name,img];}));
 for (const img of Object.values(images)) img.onload = () => {dirty=true;if(scene.isConnected)applyCamera();};
 const strokes = layout.overviewStrokes ||= paths.map(path=>new Path2D(path));
 canvas.innerHTML = '<title id="overview-hover-title"></title><desc id="overview-help">Every indexed function is on this map. Use search or the arrow keys to select a function. Press Enter to open its dependence graph.</desc><g id="overview-selection" role="button"></g>';
 canvas.setAttribute('tabindex','0');canvas.setAttribute('aria-describedby','overview-help');
 const selection = canvas.querySelector('#overview-selection');
 let center = null, selected = null;
 let visibleCount = 0;
 const inView = (p,b) => p.x+p.w>=b.x && p.x<=b.x+b.w && p.y+p.h>=b.y && p.y<=b.y+b.h;
 const cached = (img, bounds, alpha) => {
  if(!img.complete || !img.naturalWidth)return;
  const x=Math.max(0,bounds.x),y=Math.max(0,bounds.y),w=Math.min(layout.width,bounds.x+bounds.w)-x,h=Math.min(layout.height,bounds.y+bounds.h)-y;
  if(w<=0||h<=0)return;
  ctx.globalAlpha=alpha;
  ctx.drawImage(img,x/layout.width*img.width,y/layout.height*img.height,w/layout.width*img.width,h/layout.height*img.height,x,y,w,h);
  ctx.globalAlpha=1;
 };
 const tile = (target,id,alpha,label) => {
  const n=index.nodes.get(id),p=layout.positions.get(id),colors=palette.get(n.layer),chosen=id===selected;
  target.globalAlpha=alpha;target.fillStyle=chosen||id===center&&id===state.hover?colors.tile:colors.fill;
  target.strokeStyle=chosen?color('ink'):colors.line;target.lineWidth=chosen?1.2/cam.k:1;
  target.beginPath();target.roundRect(p.x-p.w/2,p.y-p.h/2,p.w,p.h,2);target.fill();target.stroke();
  if(label){target.fillStyle=color('ink');target.font='11px '+root.getPropertyValue('--mono');target.fillText(clip(n.label,16),p.x-p.w/2+6,p.y+4);}
  target.globalAlpha=1;
 };
 const arrow = (i) => {
  const route=layout.routes[i],p=layout.positions.get(graph.edges[i].source);
  const end=route?.at(-1) || {x:p.x+p.w/2-6,y:p.y-p.h/2-1};
  const prev=route?.at(-2),angle=prev?Math.abs(end.x-prev.x)>.01?(end.x>prev.x?0:Math.PI):(end.y>prev.y?Math.PI/2:-Math.PI/2):Math.atan2(29,-8);
  ctx.save();ctx.translate(end.x,end.y);ctx.rotate(angle);ctx.beginPath();ctx.moveTo(0,0);ctx.lineTo(-8,-3.2);ctx.lineTo(-8,3.2);ctx.closePath();ctx.fill();ctx.restore();
 };
 const camera = () => {
  const width=canvas.clientWidth,height=canvas.clientHeight,dpr=Math.min(devicePixelRatio||1,2);
  // A padded surface moves through the compositor during a drag. Paint again
  // only when its margin is exhausted or the content changes.
  const moving=!!drag?.moved,dx=pose?(pose.x-cam.x)*cam.k:0,dy=pose?(pose.y-cam.y)*cam.k:0;
  if(moving&&!dirty&&pose&&pose.k===cam.k&&pose.width===width&&pose.height===height&&pose.dpr===dpr&&pose.calls===state.allCalls&&Math.abs(dx)<pose.pad*.75&&Math.abs(dy)<pose.pad*.75){
   scene.style.transform=`translate3d(${dx}px,${dy}px,0)`;
   return;
  }
  const pad=moving&&cam.k>=.5?320:0;
  scene.style.width=(width+2*pad)+'px';scene.style.height=(height+2*pad)+'px';scene.style.left=-pad+'px';scene.style.top=-pad+'px';scene.style.transform='';
  const pixelWidth=Math.round((width+2*pad)*dpr),pixelHeight=Math.round((height+2*pad)*dpr);
  if(scene.width!==pixelWidth||scene.height!==pixelHeight){scene.width=pixelWidth;scene.height=pixelHeight;}
  ctx.setTransform(1,0,0,1,0,0);ctx.clearRect(0,0,scene.width,scene.height);
  ctx.setTransform(cam.k*dpr,0,0,cam.k*dpr,(-cam.x*cam.k+pad)*dpr,(-cam.y*cam.k+pad)*dpr);
  const bounds={x:cam.x-pad/cam.k,y:cam.y-pad/cam.k,w:(width+2*pad)/cam.k,h:(height+2*pad)/cam.k},related=index.adjacent.get(center)||new Set();
  const frames=layout.frames.filter(p=>inView(p,bounds));
  ctx.lineWidth=1/cam.k;
  for(const p of frames){ctx.fillStyle='#191917';ctx.strokeStyle=p.id===(state.route.pkg||index.nodes.get(selected)?.package)?color('ink-2'):color('line');ctx.beginPath();ctx.roundRect(p.x,p.y,p.w,p.h,10);ctx.fill();ctx.stroke();}
  if(state.allCalls)cached(images.calls,bounds,.08);
  if(cam.k<.5)cached(images.tiles,bounds,center ? .35 : 1);
  if(center){
   ctx.strokeStyle=color('ink-2');ctx.fillStyle=color('ink-2');ctx.lineWidth=1.2/cam.k;
   for(const i of index.incident.get(center)){ctx.stroke(strokes[i]);arrow(i);}
  }
  if(cam.k>=.5){
   const visible=index.query(bounds);visibleCount=visible.size;
   for(const id of visible)tile(ctx,id,!center||related.has(id)?1:.35,true);
  } else {visibleCount=0;for(const id of related)tile(ctx,id,1,false);}
  if(selected){const p=layout.positions.get(selected);ctx.strokeStyle=color('ink');ctx.lineWidth=1.2/cam.k;ctx.strokeRect(p.x-p.w/2,p.y-p.h/2,p.w,p.h);}
  ctx.font=`500 ${12/cam.k}px `+root.getPropertyValue('--sans');ctx.fillStyle=color('ink-2');
  for(const p of frames){const chars=Math.max(3,Math.floor((p.w-32)*cam.k/7));ctx.fillText(clip(p.label+' · '+p.count.toLocaleString(),chars),p.x+16,p.y+60);}
  ctx.font=`600 ${14/cam.k}px `+root.getPropertyValue('--sans');
  for(const s of layout.sections){ctx.fillStyle=color(s.layer);ctx.fillText(s.layer,s.x,s.y+10);}
  pose={x:cam.x,y:cam.y,k:cam.k,width,height,dpr,pad,calls:state.allCalls};dirty=false;
 };
 const highlight = (id,chosen) => {
  if(center===id&&selected===chosen)return;
  const changed=selected!==chosen;center=id;selected=chosen;dirty=true;
  if(changed&&selected){
   const p=layout.positions.get(selected);selection.setAttribute('aria-label',spec.title(index.nodes.get(selected)));
   selection.innerHTML=`<rect x="${p.x-p.w/2}" y="${p.y-p.h/2}" width="${p.w}" height="${p.h}" fill="transparent" pointer-events="none"/>`;
   canvas.setAttribute('aria-activedescendant','overview-selection');
  }else if(changed){selection.replaceChildren();selection.removeAttribute('aria-label');canvas.removeAttribute('aria-activedescendant');}
  applyCamera();
 };
 return {key,graph,layout,spec,nodeEls:new Map(),edgeEls:[],overview:{index,hit:index.hit,camera,highlight,rasterCount:graph.nodes.length,get visibleCount(){return visibleCount;},get ready(){return Object.values(images).every(img=>img.complete&&img.naturalWidth);},detach:()=>{scene.remove();scene.width=scene.height=0;}}};
}
