import {expect,test} from 'bun:test';
import {readFileSync} from 'node:fs';
const helpers=new Function(readFileSync(new URL('./graph-layout.js',import.meta.url),'utf8')+readFileSync(new URL('./program-graph.js',import.meta.url),'utf8')+';return {buildProgramGraph,entrypointLayout,programEdgePlan}')();
const build=helpers.buildProgramGraph;
test('whole program retains disconnected functions and unresolved endpoints',()=>{
 const index={symbols:[
  {id:'a',name:'A',package:'p',calls:[{spelling:'B',target:'b'},{spelling:'remote.Do'},{spelling:'remote.Do'}]},
  {id:'b',name:'B',package:'p',calls:[{spelling:'B',target:'b'}]},
  {id:'c',name:'Unused',package:'p',calls:[]}
 ]};
 const graph=build(index);
 expect(graph.nodes.filter(n=>n.kind==='function').map(n=>n.id)).toEqual(['a','b','c']);
 expect(graph.nodes.filter(n=>n.kind==='unresolved')).toHaveLength(1);
 expect(graph.edges).toHaveLength(3);
 expect(graph.edges.find(e=>e.kind==='unresolved').count).toBe(2);
 expect(graph.edges.some(e=>e.source==='b'&&e.target==='b')).toBe(true);
});
test('unknown endpoints remain scoped to their calling package',()=>{
 const graph=build({symbols:[{id:'a',package:'a',calls:[{spelling:'Read'}]},{id:'b',package:'b',calls:[{spelling:'Read'}]}]});
 expect(graph.nodes).toHaveLength(4);
 expect(graph.edges[0].target).not.toBe(graph.edges[1].target);
});

test('entrypoint layout keeps all nodes and puts call boundaries outside the root flow',()=>{
 const graph=build({packages:[{id:'app',imports:['lib']},{id:'lib',imports:[]}],symbols:[
  {id:'main',name:'main',package:'app',role:'entrypoint',calls:[{spelling:'Run',target:'run'}]},
  {id:'run',name:'Run',package:'lib',calls:[{spelling:'main',target:'main'},{spelling:'remote.Do'}]},
  {id:'init',name:'init',package:'lib',role:'initializer',calls:[{spelling:'Setup',target:'setup'}]},
  {id:'setup',name:'Setup',package:'lib',calls:[]},
  {id:'unused',name:'Other',package:'lib',calls:[]},
  {id:'second',name:'main',package:'other',role:'entrypoint',calls:[]}
 ]});
 const layout=helpers.entrypointLayout(graph,'main');
 expect(layout.positions.size).toBe(graph.nodes.length);
 expect(layout.reached.has('run')).toBe(true);
 expect(layout.reached.has('setup')).toBe(true);
 expect(layout.reached.has('unused')).toBe(false);
 expect(layout.reached.has('second')).toBe(false);
 expect(layout.positions.get('main').y).toBeLessThan(layout.positions.get('run').y);
 const boundary=graph.nodes.find(n=>n.kind==='unresolved');
 expect(layout.positions.get(boundary.id).y).toBeGreaterThan(layout.positions.get('run').y);
 expect(layout.initializers.map(n=>n.id)).toEqual(['init']);
 const other=helpers.entrypointLayout(graph,'second');expect(other.reached.has('main')).toBe(false);
});
test('libraries do not infer an entrypoint from indegree or coordinates',()=>{
 const graph=build({symbols:[{id:'f',name:'Function',package:'lib',calls:[]}]});
 const layout=helpers.entrypointLayout(graph,'f');
 expect(graph.entrypoints).toHaveLength(0);expect(layout.entry).toBeUndefined();expect(layout.positions.size).toBe(1);
});

test('entrypoint selection does not light up every edge in its reachable subgraph',()=>{
 const graph=build({symbols:[
  {id:'main',package:'app',role:'entrypoint',calls:[{spelling:'Run',target:'run'}]},
  {id:'run',package:'lib',calls:[{spelling:'Work',target:'work'}]},
  {id:'work',package:'lib',calls:[]}
 ]});
 const plan=helpers.programEdgePlan(graph,'main');
 expect(plan.detailed).toHaveLength(1);expect(plan.detailed[0].source).toBe('main');
 expect(helpers.programEdgePlan(graph,'main',true).detailed).toHaveLength(2);
 expect(graph.nodes).toHaveLength(3);expect(graph.edges).toHaveLength(2);
});
test('overview bundles aggregate package links without losing call counts',()=>{
 const graph=build({symbols:[
  {id:'a',package:'app',calls:[{spelling:'B',target:'b'},{spelling:'B',target:'b'},{spelling:'C',target:'c'}]},
  {id:'b',package:'lib',calls:[]},{id:'c',package:'lib',calls:[]}
 ]});
 const plan=helpers.programEdgePlan(graph,'');
 expect(plan.detailed).toHaveLength(0);expect(plan.bundles).toEqual([{source:'app',target:'lib',count:3}]);
 expect(graph.edges).toHaveLength(2);
});
