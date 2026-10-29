// Package buildinfo carries build-time metadata of the server binary.
package buildinfo

// Version is the server version reported by GET /api/v1/server/info. The
// "dev" default is replaced by release builds, which inject the version
// through the linker, for example:
//
//	go build -ldflags "-X github.com/welpher/lumemor/server/internal/buildinfo.Version=0.1.0"
var Version = "dev"
