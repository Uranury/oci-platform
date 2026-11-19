package web

import (
	"io/fs"
	"testing"
)

// Whether or not `npm run build` has run, "/" must have an index.html.
func TestStatic(t *testing.T) {
	fsys, built, err := Static()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Stat(fsys, "index.html"); err != nil {
		t.Fatalf("built=%v: %v", built, err)
	}
	t.Logf("built=%v", built)
}
