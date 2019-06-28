package apriori

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"sort"

	"github.com/pkg/errors"

	"github.com/sirkon/goproxy"
	"github.com/sirkon/goproxy/semver"
)

type aprioriModule struct {
	path string
	mod  map[string]ModuleInfo
}

func (s *aprioriModule) ModulePath() string {
	return s.path
}

func (s *aprioriModule) Versions(ctx context.Context, prefix string) (tags []string, err error) {
	for version := range s.mod {
		tags = append(tags, version)
	}
	for _, tag := range tags {
		if !semver.IsValid(tag) {
			return nil, errors.Errorf("invalid semver value %s", tag)
		}
	}
	sort.Slice(tags, func(i, j int) bool {
		return semver.Compare(tags[i], tags[j]) < 0
	})
	return
}

func (s *aprioriModule) Stat(ctx context.Context, rev string) (*goproxy.RevInfo, error) {
	res, ok := s.mod[rev]
	if !ok {
		return nil, s.errMsg("version %s not found", rev)
	}
	return &res.RevInfo, nil
}

func (s *aprioriModule) GoMod(ctx context.Context, version string) (data []byte, err error) {
	item, ok := s.mod[version]
	if !ok {
		return nil, errors.Errorf("module %s: version %s not found", version)
	}
	data, err = ioutil.ReadFile(item.GoModPath)
	if err != nil {
		return nil, s.errMsg("go.mod file for version %s not found: %s", version, err)
	}
	return
}

func (s *aprioriModule) Zip(ctx context.Context, version string) (file io.ReadCloser, err error) {
	item, ok := s.mod[version]
	if !ok {
		return nil, errors.Errorf("module %s: version %s not found", version)
	}
	file, err = os.Open(item.ArchivePath)
	if err != nil {
		return nil, s.errMsg("archive file for version %s not found: %s", version, err)
	}
	return
}

func (s *aprioriModule) errMsg(format string, a ...interface{}) error {
	head := "module " + s.path + ": "
	return errors.Errorf(head+format, a...)
}
