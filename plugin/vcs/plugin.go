package vcs

import (
	"net/http"
	"os"
	"sync"

	"github.com/sirkon/goproxy/internal/errors"

	"github.com/sirkon/goproxy"
	"github.com/sirkon/goproxy/internal/modfetch"
)

// plugin creates source for VCS repositories
type plugin struct {
	rootDir string

	// accessLock is for access to inWork
	accessLock sync.Locker
	inWork     map[string]modfetch.Repo
}

func (f *plugin) String() string {
	return "legacy"
}

// NewPlugin creates new valid plugin instance
func NewPlugin(rootDir string) (f goproxy.Plugin, err error) {
	setupEnv(rootDir)
	stat, err := os.Stat(rootDir)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(rootDir, 0755); err != nil {
			return nil, errors.Wrapf(err, "vcs creating directory `%s`", rootDir)
		}
	} else {
		if !stat.IsDir() {
			return nil, errors.Newf("vcs %s is not a directory", rootDir)
		}
	}
	if err = os.Chdir(rootDir); err != nil {
		return nil, errors.Wrapf(err, "vcs cding into `%s`", rootDir)
	}

	if err = os.Setenv("GOPATH", rootDir); err != nil {
		return nil, errors.Wrapf(err, "vcs setting up GOPATH environment variable")
	}

	if err = os.Setenv("GO111MODULE", "on"); err != nil {
		return nil, errors.Wrapf(err, "vcs setting up GO111MODULE environment variable")
	}

	return &plugin{
		rootDir:    rootDir,
		inWork:     map[string]modfetch.Repo{},
		accessLock: &sync.Mutex{},
	}, nil
}

// Module creates a source for a module with given path
func (f *plugin) Module(req *http.Request, prefix string) (goproxy.Module, error) {
	path, _, err := goproxy.GetModInfo(req, prefix)
	if err != nil {
		return nil, err
	}

	repo, err := f.getRepo(path)
	if err != nil {
		return nil, err
	}

	return &vcsModule{
		repo: repo,
	}, nil
}

// Leave unset a lock of a given module
func (f *plugin) Leave(s goproxy.Module) error {
	return nil
}

// Close ...
func (f *plugin) Close() error {
	return nil
}

func (f *plugin) getRepo(path string) (repo modfetch.Repo, err error) {
	f.accessLock.Lock()
	defer f.accessLock.Unlock()
	repo, ok := f.inWork[path]
	if !ok {
		repo, err = modfetch.Lookup(path)
		if err != nil {
			return nil, errors.Wrapf(err, "vcs getting module for `%s`", path)
		}
	}
	f.inWork[path] = repo
	return repo, nil
}
