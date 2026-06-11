//go:build webembed

package web

import (
	"embed"
	"io/fs"
)

//go:embed dist
var embeddedDist embed.FS

func distFS() (fs.FS, error) {
	return fs.Sub(embeddedDist, "dist")
}
