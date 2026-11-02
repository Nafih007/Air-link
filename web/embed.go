package web

import "embed"

// FS holds the phone landing page and brand icons (no CDN, no build step).
//
//go:embed index.html 404.html app.js manifest.json favicon-32.png apple-touch-icon.png icon-192.png icon-512.png logomark-transparent.png
var FS embed.FS
