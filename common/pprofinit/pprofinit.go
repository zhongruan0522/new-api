// Package pprofinit provides lazy registration of net/http/pprof handlers.
//
// Importing this package triggers net/http/pprof's init(), which registers
// debug handlers on http.DefaultServeMux. This indirection lets the main
// binary avoid paying that init cost unless profiling is explicitly enabled.
package pprofinit

import (
	_ "net/http/pprof"
)

// PackageInitMarker exists so that other packages can reference it via
// `_ = pprofinit.PackageInitMarker` to force the Go linker to keep this
// package (and therefore net/http/pprof's init) in the final binary.
var PackageInitMarker = false
