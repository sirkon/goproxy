package cascade

import (
	"fmt"
	"net/http"

	"github.com/sirkon/goproxy"
)

// NewPlugin plugin returning source pointing to another proxy
func NewPlugin(url string) goproxy.Plugin {
	return &plugin{url: url, client: &http.Client{}}
}

// plugin of sources for another go proxy
type plugin struct {
	client *http.Client
	url    string
}

func (f *plugin) String() string {
	return "cascade"
}

func (f *plugin) Module(req *http.Request, prefix string) (goproxy.Module, error) {
	path, _, err := goproxy.GetModInfo(req, prefix)
	if err != nil {
		return nil, fmt.Errorf("%s invalid request: %s", req.URL, err)
	}

	res := &cascadeModule{
		mod:    path,
		url:    f.url,
		client: f.client,
	}
	if user, pass, ok := req.BasicAuth(); ok {
		res.basicAuth.ok = true
		res.basicAuth.user = user
		res.basicAuth.password = pass
	}

	return res, nil
}

func (f *plugin) Leave(source goproxy.Module) error {
	return nil
}

func (f *plugin) Close() error {
	return nil
}
