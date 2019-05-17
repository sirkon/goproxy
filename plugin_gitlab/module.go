package gitlab

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/sirkon/gitlab"
	"github.com/sirkon/gitlab/gitlabdata"

	"github.com/sirkon/goproxy"
	"github.com/sirkon/goproxy/fsrepack"
	"github.com/sirkon/goproxy/semver"
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
	if semver.IsValid(rev) {
		return s.statVersion(ctx, rev)
	}

	// revision looks like a branch or non-semver tag, need to build pseudo-version
	return s.statWithPseudoVersion(ctx, rev)
}

// statVersion processing for semver revision
func (s *gitlabModule) statVersion(ctx context.Context, rev string) (*goproxy.RevInfo, error) {
	// check if this rev does look like pseudo-version – will try statWithPseudoVersion in this case with short SHA
	pseudo := semver.Pseudo(rev)
	if len(pseudo) > 0 {
		res, err := s.statWithPseudoVersion(ctx, pseudo)
		if err == nil {
			// should use base version from the commit itself
			if semver.Compare(rev, res.Version) > 0 {
				res.Version = rev
			}
			return res, nil
		}
	}

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

func (s *gitlabModule) statWithPseudoVersion(ctx context.Context, rev string) (*goproxy.RevInfo, error) {
	commits, err := s.client.Commits(ctx, s.path, rev)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits for `%s`: %s", rev, err)
	}
	if len(commits) == 0 {
		return nil, fmt.Errorf("no commits found for revision %s", rev)
	}

	commitMap := make(map[string]*gitlabdata.Commit, len(commits))
	for _, commit := range commitMap {
		commitMap[commit.ID] = commit
	}

	// looking for the most recent semver tag
	tags, err := s.client.Tags(ctx, s.path, "") // all tags
	maxVer := "v0.0.0"
	for _, tag := range tags {
		if _, ok := commitMap[tag.Commit.ID]; !ok {
			continue
		}
		if !semver.IsValid(tag.Name) {
			continue
		}
		maxVer = semver.Max(maxVer, tag.Name)
	}

	// Should set appropriate version
	base := semver.Base(maxVer)
	commit := commits[0]

	moment := commit.CreatedAt
	var (
		year   = moment[:4]
		month  = moment[5:7]
		day    = moment[8:10]
		hour   = moment[11:13]
		minute = moment[14:16]
		second = moment[17:19]
	)
	pseudoVersion := fmt.Sprintf("%s-%s%s%s%s%s%s-%s",
		base,
		year, month, day, hour, minute, second,
		commit.ShortID,
	)
	return &goproxy.RevInfo{
		Version: pseudoVersion,
		Time:    moment,
	}, nil
}

func (s *gitlabModule) GoMod(ctx context.Context, version string) (data []byte, err error) {
	// try with pseudo version first
	if sha := semver.Pseudo(version); len(sha) > 0 {
		res, err := s.client.File(ctx, s.path, "go.mod", sha)
		if err == nil {
			return res, nil
		}
	}
	return s.client.File(ctx, s.path, "go.mod", version)
}

type bufferCloser struct {
	bytes.Buffer
}

// Close makes bufferCloser io.ReadCloser
func (*bufferCloser) Close() error { return nil }

func (s *gitlabModule) Zip(ctx context.Context, version string) (io.ReadCloser, error) {
	if sha := semver.Pseudo(version); len(sha) > 0 {
		res, err := s.getZip(ctx, sha, version)
		if err == nil {
			return res, nil
		}
	}
	return s.getZip(ctx, version, version)
}

func (s *gitlabModule) getZip(ctx context.Context, revision, version string) (io.ReadCloser, error) {
	modInfo, err := s.client.ProjectInfo(ctx, s.path)
	if err != nil {
		return nil, fmt.Errorf("failed to get project %s info: %s", s.path, err)
	}

	archive, err := s.client.Archive(ctx, modInfo.ID, revision)
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
