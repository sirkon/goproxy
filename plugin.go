package goproxy

import (
	"net/http"
)

// Plugin gives a way to get a source object for a request
type Plugin interface {
	Module(req *http.Request, prefix string) (Module, error)
	Leave(source Module) error
	Close() error
	String() string
}
