import {type Graph,type Witness} from './model';
export type ExplanationStep={title:string;body:string;selected:string;nodes:string[]};
export function explanationSteps(g:Graph,w:Witness):ExplanationStep[] {
  const steps=new Map(w.steps.map(s=>[s.node,s]));
  const read=w.roots[0],join=steps.get(read)?.dependencies[0];
  const writes=steps.get(join)?.dependencies||[];
  const control=g.nodes.find(n=>n.category==='control'&&writes.every(id=>g.edges.some(e=>e.kind==='control'&&e.source===n.id&&e.target===id)));
  if(!control||writes.length!==2)throw new Error('Expected two branch assignments in the solver witness');
  const seeds=writes.flatMap(id=>steps.get(id)?.dependencies||[]);
  return [
    {title:'1. enabled chooses one branch',body:'If enabled is true, the left branch runs. If it is false, the right branch runs. They do not both run in one call.',selected:control.id,nodes:[control.id,...seeds]},
    {title:'2. Either branch puts a value in boxed',body:'The left branch stores Parcel{Code: 87}. The right branch stores "queued". Both branches store a concrete value in boxed. Neither leaves it as an empty interface (nil).',selected:writes[0],nodes:[...seeds,...writes]},
    {title:'3. The later code can receive either value',body:'These arrows follow the two possible definitions of boxed. The join is a graph operation: it means "this read can get the value from either branch". It does not combine the values.',selected:join,nodes:[...writes,join,read]},
    {title:'4. boxed == nil is always false here',body:'The check asks whether boxed is an empty interface. Both branches stored a value and its type, so boxed == nil cannot be true. That is the finding gstyle reports; the fixture does not tell us what the author intended instead.',selected:w.comparison,nodes:[join,read,w.comparison]},
  ];
}
