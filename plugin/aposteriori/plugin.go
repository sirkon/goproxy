package aposteriori

import (
	"io"
	"net/http"
	"sync"

	"github.com/sirkon/goproxy/internal/errors"

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

// NewCachePriority aposteriori plugin constructor with cache-priority behavior
func NewCachePriority(next goproxy.Plugin, cache FileCache, availablity map[string]map[string]struct{}) goproxy.Plugin {
	return &plugin{next: next, cache: cache, registry: availablity}
}

type plugin struct {
	sync.Mutex
	next     goproxy.Plugin
	cache    FileCache
	registry map[string]map[string]struct{} // registry is <module path> → <version>
}

func (p *plugin) Module(req *http.Request, prefix string) (goproxy.Module, error) {
	next, err := p.next.Module(req, prefix)
	if err != nil {
		return nil, errors.Wrapf(err, "aposteriori delegation error")
	}

	return &module{
		next:   next,
		parent: p,
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
