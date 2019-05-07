package cascade

import (
	"fmt"
	"net/http"

	"github.com/sirkon/goproxy/source"
)

// NewPlugin plugin returning source pointing to another proxy
func NewPlugin(url string) source.Plugin {
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

func (f *plugin) Source(req *http.Request, prefix string) (source.Source, error) {
	path, _, err := source.GetModInfo(req, prefix)
	if err != nil {
		return nil, fmt.Errorf("%s invalid request: %s", req.URL, err)
	}

	return &cascadeSource{
		mod:    path,
		url:    f.url,
		client: f.client,
	}, nil
}

func (f *plugin) Leave(source source.Source) error {
	return nil
}

func (f *plugin) Close() error {
	return nil
}
