package web

import (
	"embed"
	"io/fs"
	"os"
)

//go:embed build/*
var distFS embed.FS

// GetFileSystem returns the static files to serve.
func GetFileSystem() (fs.FS, error) {
	// 1. Dev mode: Serve from disk
	if dir := os.Getenv("FRONTEND_DIR"); dir != "" {
		return os.DirFS(dir), nil
	}

	// 2. Production mode: Serve embedded files
	// SvelteKit outputs to "build" folder by default with adapter-static
	sub, err := fs.Sub(distFS, "build")
	if err != nil {
		return nil, err
	}
	return sub, nil
}
