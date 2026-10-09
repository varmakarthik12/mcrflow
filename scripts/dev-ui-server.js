#!/usr/bin/env node
const http = require('http');
const fs = require('fs');
const path = require('path');

const PORT = process.env.PORT || 3080;
const BACKEND_HOST = process.env.BACKEND_HOST || 'localhost';
const BACKEND_PORT = process.env.BACKEND_PORT || 3081;
const WEB_DIR = path.resolve(__dirname, '..', 'web');

const MIME_TYPES = {
  '.html': 'text/html; charset=UTF-8',
  '.js': 'application/javascript; charset=UTF-8',
  '.css': 'text/css; charset=UTF-8',
  '.json': 'application/json; charset=UTF-8',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.svg': 'image/svg+xml',
  '.ico': 'image/x-icon',
};

const server = http.createServer((req, res) => {
  // Proxy API and HLS requests to the Control Plane
  if (req.url.startsWith('/api/') || req.url.startsWith('/hls/')) {
    console.log(`[PROXY] ${req.method} ${req.url} -> http://${BACKEND_HOST}:${BACKEND_PORT}${req.url}`);
    const proxyReq = http.request(
      {
        host: BACKEND_HOST,
        port: BACKEND_PORT,
        path: req.url,
        method: req.method,
        headers: { ...req.headers, host: `${BACKEND_HOST}:${BACKEND_PORT}` },
      },
      (proxyRes) => {
        res.writeHead(proxyRes.statusCode, proxyRes.headers);
        proxyRes.pipe(res);
      }
    );

    proxyReq.on('error', (err) => {
      console.error(`[PROXY ERROR] ${err.message}`);
      res.writeHead(502, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'Control Plane unreachable at http://' + BACKEND_HOST + ':' + BACKEND_PORT }));
    });

    req.pipe(proxyReq);
    return;
  }

  // Serve static files from web/
  let reqPath = req.url.split('?')[0];
  if (reqPath === '/' || reqPath === '') {
    reqPath = '/index.html';
  }

  let filePath = path.join(WEB_DIR, reqPath);
  if (!fs.existsSync(filePath) || fs.statSync(filePath).isDirectory()) {
    filePath = path.join(WEB_DIR, 'index.html');
  }

  const ext = path.extname(filePath).toLowerCase();
  const contentType = MIME_TYPES[ext] || 'application/octet-stream';

  fs.readFile(filePath, (err, content) => {
    if (err) {
      res.writeHead(500);
      res.end('Error loading ' + reqPath);
      return;
    }
    res.writeHead(200, { 'Content-Type': contentType });
    res.end(content);
  });
});

server.listen(PORT, () => {
  console.log('================================================================================');
  console.log(`  MCRFlow Web UI Dev Server listening at: http://localhost:${PORT}`);
  console.log(`  Proxying /api/* & /hls/* -> http://${BACKEND_HOST}:${BACKEND_PORT}`);
  console.log('================================================================================');
});
