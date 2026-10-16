import {test,expect} from 'bun:test';
import data from './data/evidence.json';
import {explanationSteps} from './explanation';
import {type Graph} from './model';
const g=data.graph as Graph,w=data.detector.witness;
test('walkthrough is grounded in the actual branch and solver witness',()=>{
  const steps=explanationSteps(g,w),ids=new Set(g.nodes.map(n=>n.id));
  expect(steps.length).toBe(4);
  for(const s of steps){expect(s.nodes).toContain(s.selected);for(const id of s.nodes)expect(ids.has(id)).toBe(true);}
  expect(g.nodes.find(n=>n.id===steps[0].selected)!.category).toBe('control');
  const writes=steps[1].nodes.filter(id=>g.nodes.find(n=>n.id===id)!.category==='assignment');
  expect(writes.length).toBe(2);
  for(const write of writes)expect(w.steps.some(s=>s.node===write&&!s.seed&&s.dependencies.length===1)).toBe(true);
  expect(g.nodes.find(n=>n.id===steps[2].selected)!.category).toBe('merge');
  expect(steps[3].selected).toBe(w.comparison);
  expect(steps[0].body).toContain('do not both run');
  expect(steps[2].body).toContain('does not combine');
  expect(steps[3].body).toContain('does not tell us');
});
