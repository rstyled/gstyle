import {type Graph,type Witness} from './model';
export type Position={x:number;y:number;w:number;h:number};
export type Point={x:number;y:number};
export function investigationLayout(g:Graph,w:Witness) {
  const positions=new Map<string,Position>();
  const roles=new Map<string,string>();
  const put=(id:string|undefined,x:number,y:number,role:string)=>{if(!id)return;positions.set(id,{x,y,w:220,h:88});roles.set(id,role);};
  const steps=new Map(w.steps.map(s=>[s.node,s]));
  const read=w.roots[0],join=steps.get(read)?.dependencies[0];
  const writes=steps.get(join)?.dependencies||[];
  const control=g.nodes.find(n=>n.category==='control'&&writes.every(id=>g.edges.some(e=>e.kind==='control'&&e.source===n.id&&e.target===id)));
  if(!control||writes.length!==2)throw new Error('This directed view requires the two-branch fixture witness');
  const branchRead=g.edges.find(e=>e.target===control.id&&e.kind==='data')?.source;
  const input=g.edges.find(e=>e.target===branchRead&&e.subkind==='reaching-definition')?.source;
  put(input,130,100,'Input');put(branchRead,400,100,'Read enabled');put(control.id,670,100,'Choose one branch');
  for(const id of writes) {
    const edge=g.edges.find(e=>e.kind==='control'&&e.target===id)!;
    const x=edge.operand_position===1?940:400;
    const seed=steps.get(id)?.dependencies[0];
    put(seed,x,285,'Non-nil value');put(id,x,435,'Set boxed');
  }
  put(join,670,620,'Join both values');put(read,940,620,'Read boxed');put(w.comparison,1210,620,'Nil check is false');
  const after=g.edges.find(e=>e.source===w.comparison&&e.kind==='data')?.target;
  put(after,1210,790,'Discard result');
  let row=0;
  for(const n of g.nodes.filter(n=>!positions.has(n.id)).sort((a,b)=>(a.source_span?.start||0)-(b.source_span?.start||0))) {
    put(n.id,130,285+row++*150,n.category==='declaration'?'Initial value':n.category==='read'?'Field name':'Literal');
  }
  return {positions,roles,width:1380,height:900,control:control.id,join};
}

function inside(p:Point,n:Position,pad=0) {return p.x>n.x-n.w/2-pad&&p.x<n.x+n.w/2+pad&&p.y>n.y-n.h/2-pad&&p.y<n.y+n.h/2+pad;}
export function segmentHits(a:Point,b:Point,n:Position,pad=0) {
  if(a.x===b.x)return a.x>n.x-n.w/2-pad&&a.x<n.x+n.w/2+pad&&Math.max(a.y,b.y)>n.y-n.h/2-pad&&Math.min(a.y,b.y)<n.y+n.h/2+pad;
  return a.y>n.y-n.h/2-pad&&a.y<n.y+n.h/2+pad&&Math.max(a.x,b.x)>n.x-n.w/2-pad&&Math.min(a.x,b.x)<n.x+n.w/2+pad;
}
// Small obstacle-aware Manhattan router. Ports face the direction of travel;
// grid search rejects every segment that enters a node, including end nodes.
export function routeEdge(a:Position,b:Position,obstacles:Position[],index:number,ports:{vertical?:boolean;horizontal?:boolean;sourceOffset?:number;targetOffset?:number}={}) {
  const horizontal=ports.vertical?false:ports.horizontal?true:Math.abs(b.x-a.x)>Math.abs(b.y-a.y);
  const sign=horizontal?Math.sign(b.x-a.x):Math.sign(b.y-a.y);
  const start={x:a.x+(horizontal?sign*a.w/2:ports.sourceOffset||0),y:a.y+(horizontal?0:sign*a.h/2)};
  const end={x:b.x-(horizontal?sign*b.w/2:-(ports.targetOffset||0)),y:b.y-(horizontal?0:sign*b.h/2)};
  const first={x:start.x+(horizontal?sign*18:0),y:start.y+(horizontal?0:sign*18)};
  const last={x:end.x-(horizontal?sign*18:0),y:end.y-(horizontal?0:sign*18)};
  const xs=[...new Set([first.x,last.x,25,1355,...obstacles.flatMap(n=>[n.x-n.w/2-22-(index%3)*5,n.x+n.w/2+22+(index%3)*5])])].sort((a,b)=>a-b);
  const ys=[...new Set([first.y,last.y,25,875,...obstacles.flatMap(n=>[n.y-n.h/2-22-(index%3)*5,n.y+n.h/2+22+(index%3)*5])])].sort((a,b)=>a-b);
  const path=searchGrid(xs,ys,first,last,obstacles);
  return simplify([start,...path,end]);
}
function searchGrid(xs:number[],ys:number[],start:Point,end:Point,obstacles:Position[]) {
  type State={x:number;y:number;dir:number;cost:number;path:Point[]};
  const queue:State[]=[{x:xs.indexOf(start.x),y:ys.indexOf(start.y),dir:0,cost:0,path:[start]}],best=new Map<string,number>();
  while(queue.length) {
    queue.sort((a,b)=>a.cost-b.cost);const s=queue.shift()!,p={x:xs[s.x],y:ys[s.y]};
    if(p.x===end.x&&p.y===end.y)return s.path;
    for(const [dx,dy,dir] of [[1,0,1],[-1,0,1],[0,1,2],[0,-1,2]]) {
      const x=s.x+dx,y=s.y+dy;if(x<0||y<0||x>=xs.length||y>=ys.length)continue;
      const next={x:xs[x],y:ys[y]};
      if(obstacles.some(n=>inside(next,n,4)||segmentHits(p,next,n,4)))continue;
      const cost=s.cost+Math.abs(next.x-p.x)+Math.abs(next.y-p.y)+(s.dir&&s.dir!==dir?35:0),key=`${x}:${y}:${dir}`;
      if((best.get(key)??Infinity)<=cost)continue;
      best.set(key,cost);queue.push({x,y,dir,cost,path:[...s.path,next]});
    }
  }
  throw new Error('No unobstructed graph route');
}
function simplify(points:Point[]) {
  const out:Point[]=[];
  for(const p of points) {
    const a=out.at(-2),b=out.at(-1);
    if(a&&b&&((a.x===b.x&&b.x===p.x)||(a.y===b.y&&b.y===p.y)))out.pop();
    if(!out.length||out.at(-1)!.x!==p.x||out.at(-1)!.y!==p.y)out.push(p);
  }
  return out;
}
export function routePath(points:Point[]) {return points.map((p,i)=>`${i?'L':'M'}${p.x} ${p.y}`).join(' ');}

export function directedRoutes(g:Graph,w:Witness,layout:ReturnType<typeof investigationLayout>) {
  const seeds=new Set(w.steps.filter(s=>s.seed).map(s=>s.node));
  return new Map(g.edges.map((e,i)=>{
    const a=layout.positions.get(e.source)!,b=layout.positions.get(e.target)!;
    const join=e.target===layout.join,branch=e.kind==='control'&&seeds.has(e.target);
    const ports=join?{vertical:true,targetOffset:a.x<b.x?-43:43}:branch?{vertical:true,sourceOffset:a.x>b.x?-43:43}:seeds.has(e.target)?{horizontal:true}:{};
    return [e.id,routeEdge(a,b,[...layout.positions.values()],i,ports)];
  }));
}
