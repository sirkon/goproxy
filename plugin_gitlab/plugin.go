package gitlab

import (
	"fmt"
	"net/http"
	path2 "path"
	"strings"

	"github.com/sirkon/gitlab"

	"github.com/sirkon/goproxy"
)

// plugin of sources for gitlab
type plugin struct {
	apiAccess gitlab.APIAccess
	needAuth  bool
	token     string
}

func (f *plugin) String() string {
	return "gitlab"
}

// NewPlugin constructor
func NewPlugin(access gitlab.APIAccess, needAuth bool) goproxy.Plugin {
	return &plugin{
		apiAccess: access,
		needAuth:  needAuth,
	}
}

// NewPluginToken constructor
func NewPluginToken(access gitlab.APIAccess, token string) goproxy.Plugin {
	return &plugin{
		apiAccess: access,
		token:     token,
		needAuth:  true,
	}
}

// NewPluginGitlabClient constructor with given gitlab apiAccess
func NewPluginGitlabClient(needAuth bool, access gitlab.APIAccess) goproxy.Plugin {
	return &plugin{
		apiAccess: access,
		needAuth:  needAuth,
	}
}

// NewPluginGitlabTokenClient constructor with given gitlab apiAccess
func NewPluginGitlabTokenClient(token string, access gitlab.APIAccess) goproxy.Plugin {
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

func (f *plugin) Module(req *http.Request, prefix string) (goproxy.Module, error) {
	path, _, err := goproxy.GetModInfo(req, prefix)
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

	// cut the tail and see if it denounces version suffix (vXYZ)
	pos := strings.LastIndexByte(fullPath, '/')
	if pos < 0 {
		return &gitlabModule{
			client:          f.apiAccess.Client(token),
			fullPath:        fullPath,
			path:            path,
			pathUnversioned: path,
			major:           0,
		}, nil
	}

	tail := fullPath[pos+1:]
	var ve pathVersionExtractor
	if ok, _ := ve.Extract(tail); !ok {

		return &gitlabModule{
			client:          f.apiAccess.Client(token),
			fullPath:        fullPath,
			path:            path,
			pathUnversioned: path,
			major:           0,
		}, nil
	}

	unversionedPath, _ := path2.Split(path)
	return &gitlabModule{
		client:          f.apiAccess.Client(token),
		fullPath:        fullPath,
		path:            path,
		pathUnversioned: strings.Trim(unversionedPath, "/"),
		major:           ve.Version,
	}, nil
}

func (f *plugin) Leave(source goproxy.Module) error {
	return nil
}

func (f *plugin) Close() error {
	return nil
}
