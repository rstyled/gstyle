import {expect,test} from 'bun:test';
import {readFileSync} from 'node:fs';
const layout=new Function(readFileSync(new URL('./graph-layout.js',import.meta.url),'utf8')+';return {graphLayout,routePath}')();
const ids=(...names:string[])=>names.map(id=>({id}));
test('ranks follow dependencies and cycles keep their edges',()=>{
 const edges=[{source:'a',target:'b'},{source:'b',target:'c'},{source:'c',target:'b'},{source:'c',target:'d'}];
 const {positions,routes}=layout.graphLayout(ids('a','b','c','d'),edges);
 const x=(id:string)=>positions.get(id).x;
 expect(x('a')).toBeLessThan(x('b'));expect(x('b')).toBeLessThan(x('c'));expect(x('c')).toBeLessThan(x('d'));
 expect(routes.every((r:any)=>r&&r.length>=2)).toBe(true);
 const back=routes[2];expect(back[0].x).toBeGreaterThan(back.at(-1).x);
 expect(back[0].x).toBe(x('c')-90);expect(back.at(-1).x).toBe(x('b')+90);
});
test('long edges route through every column they cross',()=>{
 const {routes}=layout.graphLayout(ids('a','b','c','d'),[{source:'a',target:'b'},{source:'b',target:'c'},{source:'c',target:'d'},{source:'a',target:'d'}]);
 expect(routes[3]).toHaveLength(6);
 expect(layout.routePath(routes[3]).startsWith('M')).toBe(true);
});
test('nodes in a column never overlap',()=>{
 const nodes=ids('root',...Array.from({length:30},(_,i)=>'leaf'+i));
 const {positions}=layout.graphLayout(nodes,nodes.slice(1).map(n=>({source:'root',target:n.id})));
 const ys=nodes.slice(1).map(n=>positions.get(n.id).y).sort((a,b)=>a-b);
 for(let i=1;i<ys.length;i++)expect(ys[i]-ys[i-1]).toBeGreaterThanOrEqual(40);
});
test('isolated nodes pack into the tray, self loops have no route',()=>{
 const result=layout.graphLayout(ids('a','b','lonely','self'),[{source:'a',target:'b'},{source:'self',target:'self'}]);
 expect(result.tray).not.toBeNull();
 expect(result.positions.get('lonely').y).toBeGreaterThan(result.positions.get('a').y);
 expect(result.positions.size).toBe(4);expect(result.routes[1]).toBeNull();
});
test('deep graphs avoid recursive stack limits',()=>{
 const nodes=Array.from({length:12000},(_,i)=>({id:String(i)}));
 const edges=nodes.slice(1).map((n,i)=>({source:String(i),target:n.id}));
 expect(layout.graphLayout(nodes,edges).positions.size).toBe(12000);
});
test('package map retains every package and gives import edges exact endpoints',()=>{
 const map = new Function(readFileSync(new URL('./graph-layout.js',import.meta.url),'utf8')+';return packageLayout')();
 const nodes = [{id:'a',layer:'application'},{id:'b',layer:'domain'},{id:'c',layer:'domain'},{id:'isolated',layer:'root'}];
 const edges = [{source:'a',target:'b'},{source:'b',target:'c'}];
 const result=map(nodes,edges);
 expect(result.positions.size).toBe(nodes.length);
 for(let i=0;i<edges.length;i++){
  const a=result.positions.get(edges[i].source), b=result.positions.get(edges[i].target), points=result.routes[i];
  expect(points[0].y).toBe(a.y);expect(points.at(-1).y).toBe(b.y);
  expect(Math.abs(points[0].x-a.x)).toBe(a.w/2);
  expect(Math.abs(points.at(-1).x-b.x)).toBe(b.w/2);
 }
 const p=result.positions.get('b'),q=result.positions.get('c');
 expect(q.y-p.y).toBeGreaterThan(p.h);
});

test('dependence layout flows down with normal node sizes and preserves every edge',()=>{
 const down = new Function(readFileSync(new URL('./graph-layout.js',import.meta.url),'utf8')+';return dependenceLayout')();
 const result=down(ids('a','b','alone'),[{source:'a',target:'b'}],()=>({w:200,h:46}));
 const a=result.positions.get('a'),b=result.positions.get('b');
 expect(b.y).toBeGreaterThan(a.y);
 expect(a.w).toBe(200);expect(a.h).toBe(46);
 expect(result.routes[0][0].y).toBe(a.y+23);
 expect(result.routes[0].at(-1).y).toBe(b.y-23);
 expect(result.positions.size).toBe(3);
});
