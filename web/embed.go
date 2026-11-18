// Package web embeds the frontend (plain HTML/CSS/JS, no build step) into
// the binary; the app serves it at "/" so nginx and deployment don't change.
package web

import (
	"embed"
	"io/fs"
)

//go:embed static
var files embed.FS

// Static returns the frontend files rooted at static/.
func Static() (fs.FS, error) {
	return fs.Sub(files, "static")
}
