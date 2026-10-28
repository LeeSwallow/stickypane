// Dev only: an html page that tries to reach url every way a page can —
// an image, a stylesheet, an @import and a background, a script, a frame,
// an object, a refresh, a form — and to run its own script, which would
// say whether it reached the window. A server at url sees no request
// when the sandbox (src/shapes/html.ts) holds.
export const probePage = (url: string): string =>
  '<p id="s">The page\'s own script did not run.</p>' +
  '<script>document.getElementById("s").textContent = "SCRIPT RAN: " + (function(){try{return parent.document.title}catch(e){return "parent blocked"}})()</script>' +
  `<img src="${url}/leak-img" alt="(no image)">` +
  `<link rel="stylesheet" href="${url}/leak-css">` +
  `<style>@import url("${url}/leak-import"); p{background:url("${url}/leak-bg")}</style>` +
  `<script src="${url}/leak-js"></script>` +
  `<iframe src="${url}/leak-frame"></iframe>` +
  `<object data="${url}/leak-object"></object>` +
  `<meta http-equiv="refresh" content="0;url=${url}/leak-refresh">` +
  `<form action="${url}/leak-form" method="post"><button>post</button></form>`
