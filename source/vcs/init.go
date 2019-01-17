package vcs

import (
	"path/filepath"

	"github.com/sirkon/goproxy/internal/cfg"
	"github.com/sirkon/goproxy/internal/modfetch"
	"github.com/sirkon/goproxy/internal/modfetch/codehost"
)

func setupEnv(basedir string) {
	modfetch.QuietLookup = true // just to hide modfetch/cache.go#127
	modfetch.PkgMod = filepath.Join(basedir, "pkg", "mod")
	codehost.WorkRoot = filepath.Join(modfetch.PkgMod, "cache", "vcs")
	cfg.CmdName = "mod download" // just to hide modfetch/fetch.go#L87
}
