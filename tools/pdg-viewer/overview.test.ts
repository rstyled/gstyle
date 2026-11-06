import {expect, test} from 'bun:test';
import {readFileSync} from 'node:fs';
const {overviewIndex} = new Function(readFileSync(new URL('./overview.js',import.meta.url),'utf8')+';return {overviewIndex}')();
const graph = {nodes:[{id:'a'},{id:'b'},{id:'isolated'}],edges:[{source:'a',target:'b'},{source:'b',target:'a'},{source:'a',target:'a'}]};
const layout = {positions:new Map([['a',{x:255,y:100,w:120,h:20}],['b',{x:400,y:100,w:120,h:20}],['isolated',{x:700,y:200,w:120,h:20}]]),frames:[{id:'pkg',x:0,y:0,w:900,h:300}]};
test('hit lookup covers both sides of bucket boundaries and empty package space',()=>{
 const index=overviewIndex(graph,layout);
 expect(index.hit(200,100)).toEqual({id:'a'});
 expect(index.hit(300,100)).toEqual({id:'a'});
 expect(index.hit(400,100)).toEqual({id:'b'});
 expect(index.hit(700,200)).toEqual({id:'isolated'});
 expect(index.hit(20,20)).toEqual({pkg:'pkg'});
 expect(index.hit(-20,20)).toBeNull();
 expect([...index.query({x:190,y:85,w:140,h:30})]).toEqual(['a']);
});
test('direct call indexes preserve cycles and record a self-call once',()=>{
 const index=overviewIndex(graph,layout);
 expect(index.incident.get('a')).toEqual([0,1,2]);
 expect(index.incident.get('b')).toEqual([0,1]);
 expect([...index.adjacent.get('a')]).toEqual(['a','b']);
 expect([...index.adjacent.get('isolated')]).toEqual(['isolated']);
 expect(index.nodes.size).toBe(graph.nodes.length);
});
test('spatial queries match geometric intersections without changing the full model',()=>{
 const nodes=Array.from({length:6000},(_,i)=>({id:String(i)}));
 const positions=new Map(nodes.map((n,i)=>[n.id,{x:(i%60)*132-500,y:Math.floor(i/60)*28-500,w:120,h:20}]));
 const index=overviewIndex({nodes,edges:[]},{positions,frames:[]});
 for(let i=0;i<30;i++){
  const bounds={x:i*137-600,y:i*53-600,w:570,h:240};
  const expected=nodes.filter(n=>{const p=positions.get(n.id)!;return p.x+60>=bounds.x&&p.x-60<=bounds.x+bounds.w&&p.y+10>=bounds.y&&p.y-10<=bounds.y+bounds.h;}).map(n=>n.id).sort();
  expect([...index.query(bounds)].sort()).toEqual(expected);
 }
 expect(index.nodes.size).toBe(6000);
});
