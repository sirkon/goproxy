package cascade

import (
	"fmt"
	"net/http"

	"github.com/sirkon/goproxy"
	"github.com/sirkon/goproxy/internal/module"
)

// NewPlugin plugin returning source pointing to another proxy
func NewPlugin(url string) goproxy.Plugin {
	return &plugin{url: url, client: &http.Client{}, passCreds: nil}
}

// NewPluginPassCreds this gets a function deciding is it worth to pass BasicAuth further
func NewPluginPassCreds(url string, passCreds func(r *http.Request) bool) goproxy.Plugin {
	return &plugin{url: url, client: &http.Client{}, passCreds: passCreds}
}

// plugin of sources for another go proxy
type plugin struct {
	client    *http.Client
	url       string
	passCreds func(req *http.Request) bool
}

func (f *plugin) String() string {
	return "cascade"
}

func (f *plugin) Module(req *http.Request, prefix string) (goproxy.Module, error) {
	path, _, err := goproxy.GetModInfo(req, prefix)
	if err != nil {
		return nil, fmt.Errorf("%s invalid request: %s", req.URL.Path, err)
	}
	reqPath, err := module.EncodePath(path)
	if err != nil {
		return nil, fmt.Errorf("%is invalid request: %s", req.URL.Path, err)
	}

	res := &cascadeModule{
		mod:    path,
		reqMod: reqPath,
		url:    f.url,
		client: f.client,
	}
	if f.passCreds != nil {
		if user, pass, ok := req.BasicAuth(); ok && f.passCreds(req) {
			res.basicAuth.ok = true
			res.basicAuth.user = user
			res.basicAuth.password = pass
		}
	}

	return res, nil
}

func (f *plugin) Leave(source goproxy.Module) error {
	return nil
}

func (f *plugin) Close() error {
	return nil
}
