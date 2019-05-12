package goproxy

import (
	"context"
	"io"
)

// RevInfo describes a single revision of a module source
type RevInfo struct {
	Version string // version string
	Time    string // commit time

	// These fields are used for Stat of arbitrary rev,
	// but they are not recorded when talking about module versions.
	Name  string `json:"-"` // complete ID in underlying repository
	Short string `json:"-"` // shortened ID, for use in pseudo-version
}

// Module represents go module: some VSC (git, mercurial, svn, etc), Gitlab, another Go modules proxy, etc
type Module interface {
	// ModulePath returns the module path.
	ModulePath() string

	// Versions lists all known versions with the given prefix.
	// Pseudo-versions are not included.
	// Versions should be returned sorted in semver order
	// (implementations can use SortVersions).
	Versions(ctx context.Context, prefix string) (tags []string, err error)

	// Stat returns information about the revision rev.
	// A revision can be any identifier known to the underlying service:
	// commit hash, branch, tag, and so on.
	Stat(ctx context.Context, rev string) (*RevInfo, error)

	// GoMod returns the go.mod file for the given version.
	GoMod(ctx context.Context, version string) (data []byte, err error)

	// Zip returns file reader of ZIP file for the given version of the module
	Zip(ctx context.Context, version string) (file io.ReadCloser, err error)
}
