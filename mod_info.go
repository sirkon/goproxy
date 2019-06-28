package goproxy

import (
	"net/http"
	"strings"

	"github.com/pkg/errors"

	"github.com/sirkon/goproxy/internal/module"
)

var slashDogVSlash = "/@v/"

// modInfoExtraction ...
type modInfoExtraction struct {
	Rest   string
	Module string
	Suffix string
}

// Extract ...
func (p *modInfoExtraction) Extract(line string) (bool, error) {
	p.Rest = line
	var pos int

	// Checks if the rest starts with '/' and pass it
	if len(p.Rest) >= 1 && p.Rest[0] == '/' {
		p.Rest = p.Rest[1:]
	} else {
		return false, nil
	}

	// Take until "/@v/" as Module(string)
	pos = strings.Index(p.Rest, slashDogVSlash)
	if pos >= 0 {
		p.Module = p.Rest[:pos]
		p.Rest = p.Rest[pos+len(slashDogVSlash):]
	} else {
		return false, nil
	}

	// Take the rest as Suffix(string)
	p.Suffix = p.Rest
	p.Rest = p.Rest[len(p.Rest):]
	return true, nil
}

// GetModInfo retrieves mod info from URL
func GetModInfo(req *http.Request, prefix string) (path string, suffix string, err error) {
	method := req.URL.Path
	if !strings.HasPrefix(method, prefix) {
		err = errors.Errorf("request URL path expected to be a %s*, got %s", prefix, method)
	}
	method = method[len(prefix):]
	var e modInfoExtraction

	if ok, _ := e.Extract(method); !ok {
		err = errors.Errorf("invalid go proxy request: wrong URL `%s`", method)
		return
	}

	path = e.Module
	path, err = module.DecodePath(path)
	if err != nil {
		return
	}

	suffix = e.Suffix
	return
}

// PathEncoding returns go module encoded path
func PathEncoding(path string) (string, error) {
	return module.EncodePath(path)
}
