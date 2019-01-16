package cascade

import (
	"fmt"
	"net/http"

	"github.com/sirkon/goproxy/source"
)

// NewFactory factory returning source pointing to another proxy
func NewFactory(url string) source.Factory {
	return &factory{url: url, client: &http.Client{}}
}

// factory of sources for another go proxy
type factory struct {
	client *http.Client
	url    string
}

func (f *factory) Source(req *http.Request) (source.Source, error) {
	path, _, err := source.GetModInfo(req)
	if err != nil {
		return nil, fmt.Errorf("%s invalid request: %s", req.URL, err)
	}

	return &cascadeSource{
		mod:    path,
		url:    f.url,
		client: f.client,
	}, nil
}

func (f *factory) Leave(source source.Source) error {
	return nil
}

func (f *factory) Close() error {
	return nil
}
