//go:build !webembed

package web

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func distFS() (fs.FS, error) {
	candidates := []string{
		"internal/web/dist",
		"dist",
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "internal/web/dist"))
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "dist"))
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, "index.html")); err == nil {
			return os.DirFS(candidate), nil
		}
	}

	return nil, fmt.Errorf("web dist not found: run `cd internal/web && npm run build` first")
}
