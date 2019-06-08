package aposteriori

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"path"

	"github.com/rs/zerolog"

	"github.com/sirkon/goproxy"
)

type module struct {
	cache FileCache
	next  goproxy.Module
}

func (m *module) ModulePath() string {
	return m.next.ModulePath()
}

func (m *module) Versions(ctx context.Context, prefix string) (tags []string, err error) {
	return m.next.Versions(ctx, prefix)
}

func (m *module) Stat(ctx context.Context, rev string) (*goproxy.RevInfo, error) {
	return m.next.Stat(ctx, rev)
}

func (m *module) GoMod(ctx context.Context, version string) (data []byte, err error) {
	p := path.Join(m.next.ModulePath(), fmt.Sprintf("%s.mod", version))
	res, err := m.cache.Get(p)
	if err == nil {
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

	if err := m.cache.Set(p, bytes.NewReader(data)); err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("failed to cache go.mod")
	}
	return data, nil
}

var _ io.ReadCloser = &cachingReadCloser{}

type cachingReadCloser struct {
	logger *zerolog.Logger
	src    io.ReadCloser
	buf    *bytes.Buffer
	cache  FileCache

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
		if sErr := r.cache.Set(r.name, r.buf); sErr != nil {
			r.logger.Error().Err(err).Msg("aposteriori: failed to save incoming source archive into cache")
		}
	}
	return err
}

func (m *module) Zip(ctx context.Context, version string) (io.ReadCloser, error) {
	p := path.Join(m.next.ModulePath(), fmt.Sprintf("%s.zip", version))
	file, err := m.cache.Get(p)
	if err == nil {
		return file, err
	}
	zerolog.Ctx(ctx).Debug().Err(err).Msg("no cached module source found")

	file, err = m.next.Zip(ctx, version)
	if err != nil {
		return nil, fmt.Errorf("aposteriori source delegation error: %s", err)
	}

	file = &cachingReadCloser{
		logger: zerolog.Ctx(ctx),
		src:    file,
		buf:    &bytes.Buffer{},
		cache:  m.cache,
		name:   p,
	}
	return file, nil
}
