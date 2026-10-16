import { describe, expect, test } from 'bun:test';
import { createHash } from 'node:crypto';
import data from './data/evidence.json';
import { bytes, lineAt, lineNodes, neighborhood, snippet, validate, upstream,proofRegion,fitBox,zoomBox,resizeBox,type Graph } from './model';
const graph = data.graph as Graph;

describe('pinned source and raw extraction', () => {
  test('every graph endpoint and source span is valid', () => {
    expect(() => validate(graph,data.source)).not.toThrow();
    expect(graph.sources[0].content_identity.value).toBe(createHash('sha256').update(data.source).digest('hex'));
    expect(data.structure.typed).toBe(true);
    for (const n of graph.nodes.filter(n=>!n.synthetic)) {
      const s=n.source_span!;
      expect(data.structure.ast.some(a=>a.start===s.start && a.end===s.end && a.kind===n.original_kind!.value)).toBe(true);
    }
  });
  test('rejects a broken endpoint and out-of-range source', () => {
    expect(()=>validate({...graph,edges:[{...graph.edges[0],target:'missing'}]},data.source)).toThrow('Missing edge endpoint');
    expect(()=>validate({...graph,nodes:[{...graph.nodes[0],source_span:{source_id:data.path,start:-1,end:2}}]},data.source)).toThrow('Invalid source span');
  });
  test('byte locations still map Unicode to the right source line', () => {
    const source='α\nboxed';
    expect(bytes(source,3,8)).toBe('boxed');
    expect(lineAt(source,3)).toBe(2);
  });
});

describe('whole graph investigation',()=>{
  test('solver witness maps to exact supported graph edges, not nearby control edges',()=>{
    const w=data.detector.witness,region=proofRegion(graph,w);
    expect([...region.ids].sort()).toEqual(['n3','n4','n6','n7','n9','n13','n14'].sort());
    expect([...region.edges].sort()).toEqual(['e19','e16','e23','e24','e25','e20'].sort());
    expect([...region.ids]).not.toContain('n10');
    expect(graph.edges.filter(e=>region.edges.has(e.id)).every(e=>e.kind==='data'&&e.evidence.status==='supported')).toBe(true);
    expect(graph.nodes.length).toBe(14);expect(graph.edges.length).toBe(20);
  });
  test('backward slice follows cycles safely and retains comparison input context',()=>{
    const slice=upstream(graph,data.detector.witness.comparison);
    expect([...slice]).toContain('n14');
    expect([...slice]).toContain('n3');expect([...slice]).toContain('n6');
    const cycle={...graph,edges:[...graph.edges,{...graph.edges[0],kind:'data',source:'n9',target:'n13'}]};
    expect(upstream(cycle,'n9').size).toBeLessThanOrEqual(graph.nodes.length);
  });
  test('missing proof nodes or edges fail instead of inventing a link',()=>{
    const w=data.detector.witness;
    expect(()=>proofRegion(graph,{...w,comparison:'missing'})).toThrow('Unknown proof node');
    expect(()=>proofRegion({...graph,edges:[]},w)).toThrow('Missing supported proof edge');
  });
  test('camera zoom preserves the world coordinate under the pointer',()=>{
    const b={x:20,y:40,w:1000,h:600},px=.23,py=.71;
    const z=zoomBox(b,.6,px,py);
    expect(z.x+z.w*px).toBeCloseTo(b.x+b.w*px);
    expect(z.y+z.h*py).toBeCloseTo(b.y+b.h*py);
    const fit=fitBox({x:100,y:200,w:300,h:400},1.5);
    expect(fit.w/fit.h).toBeCloseTo(1.5);
    expect(fit.x).toBeLessThan(100);expect(fit.y).toBeLessThan(200);
  });
  test('transient zero-size resize cannot corrupt the camera',()=>{
    const b={x:20,y:40,w:1000,h:600};
    expect(resizeBox(b,0)).toEqual(b);
    expect(resizeBox(b,Infinity)).toEqual(b);
    expect(resizeBox(b,NaN)).toEqual(b);
    const next=resizeBox(b,2);
    expect(next.y+next.h/2).toBe(b.y+b.h/2);
  });
  test('the explicit repair contract changes disabled behavior and removes the real report',()=>{
    expect(data.replay.false.at(-1)!.result).toBe(false);
    expect(data.repair.runs.false.at(-1)!.result).toBe(true);
    expect(data.repair.runs.true.at(-1)!.result).toBe(false);
    expect(data.repair.remainingFindings).toBe(0);
    expect(data.repair.after).toContain('boxed = nil');
    expect(data.repair.contract).toContain('does not specify');
  });
});

describe('independent expectations for this branch fixture', () => {
  const merge=graph.nodes.find(n=>n.category==='merge')!;
  test('both writes reach the join; the initial nil definition does not', () => {
    const incoming=neighborhood(graph,merge.id,'data').filter(e=>e.target===merge.id);
    expect(incoming.length).toBe(2);
    const definitions=incoming.map(e=>graph.nodes.find(n=>n.id===e.source)!);
    expect(definitions.map(n=>snippet(data.source,n)).sort()).toEqual(['boxed = "queued"','boxed = Parcel{Code: 87}']);
    expect(incoming.every(e=>e.subkind==='reaching-definition' && e.evidence.status==='supported')).toBe(true);
    expect(definitions.some(n=>n.category==='declaration')).toBe(false);
  });
  test('control outcomes distinguish the two writes', () => {
    const writes=graph.nodes.filter(n=>n.category==='assignment' && snippet(data.source,n).startsWith('boxed ='));
    expect(writes.length).toBe(2);
    expect(writes.map(n=>graph.edges.find(e=>e.target===n.id&&e.kind==='control')!.operand_position).sort()).toEqual([0,1]);
    expect(graph.edges.filter(e=>e.kind==='control').every(e=>e.evidence.status==='approximated')).toBe(true);
  });
  test('Go CFG has two branches that converge before the comparison', () => {
    const blocks=data.structure.cfg;
    const branch=blocks.find(b=>b.successors.length===2)!;
    expect(branch).toBeDefined();
    const paths=branch.successors.map(id=>blocks.find(b=>b.id===id)!);
    expect(paths[0].successors.length).toBe(1);
    expect(paths[0].successors).toEqual(paths[1].successors);
    const join=blocks.find(b=>b.id===paths[0].successors[0])!;
    expect(join.nodes.some(n=>bytes(data.source,n.start,n.end).startsWith('_ = boxed == nil'))).toBe(true);
  });
  test('Go executes one write per input and both comparisons are false', () => {
    for (const [input,events] of Object.entries(data.replay)) {
      expect(events.length).toBe(4);
      expect(events[0].value).toBe(null);
      expect(events[0].result).toBe(true);
      expect(events[1].result).toBe(input==='true');
      expect(events[2].result).toBe(false);
      expect(events[3].result).toBe(false);
      expect(events[3].value).toEqual(input==='true'?{Code:87}:'queued');
      expect(lineNodes(graph,data.source,lineAt(data.source,events[2].start)).some(n=>n.category==='assignment')).toBe(true);
    }
    expect(data.finding.Kind).toBe('typednil');
    expect(data.finding.Sites[0].Line).toBe(lineAt(data.source,merge.source_span!.start));
  });
});
