package buildinfo

import "time"

// Set via -ldflags at build time
var (
	BuildTime  string // when the binary was compiled
	CommitTime string // last git commit time (last code edit)
	CommitHash string // short git commit hash
)

// StartTime is recorded when the process starts
var StartTime = time.Now().UTC().Format(time.RFC3339)
