package source

import (
	"net/http"
)

// Factory gives a way to get a source object for a request
type Factory interface {
	Source(req *http.Request, prefix string) (Source, error)
	Leave(source Source) error
	Close() error
}
