package gitlab

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/sirkon/gitlab"

	"github.com/sirkon/goproxy/source"
)

// plugin of sources for gitlab
type plugin struct {
	apiAccess gitlab.APIAccess
	needAuth  bool
	token     string
}

// NewPlugin constructor
func NewPlugin(access gitlab.APIAccess, needAuth bool) source.Plugin {
	return &plugin{
		apiAccess: access,
		needAuth:  needAuth,
	}
}

// NewPluginToken constructor
func NewPluginToken(access gitlab.APIAccess, token string) source.Plugin {
	return &plugin{
		apiAccess: access,
		token:     token,
		needAuth:  true,
	}
}

// NewPluginGitlabClient constructor with given gitlab apiAccess
func NewPluginGitlabClient(needAuth bool, access gitlab.APIAccess) source.Plugin {
	return &plugin{
		apiAccess: access,
		needAuth:  needAuth,
	}
}

// NewPluginGitlabTokenClient constructor with given gitlab apiAccess
func NewPluginGitlabTokenClient(token string, access gitlab.APIAccess) source.Plugin {
	return &plugin{
		apiAccess: access,
		token:     token,
		needAuth:  true,
	}
}

func getGitlabPath(fullPath string) string {
	pos := strings.IndexByte(fullPath, '/')
	if pos >= 0 {
		return fullPath[pos+1:]
	}
	return fullPath
}

func (f *plugin) Source(req *http.Request, prefix string) (source.Source, error) {
	path, _, err := source.GetModInfo(req, prefix)
	if err != nil {
		return nil, err
	}
	// url prefix (gitlab.XXXX, etc) is not needed for gitlab projects
	fullPath := path
	path = getGitlabPath(fullPath)

	var token string
	if f.needAuth && len(f.token) == 0 {
		var ok bool
		token, _, ok = req.BasicAuth()
		if !ok || len(token) == 0 {
			return nil, fmt.Errorf("authorization required")
		}
	} else if f.needAuth {
		token = f.token
	}

	s1 := &gitlabSource{
		fullPath: fullPath,
		path:     path,
		client:   f.apiAccess.Client(token),
	}

	// cut the tail and see if it denounces version suffix (vXYZ)
	pos := strings.LastIndexByte(fullPath, '/')
	if pos < 0 {
		return overlapGitlabSource{
			sources: []source.Source{s1},
			version: 0,
		}, nil
	}

	tail := fullPath[pos+1:]
	var ve pathVersionExtractor
	if ok, _ := ve.Extract(tail); !ok {
		return overlapGitlabSource{
			sources: []source.Source{s1},
			version: 0,
		}, nil
	}

	fullPath = fullPath[:pos]
	path = getGitlabPath(fullPath)

	s2 := &gitlabSource{
		fullPath: fullPath,
		path:     path,
		client:   f.apiAccess.Client(token),
	}

	return overlapGitlabSource{
		sources: []source.Source{s1, s2},
		version: ve.Version,
	}, nil

}

func (f *plugin) Leave(source source.Source) error {
	return nil
}

func (f *plugin) Close() error {
	return nil
}
