package goproxy

import (
	"net/http"
	"strings"

	"github.com/sirkon/goproxy/internal/errors"

	"github.com/sirkon/goproxy/internal/module"
)

const (
	slashDogVSlash      = "/@v/"
	constSlashDogLatest = "/@latest"
)

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
		return false, errors.Newf("/<url> expected, got %s", line)
	}

	// Take until "/@v/" as Module(string)
	pos = strings.Index(p.Rest, slashDogVSlash)
	if pos >= 0 {
		p.Module = p.Rest[:pos]
		p.Rest = p.Rest[pos+len(slashDogVSlash):]
	} else {
		return false, errors.Newf("/@v/ was not found in %s", p.Rest)
	}

	// Take the rest as Suffix(string)
	p.Suffix = p.Rest
	p.Rest = p.Rest[len(p.Rest):]
	return true, nil
}

// latestExtraction ...
type latestExtraction struct {
	Rest string
	Path string
}

// Extract ...
func (p *latestExtraction) Extract(line string) (bool, error) {
	p.Rest = line
	var pos int

	// Checks if the rest starts with '/' and pass it
	if len(p.Rest) >= 1 && p.Rest[0] == '/' {
		p.Rest = p.Rest[1:]
	} else {
		return false, nil
	}

	// Take until "/@latest" as Path(string)
	pos = strings.Index(p.Rest, constSlashDogLatest)
	if pos >= 0 {
		p.Path = p.Rest[:pos]
		p.Rest = p.Rest[pos+len(constSlashDogLatest):]
	} else {
		return false, nil
	}

	return true, nil
}

// GetModInfo retrieves mod info from URL
func GetModInfo(req *http.Request, prefix string) (path string, suffix string, err error) {
	method := req.URL.Path
	if !strings.HasPrefix(method, prefix) {
		err = errors.Newf("request URL path expected to be a %s*, got %s", prefix, method)
	}
	method = method[len(prefix):]

	var ok bool
	var le latestExtraction
	var e modInfoExtraction

	if ok, _ = le.Extract(method); ok {
		path = le.Path
		suffix = "latest"

	} else if ok, err = e.Extract(method); ok {
		path = e.Module
		suffix = e.Suffix
	} else {
		err = errors.Wrapf(err, "invalid go proxy request, wrong URL %s", method)
		return
	}

	path, err = module.DecodePath(path)
	if err != nil {
		return
	}

	return
}

// PathEncoding returns go module encoded path
func PathEncoding(path string) (string, error) {
	return module.EncodePath(path)
}
