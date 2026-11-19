// Package web embeds the frontend into the binary; the app serves it at "/"
// so nginx and deployment don't change.
//
// The frontend is a Vite project in this directory (Svelte + TypeScript).
// `npm run build` writes it to dist/app, which is embedded below; release
// builds (CI, ansible/, local/Dockerfile) always run it before `go build`.
// dist/.gitkeep is committed so a plain `go build` still compiles without
// Node; such a binary serves notbuilt/ instead.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist notbuilt
var files embed.FS

// Static returns the built frontend (dist/app), or the "frontend not built"
// page with built=false when `npm run build` hasn't run.
func Static() (fsys fs.FS, built bool, err error) {
	dir := "dist/app"
	if _, err := fs.Stat(files, dir+"/index.html"); err != nil {
		dir = "notbuilt"
	}
	fsys, err = fs.Sub(files, dir)
	return fsys, dir == "dist/app", err
}
