import {expect,test} from 'bun:test';
import {mkdtempSync,readFileSync,existsSync,rmSync,writeFileSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
const chromium=Bun.which('chromium');
test.skipIf(!chromium)('real CPG evidence renders, filters graph layers and links source lines',async()=>{
 const asset=(name:string)=>readFileSync(new URL(name,import.meta.url),'utf8');
 const scripts=['graph-layout.js','model.js','overview.js','cpg.js','app.js'].map(asset).join('\n');
 const html=asset('app.html').replace('__PDG_JSON__','null').replace('/* SCRIPTS */',scripts);
 const cpg=JSON.parse(asset('testdata/rstyle-capture/documents/document-000000.cpg.json'));
 const report=JSON.parse(asset('testdata/rstyle-capture/report.json'));
 const payload={format:'rstyle.cpg-view/1',function:cpg.unit.functions[0],files:cpg.unit.files,producer:cpg.producer,coverage:cpg.coverage,flow:report.flow.functions[0].flow,diagnostics:report.check.diagnostics,patterns:[]};
 const index={root:'/fixture',module:'example.com/flow',lines:13,packages:[{id:'example.com/flow',files:1,lines:13,imports:[]}],symbols:[{id:'forward',name:'Forward',package:'example.com/flow',file:'flow.go',line:3,end_line:11,start:14,end:104,call_resolution:'typed',calls:[]}]};
 const server=Bun.serve({port:0,hostname:'127.0.0.1',fetch(request){const path=new URL(request.url).pathname;if(path==='/api/index')return Response.json(index);if(path==='/api/pdg')return Response.json([payload]);return new Response(html,{headers:{'content-type':'text/html'}});}});
 const profile=mkdtempSync(join(tmpdir(),'cpg-browser-'));
 const browser=Bun.spawn([chromium!,'--headless','--no-sandbox','--disable-gpu','--remote-debugging-port=0',`--user-data-dir=${profile}`,'about:blank'],{stdout:'ignore',stderr:'ignore'});
 let socket:WebSocket|undefined;
 try{
  const deadline=Date.now()+10000,portFile=join(profile,'DevToolsActivePort');while(!existsSync(portFile)&&Date.now()<deadline)await Bun.sleep(25);
  if(!existsSync(portFile))throw Error('Chromium does not expose its test port');
  const port=readFileSync(portFile,'utf8').split('\n')[0];
  const tab=await(await fetch(`http://127.0.0.1:${port}/json/new?about:blank`,{method:'PUT'})).json() as {webSocketDebuggerUrl:string};
  socket=new WebSocket(tab.webSocketDebuggerUrl);await new Promise<void>((resolve,reject)=>{socket!.onopen=()=>resolve();socket!.onerror=reject;});
  const pending=new Map<number,{resolve:(value:any)=>void,reject:(error:any)=>void}>();let sequence=0;
  socket.onmessage=e=>{const reply=JSON.parse(String(e.data)),task=pending.get(reply.id);if(!task)return;pending.delete(reply.id);reply.error?task.reject(reply.error):task.resolve(reply.result);};
  const send=(method:string,params:object={})=>new Promise<any>((resolve,reject)=>{const id=++sequence;pending.set(id,{resolve,reject});socket!.send(JSON.stringify({id,method,params}));});
  const evaluate=async(expression:string)=>{const r=await send('Runtime.evaluate',{expression,returnByValue:true,awaitPromise:true});if(r.exceptionDetails)throw Error(JSON.stringify(r.exceptionDetails));return r.result.value;};
  await send('Page.enable');await send('Emulation.setDeviceMetricsOverride',{width:1600,height:1000,deviceScaleFactor:1,mobile:false});await send('Page.navigate',{url:`${server.url}#/pdg/forward`});
  const ready=Date.now()+10000;while(Date.now()<ready){if(await evaluate("typeof state!=='undefined'&&state.view?.doc?.cpg"))break;await Bun.sleep(25);}
  expect(await evaluate("!!state.view?.doc?.cpg")).toBe(true);
  expect(await evaluate("document.querySelector('#cpg-findings').textContent.includes('mutation_per_target')")).toBe(true);
  expect(await evaluate("[...document.querySelectorAll('#canvas .edge')].some(n=>n.classList.contains('ast'))&&[...document.querySelectorAll('#canvas .edge')].some(n=>n.classList.contains('cfg'))")).toBe(true);
  expect(await evaluate("(()=>{document.querySelector('[data-finding]').click();return [...document.querySelectorAll('#pdg-source .ln.sel')].map(n=>Number(n.dataset.n))})()")).toEqual([6,7]);
  expect(await evaluate("state.view.evidenceIDs.has('f0.n10')&&state.view.evidenceIDs.has('f0.n14')")).toBe(true);
  await evaluate("[...document.querySelectorAll('#pdg-source .ln')].find(n=>n.dataset.n==='6').click()");
  expect(await evaluate("state.view.evidenceIDs===null&&state.selected!==null")).toBe(true);
  await evaluate("document.querySelector('[data-kind=ast]').click()");await Bun.sleep(100);
  expect(await evaluate("document.querySelectorAll('#canvas .edge.ast').length")).toBe(0);
  await evaluate("document.querySelector('[data-kind=ast]').click()");await Bun.sleep(100);
  if(process.env.RSTYLE_VIEWER_SCREENSHOT){const shot=await send('Page.captureScreenshot',{format:'png'});writeFileSync(process.env.RSTYLE_VIEWER_SCREENSHOT,Buffer.from(shot.data,'base64'));}
  await send('Emulation.setDeviceMetricsOverride',{width:390,height:844,deviceScaleFactor:1,mobile:true});await Bun.sleep(100);
  expect(await evaluate("document.documentElement.scrollWidth<=innerWidth")).toBe(true);
 }finally{socket?.close();browser.kill();await browser.exited;await server.stop(true);rmSync(profile,{recursive:true,force:true});}
},30000);
