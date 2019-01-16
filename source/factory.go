package source

import (
	"net/http"
)

// Factory
type Factory interface {
	Source(req *http.Request) (Source, error)
	Leave(source Source) error
	Close() error
}
