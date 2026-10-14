// Package version exposes the running API version. It exists so the public
// version endpoint reports the value the binary was built with, rather than a
// value guessed from documentation.
package version

// Version is the running API version. It defaults to "dev" for local builds and
// is overridden at build time with:
//
//	go build -ldflags "-X ryze/backend/version.Version=<version>"
//
// The deployment pipeline is expected to pass the released version here, so the
// version reported by the API and the version recorded in the release notes can
// never drift apart.
var Version = "dev"
