package gitlab

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/rs/zerolog"

	"github.com/sirkon/goproxy"
	"github.com/sirkon/goproxy/internal/modfile"
	"github.com/sirkon/goproxy/semver"
)

var _ goproxy.Module = overlapGitlabModule{}

type overlapGitlabModule struct {
	sources []goproxy.Module
	version int
}

func (s overlapGitlabModule) ModulePath() string {
	for _, ss := range s.sources {
		return ss.ModulePath()
	}
	return ""
}

func (s overlapGitlabModule) Versions(ctx context.Context, prefix string) (tags []string, err error) {
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

func (s overlapGitlabModule) Stat(ctx context.Context, rev string) (*goproxy.RevInfo, error) {
	for _, ss := range s.sources {
		info, err := ss.Stat(ctx, rev)
		if err == nil {
			// need to check if major in result is less than requested version. Need to use v<Major>.0.0 prefix instead
			base, moment, sha := semver.PseudoParts(info.Version)
			if len(base) != 0 {
				if semver.Major(base) < s.version {
					info.Version = fmt.Sprintf("v%d.0.0-pre-%s-%s", s.version, moment, sha)
				}
			}
			return info, nil
		}
	}
	return nil, fmt.Errorf("cannot get stat for given module")
}

func (s overlapGitlabModule) GoMod(ctx context.Context, version string) (data []byte, err error) {
	for _, ss := range s.sources {
		data, err = ss.GoMod(ctx, version)
		if err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msgf("failed to get zip archive for %s", ss.ModulePath())
			continue
		}
		modFile, err := modfile.Parse("go.mod", data, nil)
		if err == nil {
			zerolog.Ctx(ctx).Debug().Str("module", modFile.Module.Mod.Path).Msgf("got module path")
			if strings.HasSuffix(modFile.Module.Mod.Path, fmt.Sprintf("/v%d", s.version)) || s.version < 2 {
				return data, nil
			}
			zerolog.Ctx(ctx).Warn().Int("version", s.version).Str("path", modFile.Module.Mod.Path).Msgf("version mismatch")
		} else {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("failed to parse go.mod")
		}
	}
	return nil, fmt.Errorf("failed to find go.mod for given module and version")
}

func (s overlapGitlabModule) Zip(ctx context.Context, version string) (file io.ReadCloser, err error) {
	for _, ss := range s.sources {
		file, err = ss.Zip(ctx, version)
		if err == nil {
			return
		}
		zerolog.Ctx(ctx).Warn().Err(err).Msg("failed to get archieve from underlying source")
	}

	return nil, fmt.Errorf("failed to find zip archive for given module and version")
}

func (s overlapGitlabModule) filterByVersion(tags []string) []string {
	var res []string
	filter := fmt.Sprintf("v%d", s.version)
	for _, tag := range tags {
		if strings.HasPrefix(tag, filter) {
			res = append(res, tag)
		}
	}
	return res
}

func (s overlapGitlabModule) getEarlyVersions(tags []string) []string {
	var res []string
	for _, tag := range tags {
		if strings.HasPrefix(tag, "v0") || strings.HasPrefix(tag, "v1") {
			res = append(res, tag)
		}
	}
	return res
}
