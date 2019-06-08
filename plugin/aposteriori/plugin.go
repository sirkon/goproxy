package aposteriori

import (
	"fmt"
	"io"
	"net/http"

	"github.com/sirkon/goproxy"
)

// FileCache caching primitive
type FileCache interface {
	Get(name string) (io.ReadCloser, error)
	Set(name string, data io.Reader) error
}

// New aposteriori plugin constructor
func New(next goproxy.Plugin, cache FileCache) goproxy.Plugin {
	return &plugin{next: next, cache: cache}
}

type plugin struct {
	next  goproxy.Plugin
	cache FileCache
}

func (p *plugin) Module(req *http.Request, prefix string) (goproxy.Module, error) {
	next, err := p.next.Module(req, prefix)
	if err != nil {
		return nil, fmt.Errorf("aposteriori delegation error: %s", err)
	}

	return &module{
		next:  next,
		cache: p.cache,
	}, nil
}

func (p *plugin) Leave(source goproxy.Module) error {
	return nil
}

func (p *plugin) Close() error {
	return nil
}

func (p *plugin) String() string {
	return "aposteriori"
}
