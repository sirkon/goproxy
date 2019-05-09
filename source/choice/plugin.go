package choice

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/sirkon/goproxy/source"
)

// New plugin which tries to return a source with each plugin consequently until success. Made specially for
// apriori plugin
func New(plugins ...source.Plugin) source.Plugin {
	return &choice{
		plugs: plugins,
	}
}

type choice struct {
	plugs []source.Plugin
}

func (c *choice) String() string {
	plugs := make([]string, len(c.plugs))
	for i, plug := range c.plugs {
		plugs[i] = plug.String()
	}
	return fmt.Sprintf("choice(%s)", strings.Join(plugs, ", "))
}

func (c *choice) Source(req *http.Request, prefix string) (source.Source, error) {
	for _, plug := range c.plugs {
		src, err := plug.Source(req, prefix)
		if err != nil {
			continue
		}
		return src, nil
	}
	return nil, fmt.Errorf("no suitable plugin found for request to %s", req.URL.Path)
}

func (c *choice) Leave(source source.Source) error {
	return nil
}

func (c *choice) Close() error {
	return nil
}
