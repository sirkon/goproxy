package apriori

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"

	"github.com/sirkon/goproxy/internal/errors"

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
			return nil, errors.Newf("apriori invalid semver value %s", tag)
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
		return nil, errors.New(s.errMsg("apriori version %s not found", rev))
	}
	return &res.RevInfo, nil
}

func (s *aprioriModule) GoMod(ctx context.Context, version string) (data []byte, err error) {
	item, ok := s.mod[version]
	if !ok {
		return nil, errors.Newf("apriori module %s: version %s not found", s.path, version)
	}
	data, err = ioutil.ReadFile(item.GoModPath)
	if err != nil {
		return nil, errors.Wrap(err, s.errMsg("getting go.mod file for version %s", version))
	}
	return
}

func (s *aprioriModule) Zip(ctx context.Context, version string) (file io.ReadCloser, err error) {
	item, ok := s.mod[version]
	if !ok {
		return nil, errors.Newf("apriori module %s: version %s not found", s.path, version)
	}
	file, err = os.Open(item.ArchivePath)
	if err != nil {
		return nil, errors.Wrap(err, s.errMsg("apriori getting archive file for version %s", version))
	}
	return
}

func (s *aprioriModule) errMsg(format string, a ...interface{}) string {
	head := "module " + s.path + ": "
	return fmt.Sprintf(head+format, a...)
}
