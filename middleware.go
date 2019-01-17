package goproxy

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/sirkon/goproxy/router"
	"github.com/sirkon/goproxy/source"
)

// Middleware acts as go proxy with given router.
//   transportPrefix is a head part of URL path which refers to address of go proxy before the module info. For example,
// if we serving go proxy at https://0.0.0.0:8081/goproxy/..., transportPrefix will be "/goproxy"
func Middleware(r *router.Router, transportPrefix string) http.Handler {
	return middleware{
		prefix: transportPrefix,
		router: r,
	}
}

// middleware
type middleware struct {
	prefix string
	router *router.Router
}

func (m middleware) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path, suffix, err := source.GetModInfo(req, m.prefix)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Printf("requested for module %s, operation %s", path, suffix)

	factory := m.router.Factory(path)
	if factory == nil {
		log.Printf("no go proxy handlers registered for %s", path)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	src, err := factory.Source(req, "")
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch {
	case suffix == "list":
		version, err := src.Versions(req.Context(), "")
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		if _, err := io.WriteString(w, strings.Join(version, "\n")); err != nil {
			log.Printf("failed to write list response: %s", err)
		}
	case strings.HasSuffix(suffix, ".info"):
		info, err := src.Stat(req.Context(), getVersion(suffix))
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		je := json.NewEncoder(w)
		if err := je.Encode(info); err != nil {
			log.Printf("failed to write version info response: %s", err)
		}
	case strings.HasSuffix(suffix, ".mod"):
		gomod, err := src.GoMod(req.Context(), getVersion(suffix))
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if _, err := w.Write(gomod); err != nil {
			log.Printf("failed to write go.mod response: %s", err)
		}
	case strings.HasSuffix(suffix, ".zip"):
		archiveReader, err := src.Zip(req.Context(), getVersion(suffix))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer func() {
			if err := archiveReader.Close(); err != nil {
				log.Printf("failed to close zip archive reader for %s@%s: %s", path, getVersion(suffix), err)
			}
		}()
		if _, err := io.Copy(w, archiveReader); err != nil {
			log.Printf("failed to write module version zip: %s", err)
		}
	default:
		log.Printf("unsupported suffix %s in %s", suffix, req.URL)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

// getVersion we have something like v0.1.2.zip or v0.1.2.info or v0.1.2.zip in the suffix and need to cut the
func getVersion(suffix string) string {
	off := strings.LastIndex(suffix, ".")
	return suffix[:off]
}
