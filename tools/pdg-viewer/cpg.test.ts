import {expect,test} from 'bun:test';
import {readFileSync} from 'node:fs';
const read=(path:string)=>readFileSync(new URL(path,import.meta.url),'utf8');
const m=new Function(read('model.js')+'\n'+read('cpg.js')+';return {normalizeCPG,cpgView,cpgFindings,cpgLayout}')();
const document=JSON.parse(read('testdata/rstyle-capture/documents/document-000000.cpg.json'));
const report=JSON.parse(read('testdata/rstyle-capture/report.json'));
const payload=()=>({format:'rstyle.cpg-view/1',function:structuredClone(document.unit.functions[0]),files:document.unit.files,producer:document.producer,coverage:document.coverage,flow:report.flow.functions[0].flow,diagnostics:report.check.diagnostics,patterns:report.patterns.candidates});
test('real rstyle CPG retains node IDs, source spans, branch outcomes and all edge layers',()=>{
 const doc=m.normalizeCPG(payload()),graph=m.cpgView(doc,'expressions');
 expect(doc.nodes.map((n:any)=>n.id)).toContain('f0.entry');
 expect(doc.edges.filter((e:any)=>e.kind==='cfg').map((e:any)=>e.label)).toContain('false');
 expect(new Set(graph.edges.map((e:any)=>e.kind))).toEqual(new Set(['ast','cfg','data','control','execution']));
 const update=graph.nodes.find((n:any)=>n.id==='f0.n10');
 expect(update.code).toBe('y++');expect(update.line).toBe(6);expect(update.findings).toEqual(['check:0']);
 expect(graph.findings[0].lines).toEqual([6,7]);
 const condition=graph.nodes.find((n:any)=>n.id==='f0.n2');expect(condition.findings).toEqual([]);
 expect(graph.edges.some((e:any)=>e.kind==='data'&&e.raw[0].source==='f0.n8')).toBe(true);
});
test('statement folding retains real endpoints as evidence and lays out every visible operation',()=>{
 const doc=m.normalizeCPG(payload());
 for(const detail of ['expressions','statements']){
  const graph=m.cpgView(doc,detail),ids=new Set(graph.nodes.map((n:any)=>n.id));
  expect(graph.edges.every((e:any)=>ids.has(e.source)&&ids.has(e.target)&&e.raw.length)).toBe(true);
  const layout=m.cpgLayout(graph.nodes,graph.edges,()=>({w:100,h:46}));
  expect(layout.positions.size).toBe(ids.size);expect(layout.paths).toHaveLength(graph.edges.length);
  expect([...layout.positions.values()].every((p:any)=>Number.isFinite(p.x)&&Number.isFinite(p.y))).toBe(true);
 }
});
test('function-wide checks and patterns do not invent operation links',()=>{
 const input=payload();input.diagnostics=[{...input.diagnostics[0],details:[]}];input.patterns=[{kind:'guard_clause',sites:[{id:'go::example.com/flow.Forward'}],observation:'candidate',possible_refactor:'invert predicate'}];
 const doc=m.normalizeCPG(input),graph=m.cpgView(doc,'expressions');
 expect(graph.findings.map((f:any)=>f.scope)).toEqual(['function','function']);
 expect(graph.nodes.every((n:any)=>n.findings.length===0)).toBe(true);
});
test('CPG boundary rejects unsupported versions, invalid spans and unresolved endpoints',()=>{
 const bad=payload();bad.function.nodes[0].span.end=999999;expect(()=>m.normalizeCPG(bad)).toThrow('span');
 const dangling=payload();dangling.function.cfg.edges[0].target='missing';expect(()=>m.normalizeCPG(dangling)).toThrow('endpoint');
 const duplicate=payload();duplicate.function.nodes.push(duplicate.function.nodes[0]);expect(()=>m.normalizeCPG(duplicate)).toThrow('Duplicate');
 expect(()=>m.normalizeCPG({...payload(),format:'new'})).toThrow('Unsupported');
});
