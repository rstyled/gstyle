import {test,expect} from 'bun:test';
import data from './data/evidence.json';
import {investigationLayout,directedRoutes,segmentHits} from './directed-layout';
import {proofRegion,type Graph} from './model';
const graph=data.graph as Graph,witness=data.detector.witness;
const layout=investigationLayout(graph,witness);

test('directed layout retains every exported operation in separate, non-overlapping cards',()=>{
  expect([...layout.positions.keys()].sort()).toEqual(graph.nodes.map(n=>n.id).sort());
  const ps=[...layout.positions.values()];
  for(let i=0;i<ps.length;i++)for(let j=i+1;j<ps.length;j++)expect(Math.abs(ps[i].x-ps[j].x)>=(ps[i].w+ps[j].w)/2||Math.abs(ps[i].y-ps[j].y)>=(ps[i].h+ps[j].h)/2).toBe(true);
});
test('branch values flow down to their writes; both writes precede the join; comparison reads left to right',()=>{
  const step=new Map(witness.steps.map(s=>[s.node,s]));
  const read=witness.roots[0],join=step.get(read)!.dependencies[0];
  for(const write of step.get(join)!.dependencies){
    const seed=step.get(write)!.dependencies[0],a=layout.positions.get(seed)!,b=layout.positions.get(write)!;
    expect(a.x).toBe(b.x);expect(a.y).toBeLessThan(b.y);expect(b.y).toBeLessThan(layout.positions.get(join)!.y);
  }
  expect(layout.positions.get(join)!.x).toBeLessThan(layout.positions.get(read)!.x);
  expect(layout.positions.get(read)!.x).toBeLessThan(layout.positions.get(witness.comparison)!.x);
});
test('every actual edge routes around every card and lands on distinct join ports',()=>{
  const routes=directedRoutes(graph,witness,layout);
  expect([...routes.keys()].sort()).toEqual(graph.edges.map(e=>e.id).sort());
  for(const [id,points]of routes){
    for(let i=1;i<points.length;i++){
      expect(points[i].x===points[i-1].x||points[i].y===points[i-1].y).toBe(true);
      for(const [node,n]of layout.positions)expect(segmentHits(points[i-1],points[i],n),`${id} crosses ${node}`).toBe(false);
    }
  }
  const incoming=graph.edges.filter(e=>e.target===layout.join).map(e=>routes.get(e.id)!.at(-1)!);
  expect(incoming.length).toBe(2);expect(incoming[0].x).not.toBe(incoming[1].x);
  const join=layout.positions.get(layout.join)!;
  expect(incoming.every(p=>p.y===join.y-join.h/2)).toBe(true);
  expect(proofRegion(graph,witness).edges.size).toBe(6);
});
