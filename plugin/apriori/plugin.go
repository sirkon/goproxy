package apriori

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sirkon/goproxy"
)

// ModuleInfo information needed for go modules proxy protocol
type ModuleInfo struct {
	RevInfo     goproxy.RevInfo
	GoModPath   string
	ArchivePath string
}

// Mapping maps path → (version → module info)
type Mapping map[string]map[string]ModuleInfo

// NewPlugin "apriori" - "cache" is boring: some file may contains information
// <mod path> → <version> → (<rev info>, <go.mod path>, <zip archive path>) and what it hidden there is enough for a
// functional go proxy serving exactly these modules at exactly these versions
func NewPlugin(path string) (goproxy.Plugin, error) {
	var res plugin
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("no apriori info file found: %s", err)
	}
	if err := json.Unmarshal(data, &res.mapping); err != nil {
		return nil, fmt.Errorf("invalid apriori info file %s: %s", path, err)
	}
	return &res, nil
}

type plugin struct {
	mapping Mapping
}

func (p *plugin) Module(req *http.Request, prefix string) (goproxy.Module, error) {
	mod, _, err := goproxy.GetModInfo(req, prefix)
	if err != nil {
		return nil, err
	}
	modInfo, ok := p.mapping[mod]
	if !ok {
		return nil, fmt.Errorf("no module %s found in cache", mod)
	}
	return &aprioriModule{path: mod, mod: modInfo}, nil
}

func (p *plugin) Leave(source goproxy.Module) error {
	return nil
}

func (p *plugin) Close() error {
	return nil
}

func (p *plugin) String() string {
	return "cache"
}
