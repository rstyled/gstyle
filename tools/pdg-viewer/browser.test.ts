import {expect, test} from 'bun:test';
import {mkdtempSync, readFileSync, existsSync, rmSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';

// Keep a real browser check for the dense close-up path. Pure layout tests do
// not catch thousands of DOM elements being replaced during pointer movement.
const chromium = Bun.which('chromium');
test.skipIf(!chromium)('dense-map hover, selection, and panning keep the surface and sidebar stable', async () => {
 const asset = (name: string) => readFileSync(new URL(name, import.meta.url), 'utf8');
 const script = ['graph-layout.js','model.js','overview.js','app.js'].map(asset).join('\n');
 const html = asset('app.html').replace('__PDG_JSON__','null').replace('/* SCRIPTS */',script);
 const symbols = Array.from({length:2000},(_,i)=>({id:`f:${i}`,name:`F${i}`,package:'repo/adapters/big',file:'demo.go',line:i+1,end_line:i+1,start:i,end:i+1,call_resolution:'typed',calls:i<1999?[{target:`f:${i+1}`}]:[]}));
 const index = {root:'/fixture',module:'repo',lines:2000,packages:[{id:'repo/adapters/big',files:1,lines:2000,imports:[]}],symbols};
 const server = Bun.serve({port:0,hostname:'127.0.0.1',fetch(request){
  const path = new URL(request.url).pathname;
  if(path==='/api/index')return Response.json(index);
  if(path==='/api/source')return Response.json({content:'func example() {}'});
  return new Response(html,{headers:{'content-type':'text/html'}});
 }});
 const profile = mkdtempSync(join(tmpdir(),'pdg-browser-test-'));
 const browserProcess = Bun.spawn([chromium!,'--headless','--no-sandbox','--disable-gpu','--remote-debugging-port=0',`--user-data-dir=${profile}`,'about:blank'],{stdout:'ignore',stderr:'ignore'});
 let socket: WebSocket | undefined;
 try {
  const deadline = Date.now()+10000, portFile = join(profile,'DevToolsActivePort');
  while(!existsSync(portFile)&&Date.now()<deadline)await Bun.sleep(25);
  if(!existsSync(portFile))throw new Error('Chromium does not expose its test port');
  const port = readFileSync(portFile,'utf8').split('\n')[0];
  const tab = await (await fetch(`http://127.0.0.1:${port}/json/new?about:blank`,{method:'PUT'})).json() as {webSocketDebuggerUrl:string};
  socket = new WebSocket(tab.webSocketDebuggerUrl);
  await new Promise<void>((resolve,reject)=>{socket!.onopen=()=>resolve();socket!.onerror=reject;});
  const pending = new Map<number,{resolve:(result:any)=>void,reject:(error:any)=>void}>();
  let nextID = 0;
  socket.onmessage=event=>{const reply=JSON.parse(String(event.data));const task=pending.get(reply.id);if(!task)return;pending.delete(reply.id);reply.error?task.reject(reply.error):task.resolve(reply.result);};
  const send = (method:string,params:object={}) => new Promise<any>((resolve,reject)=>{const id=++nextID;pending.set(id,{resolve,reject});socket!.send(JSON.stringify({id,method,params}));});
  const evaluate = async(expression:string)=>{
   const result=await send('Runtime.evaluate',{expression,returnByValue:true,awaitPromise:true});
   if(result.exceptionDetails)throw new Error(JSON.stringify(result.exceptionDetails));
   return result.result.value;
  };
  await send('Page.enable');await send('Page.bringToFront');
  await send('Emulation.setDeviceMetricsOverride',{width:1600,height:1000,deviceScaleFactor:2,mobile:false});
  await send('Page.navigate',{url:`${server.url}#/all/f%3A500`});
  const readyUntil=Date.now()+10000;
  while(Date.now()<readyUntil){if(await evaluate("typeof state!=='undefined' && state.view?.overview && cam.k>=.5 && state.view.overview.visibleCount>0 && state.view.overview.ready && !cameraFrame"))break;await Bun.sleep(50);}
  await Bun.sleep(350);
  const result=await evaluate(`(async()=>{
   const scene=document.querySelector('.overview-scene');
   if(!scene)throw new Error('Map surface is missing');
   const observer=new MutationObserver(()=>{});observer.observe(canvas,{childList:true,subtree:true});
   for(const id of [...state.view.overview.index.nodes.keys()].slice(400,450)){state.hover=id;paint()}
   await new Promise(requestAnimationFrame);await new Promise(requestAnimationFrame);
   const hoverStable=observer.takeRecords().filter(m=>m.type==='childList').length===0;
   observer.disconnect();
   const tree=document.querySelector('#tree a');
   go(href.all('f:501'),true);
   const treeStable=tree===document.querySelector('#tree a');
   cam.x+=256;applyCamera();await new Promise(requestAnimationFrame);await new Promise(requestAnimationFrame);
   const panStable=scene===document.querySelector('.overview-scene')&&canvas.querySelectorAll('*').length<20;
   const position=state.view.layout.positions.get(state.selected), rect=canvas.getBoundingClientRect();
   const x=(position.x-cam.x)*cam.k,y=(position.y-cam.y)*cam.k;
   const pixel=scene.getContext('2d').getImageData(Math.floor(x*scene.width/rect.width),Math.floor(y*scene.height/rect.height),1,1).data;
   return {hoverStable,treeStable,panStable,count:state.view.overview.rasterCount,painted:pixel[3]>0&&pixel[2]>pixel[0]&&pixel[0]>25};
  })()`);
  expect(result).toEqual({hoverStable:true,treeStable:true,panStable:true,count:2000,painted:true});
  await send('Input.dispatchMouseEvent',{type:'mouseMoved',x:700,y:400});
  await send('Input.dispatchMouseEvent',{type:'mousePressed',button:'left',clickCount:1,x:700,y:400});
  await send('Input.dispatchMouseEvent',{type:'mouseMoved',button:'left',buttons:1,x:720,y:410});
  await evaluate("(async()=>{await new Promise(requestAnimationFrame);await new Promise(requestAnimationFrame);const ctx=document.querySelector('.overview-scene').getContext('2d');window.__textPaints=0;const original=ctx.fillText;ctx.fillText=function(...args){window.__textPaints++;return original.apply(this,args)}})()");
  await send('Input.dispatchMouseEvent',{type:'mouseMoved',button:'left',buttons:1,x:750,y:420});
  const reused=await evaluate("(async()=>{await new Promise(requestAnimationFrame);await new Promise(requestAnimationFrame);return window.__textPaints===0&&document.querySelector('.overview-scene').style.transform.includes('translate3d')})()");
  expect(reused).toBe(true);
  await send('Input.dispatchMouseEvent',{type:'mouseReleased',button:'left',clickCount:1,x:750,y:420});

 } finally {
  socket?.close();browserProcess.kill();await browserProcess.exited;await server.stop(true);rmSync(profile,{recursive:true,force:true});
 }
},30000);
