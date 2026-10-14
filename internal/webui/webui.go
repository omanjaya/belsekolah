// Package webui menyimpan hasil build frontend. Isi dist/ diganti oleh build Rencana 4.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var files embed.FS

// FS mengembalikan isi folder dist.
func FS() fs.FS {
	sub, err := fs.Sub(files, "dist")
	if err != nil {
		panic(err) // dist selalu ada saat kompilasi
	}
	return sub
}
