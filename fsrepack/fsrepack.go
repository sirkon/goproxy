package fsrepack

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/sirkon/goproxy/internal/str"
	"github.com/sirkon/goproxy/semver"
)

// FSRepacker methods for names transformations during repack process.
// For instance, gitlab.com returns `project-name@major/vX.Y.Z and go modules proxy needs `gitlab.com/<owner>/project-name>[/vX for X >= 2]`
// The process to transform a path is two-phased:
type FSRepacker interface {
	// Relativer returns relative path for given file name against predefined root
	Relativer(path string) (string, error)

	// Destinator joins given path with some predefined root
	Destinator(path string) string
}

// Gitlab returns repacker for gitlab output
func Gitlab(projectPath string, version string) (FSRepacker, error) {
	return gitlab(projectPath, version)
}

func gitlab(projectPath string, version string) (gitlabRepacker, error) {
	major := semver.Major(version)
	return gitlabRepacker{
		major:       major,
		version:     version,
		projectPath: strings.Trim(projectPath, "/"),
	}, nil
}

type gitlabRepacker struct {
	major       int
	version     string
	projectPath string
}

func (r gitlabRepacker) Relativer(path string) (string, error) {
	origPath := path
	path = strings.Trim(path, "/")
	items := strings.Split(path, "/")
	if len(items) == 0 {
		return "", errors.Errorf("wrong path `%s`", origPath)
	}
	items = items[1:]

	res := strings.Join(items, "/")
	if strings.HasSuffix(origPath, "/") {
		res += "/"
	}
	return res, nil
}

func (r gitlabRepacker) Destinator(path string) string {
	prefix := r.projectPath
	if r.major > 1 {
		prefix = fmt.Sprintf("%s/v%d", r.projectPath, r.major)
	}
	prefix += "@" + r.version
	return prefix + "/" + strings.TrimLeft(path, "/")
}

// Standard returns so called "standard" repacker. Standard means it is a typical situation when you need to get rid of
// some a1/a2/…/an prefix
func Standard(root string, projectPath string, version string) (FSRepacker, error) {
	repackerBeneath, err := gitlab(projectPath, version)
	if err != nil {
		return nil, errors.Errorf("failed to create path repacker: %s", err)
	}
	return standard{
		gitlabRepacker: repackerBeneath,
		expectedPrefix: strings.Trim(projectPath, "/"),
	}, nil
}

type standard struct {
	gitlabRepacker
	expectedPrefix string
}

func (r standard) Relativer(path string) (string, error) {
	origPath := path
	path = strings.Trim(path, "/")
	if str.HasPathPrefix(path, r.expectedPrefix) {
		path = strings.Trim(path[len(r.expectedPrefix):], "/")
	}
	if len(path) == 0 {
		return "", errors.Errorf("wrong path `%s` against root %s", origPath, r.expectedPrefix)
	}
	if strings.HasSuffix(origPath, "/") {
		path += "/"
	}
	return path, nil
}
