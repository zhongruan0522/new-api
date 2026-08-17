package webdist

import (
	"io/fs"

	webassets "github.com/NookMux/NookMux/web"
)

var (
	BuildFS   fs.FS  = webassets.BuildFS
	IndexPage []byte = webassets.IndexPage
)
