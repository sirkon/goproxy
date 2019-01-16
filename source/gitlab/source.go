package gitlab

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/sirkon/goproxy/internal/semver"
	"github.com/sirkon/goproxy/source"
)

type gitlabSource struct {
	client   Client
	token    string
	fullPath string
	path     string
	zipDir   string
}

func (s *gitlabSource) ModulePath() string {
	return s.path
}

func (s *gitlabSource) Versions(ctx context.Context, prefix string) ([]string, error) {
	data, err := s.client.Tags(ctx, s.path, "", s.token)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags from gitlab repository: %s", err)
	}

	type versionInfo struct {
		Name string `json:"name"`
	}
	var dest []versionInfo
	if err := json.Unmarshal(data, &dest); err != nil {
		return nil, fmt.Errorf("failed to parse response: %s", err)
	}

	var resp []string
	for _, tag := range dest {
		if semver.IsValid(tag.Name) {
			resp = append(resp, tag.Name)
		}
	}
	if len(resp) == 0 {
		return nil, fmt.Errorf("invalid repository %s", s.path)
	}

	return resp, nil
}

func (s *gitlabSource) Stat(ctx context.Context, rev string) (*source.RevInfo, error) {
	data, err := s.client.Tags(ctx, s.path, rev, s.token)
	if err != nil {
		return nil, fmt.Errorf("failed to get tag %s from gitlab repository: %s", rev, err)
	}

	var dest struct {
		Name   string `json:"name"`
		Commit struct {
			ID        string `json:"id"`
			ShortID   string `json:"short_id"`
			CreatedAt string `json:"created_at"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(data, &dest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tag %s info: %s", rev, err)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, dest.Commit.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("invalid time format in tag response `%s`: %s", dest.Commit.CreatedAt, err)
	}

	return &source.RevInfo{
		Version: dest.Name,
		Time:    createdAt,
		Name:    dest.Commit.ID,
		Short:   dest.Commit.ShortID,
	}, nil
}

func (s *gitlabSource) GoMod(ctx context.Context, version string) (data []byte, err error) {
	return s.client.GoMod(ctx, s.path, version, s.token)
}

type bufferCloser struct {
	bytes.Buffer
}

// Close makes bufferCloser io.ReadCloser
func (*bufferCloser) Close() error { return nil }

func (s *gitlabSource) Zip(ctx context.Context, version string) (io.ReadCloser, error) {
	modInfo, err := s.client.ModuleInfo(ctx, s.path, s.token)
	if err != nil {
		return nil, fmt.Errorf("failed to get project %s info: %s", s.path, err)
	}

	var prjInfo projectInfo
	if err := json.Unmarshal(modInfo, &prjInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project %s info data: %s", s.path, err)
	}

	archive, err := s.client.Archive(ctx, strconv.Itoa(prjInfo.ID), version, s.token)
	if err != nil {
		return nil, fmt.Errorf("failed to get zipped archive data: %s", err)
	}

	// now need to repack archive content from <pkg-name>-<hash> → <full pkg name, such as gitlab.com/user/module>
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
		pathChunks := strings.Split(file.Name, "/")
		pathChunks[0] = s.fullPath + "@" + version
		fileName := strings.Join(pathChunks, "/")

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
