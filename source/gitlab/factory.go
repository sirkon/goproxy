package gitlab

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/sirkon/goproxy/source"
)

// factory of sources for gitlab
type factory struct {
	client   Client
	needAuth bool
}

// NewFactory constructor
func NewFactory(url string, needAuth bool) source.Factory {
	return &factory{
		client:   NewClient(url, &http.Client{}),
		needAuth: needAuth,
	}
}

// NewFactoryGitlabClient constructor with given gitlab client
func NewFactoryGitlabClient(needAuth bool, client Client) *factory {
	return &factory{
		client:   client,
		needAuth: needAuth,
	}
}

func (f *factory) Source(req *http.Request, prefix string) (source.Source, error) {
	path, _, err := source.GetModInfo(req, prefix)
	if err != nil {
		return nil, err
	}
	// url prefix (gitlab.XXXX, etc) is not needed for gitlab projects
	fullPath := path
	pos := strings.IndexByte(path, '/')
	if pos >= 0 {
		path = path[pos+1:]
	}

	var token string
	if f.needAuth {
		var ok bool
		token, _, ok = req.BasicAuth()
		if !ok || len(token) == 0 {
			return nil, fmt.Errorf("authorization required")
		}
	}

	return &gitlabSource{
		token:    token,
		fullPath: fullPath,
		path:     path,
		client:   f.client,
	}, nil
}

func (f *factory) Leave(source source.Source) error {
	return nil
}

func (f *factory) Close() error {
	return nil
}
