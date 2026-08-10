// Package webui embeds the peserta-facing exam web page directly into the
// cbt-server binary. Deliberately plain HTML/CSS/vanilla JS served from Go
// itself — no separate frontend build, no CDN, no framework bundle — same
// reasoning as building the exam engine in Go in the first place: fewer
// moving parts that can fail when a whole class logs in at once.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed static
var files embed.FS

// FS returns the embedded web UI rooted at its own directory, so it serves
// cleanly at "/" (e.g. "/style.css" instead of "/static/style.css").
func FS() (fs.FS, error) {
	return fs.Sub(files, "static")
}
