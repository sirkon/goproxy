package gitlab

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/sirkon/gitlab"

	"github.com/sirkon/goproxy"
	"github.com/sirkon/goproxy/fsrepack"
	"github.com/sirkon/goproxy/internal/semver"
)

type gitlabModule struct {
	client   gitlab.Client
	fullPath string
	path     string
}

func (s *gitlabModule) ModulePath() string {
	return s.path
}

func (s *gitlabModule) Versions(ctx context.Context, prefix string) ([]string, error) {
	tags, err := s.client.Tags(ctx, s.path, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get tags from gitlab repository: %s", err)
	}

	var resp []string
	for _, tag := range tags {
		if semver.IsValid(tag.Name) {
			resp = append(resp, tag.Name)
		}
	}
	if len(resp) == 0 {
		return nil, fmt.Errorf("invalid repository %s, not tags found", s.path)
	}

	return resp, nil
}

func (s *gitlabModule) Stat(ctx context.Context, rev string) (*goproxy.RevInfo, error) {
	tags, err := s.client.Tags(ctx, s.path, rev)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags from gitlab repository: %s", err)
	}

	// Looking for exact revision match
	for _, tag := range tags {
		if tag.Name == rev {
			return &goproxy.RevInfo{
				Version: tag.Name,
				Time:    tag.Commit.CreatedAt,
				Name:    tag.Commit.ID,
				Short:   tag.Commit.ShortID,
			}, nil
		}
	}

	return nil, fmt.Errorf("state: unknown revision %s for %s", rev, s.path)
}

func (s *gitlabModule) GoMod(ctx context.Context, version string) (data []byte, err error) {
	return s.client.File(ctx, s.path, "go.mod", version)
}

type bufferCloser struct {
	bytes.Buffer
}

// Close makes bufferCloser io.ReadCloser
func (*bufferCloser) Close() error { return nil }

func (s *gitlabModule) Zip(ctx context.Context, version string) (io.ReadCloser, error) {
	modInfo, err := s.client.ProjectInfo(ctx, s.path)
	if err != nil {
		return nil, fmt.Errorf("failed to get project %s info: %s", s.path, err)
	}

	archive, err := s.client.Archive(ctx, modInfo.ID, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get zipped archive data: %s", err)
	}

	repacker, err := fsrepack.Gitlab(s.fullPath, version)
	if err != nil {
		return nil, err
	}

	// now need to repack archive content from <pkg-name>-<hash> → <full pkg name, such as gitlab.com/user/module>, e.g.
	//
	// > module-f5d5d62240829ba7f38614add00c4aba587cffb1:
	// >   go.mod
	// >   pkg.go
	//
	// from gitlab.com/user/module, where f5d5d62240829ba7f38614add00c4aba587cffb1 is a hash of the revision tagged
	// v0.0.1 will be repacked into
	//
	// > gitlab.com:
	// >    user.name:
	// >        module@v0.1.2:
	// >            go.mod
	// >            pkg.go
	zipped, err := ioutil.ReadAll(archive)
	if err != nil {
		return nil, fmt.Errorf("failed to read out archive data: %s", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(zipped), int64(len(zipped)))
	if err != nil {
		return nil, fmt.Errorf("failed to extract zipped data: %s", err)
	}

	rawDest := &bufferCloser{}
	result := rawDest
	dest := zip.NewWriter(rawDest)
	defer dest.Close()

	if err := dest.SetComment(zipReader.Comment); err != nil {
		return nil, fmt.Errorf("failed to set comment for output archive: %s", err)
	}

	for _, file := range zipReader.File {
		tmp, err := repacker.Relativer(file.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to repack: %s", err)
		}
		fileName := repacker.Destinator(tmp)

		isDir := file.FileInfo().IsDir()

		fh := file.FileHeader
		fh.Name = fileName

		fileWriter, err := dest.CreateHeader(&fh)
		if err != nil {
			return nil, fmt.Errorf("failed to copy attributes for %s: %s", fileName, err)
		}

		if isDir {
			continue
		}

		fileData, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file for %s: %s", fileName, err)
		}

		if _, err := io.Copy(fileWriter, fileData); err != nil {
			fileData.Close()
			return nil, fmt.Errorf("failed to copy content for %s: %s", fileName, err)
		}

		fileData.Close()
	}

	return result, nil
}
