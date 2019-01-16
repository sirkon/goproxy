package source

import (
	"fmt"
	"net/http"

	"github.com/sirkon/goproxy/source/internal"
)

//go:generate ldetool generate --go-string --package extraction internal/mod_info_extractor.lde

// GetModInfo retrieves mod info from URL
func GetModInfo(req *http.Request) (path string, suffix string, err error) {
	method := req.URL.Path
	var e extraction.ModInfoExtractor

	if ok, _ := e.Extract(method); !ok {
		err = fmt.Errorf("invalid go proxy request: wrong URL `%s`", method)
		return
	}

	path = e.Module
	suffix = e.Suffix
	return
}
