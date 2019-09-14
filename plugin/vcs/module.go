package vcs

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/rs/zerolog"

	"github.com/sirkon/goproxy"
	"github.com/sirkon/goproxy/internal/errors"
	"github.com/sirkon/goproxy/internal/modfetch"
)

type vcsModule struct {
	repo modfetch.Repo
}

func (s *vcsModule) ModulePath() string {
	return s.repo.ModulePath()
}

func (s *vcsModule) Versions(ctx context.Context, prefix string) (tags []string, err error) {
	defer func() {
		if err != nil {
			err = errors.Wrap(err, "vcs getting versions")
		}
	}()
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

func (s *vcsModule) Stat(ctx context.Context, rev string) (res *goproxy.RevInfo, err error) {
	defer func() {
		if err != nil {
			err = errors.Wrap(err, "vcs getting stat")
		}
	}()

	type data struct {
		info *goproxy.RevInfo
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
		res := &goproxy.RevInfo{}
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

func (s *vcsModule) GoMod(ctx context.Context, version string) (file []byte, err error) {
	defer func() {
		if err != nil {
			err = errors.Wrap(err, "vcs getting go.mod")
		}
	}()

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

func (s *vcsModule) Zip(ctx context.Context, version string) (file io.ReadCloser, err error) {
	type data struct {
		file io.ReadCloser
		err  error
	}
	dataChan := make(chan data, 1)

	go func() {
		dir, err := ioutil.TempDir(".", ".downloads")
		if err != nil {
			dataChan <- data{
				file: nil,
				err:  errors.Wrap(err, "vcs creating temporary directory to save source archive"),
			}
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
				err:  errors.Wrap(err, "vcs getting source archive"),
			}
			return
		}

		osFile, err := os.Open(fileName)
		if err != nil {
			dataChan <- data{
				file: nil,
				err:  errors.Wrap(err, "vcs opening downloaded source archive"),
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
		return nil, errors.Wrap(ctx.Err(), "vcs getting source archive")
	}
}
