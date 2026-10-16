const page = Bun.file(new URL('./index.html', import.meta.url));
const server = Bun.serve({
  hostname: '127.0.0.1',
  port: 8092,
  fetch(request) {
    if (new URL(request.url).pathname !== '/') return new Response('Not found', {status:404});
    return new Response(page, {headers:{'content-type':'text/html; charset=utf-8'}});
  },
});
console.log(`Source lens: ${server.url}`);
