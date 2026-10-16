export type Span = {start: number; end: number; source_id: string};
export type Node = { id: string; category: string; source_span?: Span; synthetic: boolean; attributes: any; original_kind?: {value: string}; evidence: any };
export type Edge = {id: string; source: string; target: string; kind: string; subkind: string; operand_position?: number; symbol_id?: string; evidence: any};
export type Graph = {nodes: Node[]; edges: Edge[]; function: any; symbols: any[]; sources: any[]; extraction_evidence: any};
export const bytes = (source: string, start: number, end: number) => new TextDecoder().decode(new TextEncoder().encode(source).slice(start, end));
export const lineAt = (source: string, offset: number) => bytes(source, 0, offset).split('\n').length;
export function snippet(source: string, node: Node) {
  const s = node.source_span;
  if (!s) return 'No source location';
  return bytes(source, s.start, s.end);
}
export function lineNodes(g: Graph, source: string, line: number) {
  return g.nodes.filter(n => n.source_span && lineAt(source, n.source_span.start) === line)
    .sort((a,b) => {
      const priority:Record<string,number>={control:0,assignment:1,declaration:2,formal_input:3};
      const rank=(priority[a.category]??4)-(priority[b.category]??4);
      return rank || (a.source_span!.end-a.source_span!.start)-(b.source_span!.end-b.source_span!.start);
    });
}
export function neighborhood(g: Graph, id: string, kind: string) {
  return g.edges.filter(e => e.kind === kind && (e.source === id || e.target === id));
}
export function related(g: Graph, id: string, kind: string) {
  const ids = new Set([id]);
  for (const e of neighborhood(g,id,kind)) { ids.add(e.source); ids.add(e.target); }
  return ids;
}
export function validate(g: Graph, source: string) {
  const ids = new Set(g.nodes.map(n => n.id));
  if (ids.size !== g.nodes.length) throw new Error('Duplicate graph nodes');
  const size = new TextEncoder().encode(source).length;
  for (const n of g.nodes) {
    const s = n.source_span;
    if (s && (s.start < 0 || s.end < s.start || s.end > size)) throw new Error(`Invalid source span: ${n.id}`);
  }
  for (const e of g.edges) {
    if (!ids.has(e.source) || !ids.has(e.target)) throw new Error(`Missing edge endpoint: ${e.id}`);
  }
}

export function upstream(g: Graph, start: string) {
  const ids=new Set([start]),queue=[start];
  for(let i=0;i<queue.length;i++) {
    for(const e of g.edges.filter(e=>e.kind==='data'&&e.target===queue[i])) {
      if(ids.has(e.source))continue;
      ids.add(e.source);queue.push(e.source);
    }
  }
  return ids;
}

export type Witness={comparison:string;operand:number;roots:string[];steps:{node:string;seed:boolean;dependencies:string[]}[]};
export function proofRegion(g:Graph,w:Witness) {
  const ids=new Set([w.comparison,...w.steps.map(s=>s.node)]);
  const graphIDs=new Set(g.nodes.map(n=>n.id));
  for(const id of ids)if(!graphIDs.has(id))throw new Error(`Unknown proof node: ${id}`);
  const pairs=new Set(w.roots.map(id=>`${id}:${w.comparison}`));
  for(const s of w.steps)for(const dep of s.dependencies){
    if(!ids.has(dep))throw new Error(`Missing proof dependency: ${dep}`);
    pairs.add(`${dep}:${s.node}`);
  }
  const matches=g.edges.filter(e=>e.kind==='data'&&e.evidence.status==='supported'&&pairs.has(`${e.source}:${e.target}`));
  for(const pair of pairs)if(!matches.some(e=>`${e.source}:${e.target}`===pair))throw new Error(`Missing supported proof edge: ${pair}`);
  const edges=new Set(matches.map(e=>e.id));
  return {ids,edges};
}

export type Box={x:number;y:number;w:number;h:number};
export function zoomBox(box:Box,factor:number,px=.5,py=.5):Box {
  const w=box.w*factor,h=box.h*factor;
  return {x:box.x+(box.w-w)*px,y:box.y+(box.h-h)*py,w,h};
}
export function fitBox(bounds:Box,aspect:number,pad=60):Box {
  const w=Math.max(bounds.w+pad*2,(bounds.h+pad*2)*aspect),h=w/aspect;
  return {x:bounds.x+bounds.w/2-w/2,y:bounds.y+bounds.h/2-h/2,w,h};
}

export function resizeBox(box:Box,aspect:number):Box {
  if(!Number.isFinite(aspect)||aspect<=0)return {...box};
  const h=box.w/aspect;
  return {...box,h,y:box.y+box.h/2-h/2};
}

