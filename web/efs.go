package web

import (
	"embed"
	"io/fs"
	"os"
)

//go:embed dist/*
var distFS embed.FS

// GetFileSystem returns the static files to serve.
// If FRONTEND_DIR env var is set, serves from disk (for dev).
// Otherwise serves from embedded binary (for production).
func GetFileSystem() (fs.FS, error) {
	// 1. Dev mode: Serve from disk if requested
	if dir := os.Getenv("FRONTEND_DIR"); dir != "" {
		return os.DirFS(dir), nil
	}

	// 2. Production mode: Serve embedded files
	// We need to strip the "dist" prefix from the embedded FS
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, err
	}
	return sub, nil
}
