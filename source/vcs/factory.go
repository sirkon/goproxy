package vcs

import (
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/sirkon/goproxy/internal/modfetch"
	"github.com/sirkon/goproxy/source"
)

// factory creates source for VCS repositories
type factory struct {
	rootDir string

	// accessLock is used for
	accessLock sync.Locker
	inWork     map[string]modfetch.Repo
}

// NewFactory creates new valid factory instance
func NewFactory(rootDir string) (f source.Factory, err error) {
	setupEnv(rootDir)
	stat, err := os.Stat(rootDir)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(rootDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %s", rootDir, err)

		}
	} else {
		if !stat.IsDir() {
			return nil, fmt.Errorf("%s is not a directory", rootDir)
		}
	}
	if err = os.Chdir(rootDir); err != nil {
		return nil, fmt.Errorf("failed to cd into %s: %s", rootDir, err)
	}

	if err = os.Setenv("GOPATH", rootDir); err != nil {
		return nil, fmt.Errorf("failed to set up GOPATH environment variable: %s", err)
	}

	if err = os.Setenv("GO111MODULE", "on"); err != nil {
		return nil, fmt.Errorf("failed to set up GO111MODULE environment variable: %s", err)
	}

	return &factory{
		rootDir:    rootDir,
		inWork:     map[string]modfetch.Repo{},
		accessLock: &sync.Mutex{},
	}, nil
}

// Source creates a source for a module with given path
func (f *factory) Source(req *http.Request, prefix string) (source.Source, error) {
	path, _, err := source.GetModInfo(req, prefix)
	if err != nil {
		return nil, err
	}

	repo, err := f.getRepo(path)
	if err != nil {
		return nil, err
	}

	return &vscSource{
		repo: repo,
	}, nil
}

// Leave unset a lock of a given module
func (f *factory) Leave(s source.Source) error {
	return nil
}

// Close ...
func (f *factory) Close() error {
	return nil
}

func (f *factory) getRepo(path string) (repo modfetch.Repo, err error) {
	f.accessLock.Lock()
	defer f.accessLock.Unlock()
	repo, ok := f.inWork[path]
	if !ok {
		repo, err = modfetch.Lookup(path)
		if err != nil {
			return nil, fmt.Errorf("failed to get module `%s`: %s", path, err)
		}
	}
	f.inWork[path] = repo
	return repo, nil
}
