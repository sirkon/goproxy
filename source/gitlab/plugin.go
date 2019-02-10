package gitlab

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/sirkon/goproxy/source"
)

// plugin of sources for gitlab
type plugin struct {
	client   Client
	needAuth bool
	token    string
}

// NewPlugin constructor
func NewPlugin(url string, needAuth bool) source.Plugin {
	return &plugin{
		client:   NewClient(url, &http.Client{}),
		needAuth: needAuth,
	}
}

// NewPluginToken constructor
func NewPluginToken(url, token string) source.Plugin {
	return &plugin{
		client:   NewClient(url, &http.Client{}),
		token:    token,
		needAuth: true,
	}
}

// NewPluginGitlabClient constructor with given gitlab client
func NewPluginGitlabClient(needAuth bool, client Client) source.Plugin {
	return &plugin{
		client:   client,
		needAuth: needAuth,
	}
}

// NewPluginGitlabTokenClient constructor with given gitlab client
func NewPluginGitlabTokenClient(token string, client Client) source.Plugin {
	return &plugin{
		client:   client,
		token:    token,
		needAuth: true,
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
		token:    token,
		fullPath: fullPath,
		path:     path,
		client:   f.client,
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
		token:    token,
		fullPath: fullPath,
		path:     path,
		client:   f.client,
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
