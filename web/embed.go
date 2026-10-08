//go:build embedweb

// Package web holds the built frontend. Release builds use -tags embedweb after
// the frontend build; dev builds leave it out and Vite serves the frontend.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

func FS() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
