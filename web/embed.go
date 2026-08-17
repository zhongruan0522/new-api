package web

import "embed"

// The embed directive must stay next to web/dist because Go embed patterns
// cannot leave the package directory with "..".

//go:embed dist
var BuildFS embed.FS

//go:embed dist/index.html
var IndexPage []byte
