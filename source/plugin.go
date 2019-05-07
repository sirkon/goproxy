package source

import (
	"net/http"
)

// Plugin gives a way to get a source object for a request
type Plugin interface {
	Source(req *http.Request, prefix string) (Source, error)
	Leave(source Source) error
	Close() error
	String() string
}
