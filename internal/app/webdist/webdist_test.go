package webdist

import (
	"io/fs"
	"testing"
)

func TestWebDistEmbedsFrontendEntry(t *testing.T) {
	indexPage := string(IndexPage)
	if indexPage == "" || !containsDOCTYPE(indexPage) {
		t.Fatal("embedded frontend index page is empty or malformed")
	}

	if _, err := fs.Stat(BuildFS, "dist/index.html"); err != nil {
		t.Fatalf("embedded frontend file system does not contain dist/index.html: %v", err)
	}
}

func containsDOCTYPE(indexPage string) bool {
	return len(indexPage) >= 15 && indexPage[:15] == "<!doctype html>"
}
