package fsrepack

import (
	"fmt"
	"strings"

	"github.com/sirkon/goproxy/internal/str"
)

// FSRepacker methods for names transformations during repack process.
// For instance, gitlab.com returns `project-name@version/vX.Y.Z and go modules proxy needs `gitlab.com/<owner>/project-name>[/vX for X >= 2]`
// The process to transform a path is two-phased:
type FSRepacker interface {
	// Relativer returns relative path for given file name against predefined root
	Relativer(path string) (string, error)

	// Destinator joins given path with some predefined root
	Destinator(path string) string
}

// Gitlab returns repacker for gitlab output
func Gitlab(projectPath string, version int) FSRepacker {
	return gitlab(projectPath, version)
}

func gitlab(projectPath string, version int) gitlabRepacker {
	return gitlabRepacker{
		version:     version,
		projectPath: strings.Trim(projectPath, "/"),
	}
}

type gitlabRepacker struct {
	version     int
	projectPath string
}

func (r gitlabRepacker) Relativer(path string) (string, error) {
	origPath := path
	path = strings.Trim(path, "/")
	items := strings.Split(path, "/")
	if len(items) == 0 {
		return "", fmt.Errorf("wrong path `%s`", origPath)
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
	if r.version > 1 {
		prefix = fmt.Sprintf("%s/v%d", r.projectPath, r.version)
	}
	return prefix + "/" + path
}

// Standard returns so called "standard" repacker. Standard means it is a typical situation when you need to get rid of
// some a1/a2/…/an prefix
func Standard(root string, projectPath string, version int) FSRepacker {
	return standard{
		gitlabRepacker: gitlab(projectPath, version),
		expectedPrefix: strings.Trim(projectPath, "/"),
	}
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
		return "", fmt.Errorf("wrong path `%s` against root %s", origPath, r.expectedPrefix)
	}
	if strings.HasSuffix(origPath, "/") {
		path += "/"
	}
	return path, nil
}
