import { mkdtempSync, readFileSync, writeFileSync, rmSync, mkdirSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { createHash } from 'node:crypto';
import { spawnSync } from 'node:child_process';

const here = import.meta.dir;
const root = resolve(here, '../..');
function run(args: string[]) {
  const result = spawnSync(args[0], args.slice(1), { cwd: root, encoding: 'utf8', maxBuffer: 32 << 20 });
  if (result.status !== 0) throw new Error(`${args.join(' ')}\n${result.stderr}`);
  return result.stdout;
}
const path = 'adapters/gopatterns/testdata/interface-nil/sample.go';
const source = readFileSync(join(root, path), 'utf8');
const revision = run(['git', 'rev-parse', 'HEAD']).trim();
if (run(['git', 'show', `HEAD:${path}`]) !== source) throw new Error('Fixture differs from pinned revision');
const graphs = JSON.parse(run(['go', 'run', './cmd/gstyle', 'patterns', '--format', 'pdg-json', './adapters/gopatterns/testdata/interface-nil']));
const graph = graphs.find((g: any) => g.function.name.value === 'branchAssignments');
if (!graph) throw new Error('Missing fixture graph');
if (graph.sources[0].content_identity.value !== createHash('sha256').update(source).digest('hex')) throw new Error('Graph/source hash mismatch');
const report = JSON.parse(run(['go', 'run', './cmd/gstyle', 'patterns', '--format', 'json', './adapters/gopatterns/testdata/interface-nil']));
const finding = report.candidates.find((c: any) => c.Kind === 'typednil' && c.Sites.some((s: any) => s.ID === graph.function.id));
if (!finding) throw new Error('Expected typednil finding is missing');
const solver = JSON.parse(run(['go','run','./tools/code-lens/detector']));
const graphIndex = solver.functions.indexOf(graph.function.id);
const proof = solver.proofs.find((p:any)=>p.comparison.graph===graphIndex);
if (!proof) throw new Error('Actual detector proof is missing');
const nodeID = (site:any) => {
  if(site.graph!==graphIndex) throw new Error('This view requires a function-local proof');
  return graph.nodes[site.node-1].id;
};
const witness = {comparison:nodeID(proof.comparison),operand:proof.operand,roots:proof.roots.map(nodeID),steps:proof.steps.map((s:any)=>({node:nodeID(s.site),seed:s.seed,dependencies:s.dependencies.map(nodeID)}))};
const detectorPath='domain/interfacenil/proof.go';
const detectorSource=readFileSync(join(root,detectorPath),'utf8');
const detector={path:detectorPath,source:detectorSource,sha256:createHash('sha256').update(detectorSource).digest('hex'),witness};
const structure = JSON.parse(run(['go', 'run', './tools/code-lens/evidence', path]));
const temp = mkdtempSync(join(tmpdir(), 'code-lens-'));
let replay;
try {
  writeFileSync(join(temp, 'main.go'), structure.replay);
  replay = JSON.parse(run(['go', 'run', join(temp, 'main.go')]));
} finally { rmSync(temp, {recursive: true, force: true}); }
delete structure.replay;
// A source-level experiment checks a stated contract; it is not an automatic fix.
const repairTemp=mkdtempSync(join(here,'data/repair-'));
let repair;
try {
  const before=source.slice(graph.function.source_span.start,graph.function.source_span.end);
  const after=before.replace('boxed = "queued"','boxed = nil');
  if(before===after) throw new Error('Repair experiment no longer matches the fixture');
  const changed=source.slice(0,graph.function.source_span.start)+after+source.slice(graph.function.source_span.end);
  const changedPath=join(repairTemp,'sample.go');
  writeFileSync(changedPath,changed);
  const candidateReport=JSON.parse(run(['go','run','./cmd/gstyle','patterns','--format','json',repairTemp]));
  const remaining=candidateReport.candidates.filter((c:any)=>c.Kind==='typednil'&&c.Sites.some((s:any)=>s.Name.endsWith('.branchAssignments')));
  if(remaining.length) throw new Error('Repair experiment still triggers the finding');
  const changedStructure=JSON.parse(run(['go','run','./tools/code-lens/evidence',changedPath]));
  const runTemp=mkdtempSync(join(tmpdir(),'lens-repair-replay-'));
  try {
    writeFileSync(join(runTemp,'main.go'),changedStructure.replay);
    const runs=JSON.parse(run(['go','run',join(runTemp,'main.go')]));
    if(runs.false.at(-1).result!==true || runs.true.at(-1).result!==false) throw new Error('Repair violates its stated contract');
    repair={before,after,runs,remainingFindings:remaining.length,contract:'For this experiment only: disabled means no value (nil); enabled means a Parcel. The original fixture does not specify this product intent.'};
  } finally {rmSync(runTemp,{recursive:true,force:true});}
} finally {rmSync(repairTemp,{recursive:true,force:true});}
const data = { revision, path, source, graph, finding, structure, replay,detector,repair };
mkdirSync(join(here, 'data'), { recursive: true });
writeFileSync(join(here, 'data/evidence.json'), JSON.stringify(data, null, 2));
const bundle = await Bun.build({ entrypoints: [join(here, 'runtime.ts')], target: 'browser', minify: false });
if (!bundle.success) throw new Error(String(bundle.logs));
const runtime = await bundle.outputs[0].text();
const template = readFileSync(join(here, 'page.html'), 'utf8');
const json = JSON.stringify(data).replaceAll('<', '\\u003c');
const html = template.replace('__EVIDENCE__', json).replace('/* RUNTIME */', runtime.replaceAll('</script', '<\\/script'));
writeFileSync(join(here, 'index.html'), html);
console.log(`Built source lens at ${here}/index.html · ${graph.nodes.length} nodes · ${graph.edges.length} edges · ${revision.slice(0, 8)}`);
