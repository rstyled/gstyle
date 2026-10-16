import {expect,test} from 'bun:test';
import {readFileSync} from 'node:fs';
const layout=new Function(readFileSync(new URL('./graph-layout.js',import.meta.url),'utf8')+';return {graphLayout,graphComponents}')();
test('layers follow dependencies and retain cycles',()=>{
 const nodes=['a','b','c','d'].map(id=>({id}));
 const edges=[{source:'a',target:'b'},{source:'b',target:'c'},{source:'c',target:'b'},{source:'c',target:'d'}];
 const result=layout.graphLayout(nodes,edges);
 expect(result.positions.get('b').y).toBe(result.positions.get('c').y);
 expect(result.positions.get('a').y).toBeLessThan(result.positions.get('b').y);
 expect(result.positions.get('d').y).toBeGreaterThan(result.positions.get('c').y);
 expect(result.positions.size).toBe(4);
});
test('deep graphs avoid recursive stack limits',()=>{
 const nodes=Array.from({length:12000},(_,i)=>({id:String(i)}));
 const edges=nodes.slice(1).map((n,i)=>({source:String(i),target:n.id}));
 expect(layout.graphLayout(nodes,edges).positions.size).toBe(12000);
});
