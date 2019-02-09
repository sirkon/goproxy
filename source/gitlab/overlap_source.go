package gitlab

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/rs/zerolog"

	"github.com/sirkon/goproxy/internal/modfile"
	"github.com/sirkon/goproxy/source"
)

var _ source.Source = overlapGitlabSource{}

type overlapGitlabSource struct {
	sources []source.Source
	version int
}

func (s overlapGitlabSource) ModulePath() string {
	for _, ss := range s.sources {
		return ss.ModulePath()
	}
	return ""
}

func (s overlapGitlabSource) Versions(ctx context.Context, prefix string) (tags []string, err error) {
	for _, ss := range s.sources {
		tags, err = ss.Versions(ctx, prefix)
		if err == nil {
			if s.version < 2 {
				return s.getEarlyVersions(tags), nil
			}
			return s.filterByVersion(tags), nil
		}
	}
	return nil, fmt.Errorf("cannot get versions for given module")
}

func (s overlapGitlabSource) Stat(ctx context.Context, rev string) (*source.RevInfo, error) {
	for _, ss := range s.sources {
		info, err := ss.Stat(ctx, rev)
		if err == nil {
			return info, nil
		}
	}
	return nil, fmt.Errorf("cannot get stat for given module")
}

func (s overlapGitlabSource) GoMod(ctx context.Context, version string) (data []byte, err error) {
	for _, ss := range s.sources {
		data, err = ss.GoMod(ctx, version)
		modFile, err := modfile.Parse("go.mod", data, nil)
		if err == nil {
			if strings.HasSuffix(modFile.Module.Mod.Path, fmt.Sprintf("/v%d", s.version)) {
				return data, nil
			}
			zerolog.Ctx(ctx).Warn().Timestamp().Int("version", s.version).Str("path", modFile.Module.Mod.Path).Msgf("version mismatch")
		}
	}
	return nil, fmt.Errorf("failed to find go.mod for given module and version")
}

func (s overlapGitlabSource) Zip(ctx context.Context, version string) (file io.ReadCloser, err error) {
	for _, ss := range s.sources {
		file, err = ss.Zip(ctx, version)
		if err == nil {
			return
		}
	}

	return nil, fmt.Errorf("failed to find go.mod for given module and version")
}

func (s overlapGitlabSource) filterByVersion(tags []string) []string {
	var res []string
	filter := fmt.Sprintf("v%d", s.version)
	for _, tag := range tags {
		if strings.HasPrefix(tag, filter) {
			res = append(res, tag)
		}
	}
	return res
}

func (s overlapGitlabSource) getEarlyVersions(tags []string) []string {
	var res []string
	for _, tag := range tags {
		if strings.HasPrefix(tag, "v0") || strings.HasPrefix(tag, "v1") {
			res = append(res, tag)
		}
	}
	return res
}
