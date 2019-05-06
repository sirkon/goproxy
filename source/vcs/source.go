package vcs

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/rs/zerolog"

	"github.com/sirkon/goproxy/internal/modfetch"
	"github.com/sirkon/goproxy/source"
)

type vscSource struct {
	repo modfetch.Repo
}

func (s *vscSource) ModulePath() string {
	return s.repo.ModulePath()
}

func (s *vscSource) Versions(ctx context.Context, prefix string) (tags []string, err error) {
	type data struct {
		tags []string
		err  error
	}
	dataChan := make(chan data, 1)
	go func() {
		t, e := s.repo.Versions(prefix)
		if len(t) == 0 {
			info, err := s.repo.Latest()
			if err != nil {
				e = err
			} else {
				t = []string{info.Version}
			}
		}
		dataChan <- data{
			tags: t,
			err:  e,
		}
	}()

	select {
	case info := <-dataChan:
		return info.tags, info.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *vscSource) Stat(ctx context.Context, rev string) (*source.RevInfo, error) {
	type data struct {
		info *source.RevInfo
		err  error
	}
	dataChan := make(chan data, 1)
	go func() {
		raw, err := s.repo.Stat(rev)
		if err != nil {
			dataChan <- data{
				info: nil,
				err:  err,
			}
			return
		}
		res := &source.RevInfo{}
		res.Name = raw.Name
		res.Short = raw.Short
		res.Time = raw.Time.Format(time.RFC3339)
		res.Version = raw.Version
		dataChan <- data{
			info: res,
			err:  nil,
		}
	}()

	select {
	case res := <-dataChan:
		return res.info, res.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *vscSource) GoMod(ctx context.Context, version string) ([]byte, error) {
	type data struct {
		data []byte
		err  error
	}
	dataChan := make(chan data, 1)
	go func() {
		res, err := s.repo.GoMod(version)
		dataChan <- data{
			data: res,
			err:  err,
		}
	}()

	select {
	case res := <-dataChan:
		return res.data, res.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *vscSource) Zip(ctx context.Context, version string) (file io.ReadCloser, err error) {
	type data struct {
		file io.ReadCloser
		err  error
	}
	dataChan := make(chan data, 1)

	go func() {
		dir, err := ioutil.TempDir(".", ".downloads")
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("failed to create temporary directory")
		}
		defer func() {
			if err := os.RemoveAll(dir); err != nil {
				zerolog.Ctx(ctx).Error().Err(err).Msg("failed to remove temporary directory")
			}
		}()
		fileName, err := s.repo.Zip(version, dir)
		if err != nil {
			dataChan <- data{
				file: nil,
				err:  err,
			}
			return
		}

		osFile, err := os.Open(fileName)
		if err != nil {
			dataChan <- data{
				file: nil,
				err:  err,
			}
			return
		}

		dataChan <- data{
			file: osFile,
			err:  nil,
		}
	}()

	select {
	case res := <-dataChan:
		return res.file, res.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
