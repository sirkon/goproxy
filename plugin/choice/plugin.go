package choice

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"

	"github.com/sirkon/goproxy"
)

// New plugin which tries to return a source with each plugin consequently until success. Made specially for
// apriori plugin
func New(plugins ...goproxy.Plugin) goproxy.Plugin {
	return &choice{
		plugs: plugins,
	}
}

type choice struct {
	plugs []goproxy.Plugin
}

func (c *choice) String() string {
	plugs := make([]string, len(c.plugs))
	for i, plug := range c.plugs {
		plugs[i] = plug.String()
	}
	return fmt.Sprintf("choice(%s)", strings.Join(plugs, ", "))
}

func (c *choice) Module(req *http.Request, prefix string) (goproxy.Module, error) {
	for _, plug := range c.plugs {
		src, err := plug.Module(req, prefix)
		if err != nil {
			continue
		}
		return src, nil
	}
	return nil, errors.Errorf("no suitable plugin found for request to %s", req.URL.Path)
}

func (c *choice) Leave(source goproxy.Module) error {
	return nil
}

func (c *choice) Close() error {
	return nil
}
