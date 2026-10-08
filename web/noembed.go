//go:build !embedweb

package web

import "io/fs"

// FS is nil when the frontend is not embedded.
func FS() fs.FS { return nil }
