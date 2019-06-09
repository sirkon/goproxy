package aposteriori

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"sort"
	"strings"

	"github.com/rs/zerolog"

	"github.com/sirkon/goproxy"
	"github.com/sirkon/goproxy/semver"
)

type module struct {
	parent *plugin
	next   goproxy.Module
}

func (m *module) ModulePath() string {
	return m.next.ModulePath()
}

func (m *module) Versions(ctx context.Context, prefix string) (tags []string, err error) {
	if m.parent.registry != nil {
		m.parent.Lock()
		defer m.parent.Unlock()
		versions, ok := m.parent.registry[m.ModulePath()]
		if ok {
			zerolog.Ctx(ctx).Info().Msg("module versions list detected in a cache")
			for version := range versions {
				if strings.HasPrefix(version, prefix) {
					tags = append(tags, version)
				}
			}
			sort.Slice(tags, func(i, j int) bool {
				return semver.Compare(tags[i], tags[j]) < 0
			})
		}
		return tags, nil
	}
	return m.next.Versions(ctx, prefix)
}

func (m *module) Stat(ctx context.Context, rev string) (*goproxy.RevInfo, error) {
	if semver.IsValid(rev) {
		p := m.relPath(rev, "revinfo.json")
		res, err := m.parent.cache.Get(p)
		if err == nil {
			zerolog.Ctx(ctx).Info().Msg("module revision info for given version detected in a cache")
			defer func() {
				if err := res.Close(); err != nil {
					zerolog.Ctx(ctx).Error().Err(err).Msg("failed to close rev.info")
				}
			}()
			var dst goproxy.RevInfo
			unmr := json.NewDecoder(res)
			if err := unmr.Decode(&dst); err != nil {
				return nil, fmt.Errorf("invalid revision info json data: %s", err)
			}
			return &dst, nil
		}
		zerolog.Ctx(ctx).Error().Err(err).Msg("revision info not found in cache")
	}
	res, err := m.next.Stat(ctx, rev)
	if err != nil {
		return nil, err
	}
	var dst bytes.Buffer
	mrsr := json.NewEncoder(&dst)
	if err := mrsr.Encode(res); err != nil {
		return nil, fmt.Errorf("failed to marshal revision info: %s", err)
	}
	p := m.relPath(res.Version, "revinfo.json")
	if err := m.parent.cache.Set(p, &dst); err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("aposteriori failed to cache revision info")
	}
	return res, nil
}

func (m *module) relPath(version, name string) string {
	return path.Join(m.next.ModulePath(), version, name)
}

func (m *module) GoMod(ctx context.Context, version string) (data []byte, err error) {
	p := m.relPath(version, "go.mod")
	res, err := m.parent.cache.Get(p)
	if err == nil {
		zerolog.Ctx(ctx).Info().Msg("module go.mod for given version detected in a cache")
		defer func() {
			if err := res.Close(); err != nil {
				zerolog.Ctx(ctx).Error().Err(err).Msg("failed to close response cache")
			}
		}()
		return ioutil.ReadAll(res)
	}
	zerolog.Ctx(ctx).Debug().Err(err).Msg("no cached go.mod found")

	data, err = m.next.GoMod(ctx, version)
	if err != nil {
		return nil, fmt.Errorf("aposteriori go.mod delegation error: %s", err)
	}

	if err := m.parent.cache.Set(p, bytes.NewReader(data)); err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("failed to cache go.mod")
	}
	return data, nil
}

var _ io.ReadCloser = &cachingReadCloser{}

type cachingReadCloser struct {
	logger *zerolog.Logger
	src    io.ReadCloser
	buf    *bytes.Buffer

	plugin  *plugin
	module  *module
	version string

	name       string
	doNotCache bool
}

func (r *cachingReadCloser) Read(p []byte) (n int, err error) {
	n, err = r.src.Read(p)
	if n > 0 && !r.doNotCache {
		if _, cErr := r.buf.Write(p[:n]); cErr != nil {
			r.logger.Warn().Err(cErr).Msg("aposteriori: failed to copy written data into underlying buffer")
			r.doNotCache = true
		}
	}
	return n, err
}

func (r *cachingReadCloser) Close() error {
	err := r.src.Close()
	if !r.doNotCache {
		if sErr := r.plugin.cache.Set(r.name, r.buf); sErr != nil {
			r.logger.Error().Err(err).Msg("aposteriori: failed to save incoming source archive into cache")
			return nil
		}
		if r.plugin.registry != nil {
			r.plugin.Lock()
			res, ok := r.plugin.registry[r.module.ModulePath()]
			if !ok {
				res = map[string]struct{}{}
			}
			res[r.version] = struct{}{}
			r.plugin.registry[r.module.ModulePath()] = res
			r.plugin.Unlock()
		}
	}
	return err
}

func (m *module) Zip(ctx context.Context, version string) (io.ReadCloser, error) {
	p := m.relPath(version, "src.zip")
	file, err := m.parent.cache.Get(p)
	if err == nil {
		zerolog.Ctx(ctx).Info().Msg("module source archive for given version detected in a cache")
		return file, err
	}
	zerolog.Ctx(ctx).Debug().Err(err).Msg("no cached module source found")

	file, err = m.next.Zip(ctx, version)
	if err != nil {
		return nil, fmt.Errorf("aposteriori source delegation error: %s", err)
	}

	file = &cachingReadCloser{
		logger:  zerolog.Ctx(ctx),
		src:     file,
		buf:     &bytes.Buffer{},
		plugin:  m.parent,
		module:  m,
		version: version,
		name:    p,
	}
	return file, nil
}
