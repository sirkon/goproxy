package gitlab

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/rs/zerolog"
	"github.com/sirkon/gitlab"
	"github.com/sirkon/gitlab/gitlabdata"

	"github.com/sirkon/goproxy/internal/errors"

	"github.com/sirkon/goproxy"
	"github.com/sirkon/goproxy/fsrepack"
	"github.com/sirkon/goproxy/gomod"
	"github.com/sirkon/goproxy/semver"
)

type gitlabModule struct {
	client          gitlab.Client
	fullPath        string
	path            string
	pathUnversioned string
	major           int
}

func (s *gitlabModule) ModulePath() string {
	return s.path
}

func (s *gitlabModule) Versions(ctx context.Context, prefix string) ([]string, error) {
	tags, err := s.getVersions(ctx, prefix, s.pathUnversioned)
	if err == nil {
		return tags, err
	}

	if s.pathUnversioned == s.path {
		return nil, err
	}
	zerolog.Ctx(ctx).Warn().Err(err).Msgf("failed to get with unversioned path `%s` (original %s), someone is loving cars a bit too much :)", s.pathUnversioned, s.path)
	return s.getVersions(ctx, prefix, s.path)
}

func (s *gitlabModule) getVersions(ctx context.Context, prefix string, path string) ([]string, error) {
	tags, err := s.client.Tags(ctx, path, "")
	if err != nil {
		return nil, errors.Wrapf(err, "gitlab getting tags from gitlab repository")
	}

	var resp []string
	for _, tag := range tags {
		if semver.IsValid(tag.Name) {
			resp = append(resp, tag.Name)
		}
	}
	if len(resp) > 0 {
		return resp, nil
	}
	info, err := s.getStat(ctx, "master")
	if err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).Msg("getting revision info for master")
		return nil, errors.Newf("gitlab no tags found in the current repo")
	}
	return []string{info.Version}, nil
}

func (s *gitlabModule) Stat(ctx context.Context, rev string) (*goproxy.RevInfo, error) {
	res, err := s.getStat(ctx, rev)
	if err != nil {
		return nil, err
	}

	if major := semver.Major(res.Version); major >= 2 && s.major < major {
		return nil, errors.Newf("gitlab branch relates to higher major version v%d than what was expected from module path (v%d)", major, s.major)
	}
	return res, nil
}

func (s *gitlabModule) getStat(ctx context.Context, rev string) (res *goproxy.RevInfo, err error) {
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

	tags, err := s.client.Tags(ctx, s.pathUnversioned, rev)
	if err != nil {
		tags, err = s.client.Tags(ctx, s.path, rev)
		if err != nil {
			return nil, errors.Wrapf(err, "gitlab getting tags from gitlab repository")
		}
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

	return nil, errors.Newf("gitlab state: unknown revision %s for %s", rev, s.path)
}

func (s *gitlabModule) statWithPseudoVersion(ctx context.Context, rev string) (*goproxy.RevInfo, error) {
	commits, err := s.client.Commits(ctx, s.pathUnversioned, rev)
	if err != nil {
		commits, err = s.client.Commits(ctx, s.path, rev)
		if err != nil {
			return nil, errors.Wrapf(err, "getting commits for `%s`", rev)
		}
	}
	if len(commits) == 0 {
		return nil, errors.Newf("no commits found for revision %s", rev)
	}

	commitMap := make(map[string]*gitlabdata.Commit, len(commits))
	for _, commit := range commits {
		commitMap[commit.ID] = commit
	}

	// looking for the most recent semver tag
	tags, err := s.client.Tags(ctx, s.pathUnversioned, "") // all tags
	if err != nil {
		tags, err = s.client.Tags(ctx, s.path, "")
		if err != nil {
			return nil, errors.Wrapf(err, "getting tags")
		}
	}
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

	var base string
	if semver.Major(maxVer) < s.major {
		base = fmt.Sprintf("v%d.0.0-", s.major)
	} else {
		major, minor, patch := semver.MajorMinorPatch(maxVer)
		base = fmt.Sprintf("v%d.%d.%d-0.", major, minor, patch+1)
	}

	// Should set appropriate version
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
	pseudoVersion := fmt.Sprintf("%s%s%s%s%s%s%s-%s",
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
	goMod, err := s.getGoMod(ctx, version)
	if err != nil {
		if os.IsNotExist(err) {
			return []byte("module " + s.fullPath), nil
		}
		return nil, errors.Wrap(err, "gitlab getting go.mod")
	}

	res, err := gomod.Parse("go.mod", goMod)
	if err != nil {
		return nil, errors.Wrapf(err, "gitlab parsing repository go.mod")
	}

	if res.Name != s.fullPath {
		return nil, errors.Newf("gitlab module path is not equal to go.mod module path: %s ≠ %s", res.Name, s.fullPath)
	}

	return goMod, nil
}

func (s *gitlabModule) getGoMod(ctx context.Context, version string) ([]byte, error) {
	// try with pseudo version first
	if sha := semver.Pseudo(version); len(sha) > 0 {
		res, err := s.client.File(ctx, s.pathUnversioned, "go.mod", sha)
		if err == nil {
			return res, nil
		}
		res, err = s.client.File(ctx, s.path, "go.mod", sha)
		if err == nil {
			return res, nil
		}
	}
	res, err := s.client.File(ctx, s.pathUnversioned, "go.mod", version)
	if err == nil {
		return res, nil
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
	modInfo, err := s.client.ProjectInfo(ctx, s.pathUnversioned)
	if err != nil {
		modInfo, err = s.client.ProjectInfo(ctx, s.path)
		if err != nil {
			return nil, errors.Wrapf(err, "gitlab getting project %s info", s.path)
		}
	}

	archive, err := s.client.Archive(ctx, modInfo.ID, revision)
	if err != nil {
		return nil, errors.Wrap(err, "getting zipped archive data")
	}

	repacker, err := fsrepack.Gitlab(s.fullPath, version)
	if err != nil {
		return nil, errors.Wrap(err, "gitlab initiating repacker for gitlab archive source")
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
		return nil, errors.Wrap(err, "gitlab reading gitlab source archive")
	}

	zipReader, err := zip.NewReader(bytes.NewReader(zipped), int64(len(zipped)))
	if err != nil {
		return nil, errors.Wrap(err, "gitlab extracting zipped source data")
	}

	rawDest := &bufferCloser{}
	result := rawDest
	dest := zip.NewWriter(rawDest)
	defer func() {
		if err := dest.Close(); err != nil {
			logger := zerolog.Ctx(ctx)
			logger.Error().Err(err).Msgf("closing an output archive")
		}
	}()

	if err := dest.SetComment(zipReader.Comment); err != nil {
		return nil, errors.Wrap(err, "setting comment to output archive")
	}

	for _, file := range zipReader.File {
		tmp, err := repacker.Relativer(file.Name)
		if err != nil {
			return nil, errors.Wrap(err, "gitlab relative file name computation")
		}
		fileName := repacker.Destinator(tmp)

		isDir := file.FileInfo().IsDir()

		fh := file.FileHeader
		fh.Name = fileName

		fileWriter, err := dest.CreateHeader(&fh)
		if err != nil {
			return nil, errors.Wrapf(err, "copying attributes for %s", fileName)
		}

		if isDir {
			continue
		}

		fileData, err := file.Open()
		if err != nil {
			return nil, errors.Wrapf(err, "opening file for %s", fileName)
		}

		if _, err := io.Copy(fileWriter, fileData); err != nil {
			if cErr := fileData.Close(); cErr != nil {
				logger := zerolog.Ctx(ctx)
				logger.Error().Err(cErr).Msgf("closing a file for %s", fileName)
			}
			return nil, errors.Wrapf(err, "copying content for %s", fileName)
		}

		if err := fileData.Close(); err != nil {
			return nil, errors.Wrapf(err, "closing zip file for %s", fileName)
		}
	}

	return result, nil
}
