package apriori

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"

	"github.com/sirkon/goproxy/semver"
	"github.com/sirkon/goproxy/source"
)

type sourceImpl struct {
	path string
	mod  map[string]ModuleInfo
}

func (s *sourceImpl) ModulePath() string {
	return s.path
}

func (s *sourceImpl) Versions(ctx context.Context, prefix string) (tags []string, err error) {
	for version := range s.mod {
		tags = append(tags, version)
	}
	for _, tag := range tags {
		if !semver.IsValid(tag) {
			return nil, fmt.Errorf("invalid semver value %s", tag)
		}
	}
	sort.Slice(tags, func(i, j int) bool {
		return semver.Compare(tags[i], tags[j]) < 0
	})
	return
}

func (s *sourceImpl) Stat(ctx context.Context, rev string) (*source.RevInfo, error) {
	res, ok := s.mod[rev]
	if !ok {
		return nil, s.errMsg("version %s not found", rev)
	}
	return &res.RevInfo, nil
}

func (s *sourceImpl) GoMod(ctx context.Context, version string) (data []byte, err error) {
	item, ok := s.mod[version]
	if !ok {
		return nil, fmt.Errorf("module %s: version %s not found", version)
	}
	data, err = ioutil.ReadFile(item.GoModPath)
	if err != nil {
		return nil, s.errMsg("go.mod file for version %s not found: %s", version, err)
	}
	return
}

func (s *sourceImpl) Zip(ctx context.Context, version string) (file io.ReadCloser, err error) {
	item, ok := s.mod[version]
	if !ok {
		return nil, fmt.Errorf("module %s: version %s not found", version)
	}
	file, err = os.Open(item.ArchivePath)
	if err != nil {
		return nil, s.errMsg("archive file for version %s not found: %s", version, err)
	}
	return
}

func (s *sourceImpl) errMsg(format string, a ...interface{}) error {
	head := "module " + s.path + ": "
	return fmt.Errorf(head+format, a...)
}
