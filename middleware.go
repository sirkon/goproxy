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

// Middleware acts as go proxy with given router
func Middleware(r *router.Router) http.Handler {
	return middleware{
		router: r,
	}
}

// middleware
type middleware struct {
	router *router.Router
}

func (m middleware) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path, suffix, err := source.GetModInfo(req)
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

	src, err := factory.Source(req)
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
		info, err := src.Stat(req.Context(), getVersion(suffix, 5))
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
		arciveReader, err := src.Zip(req.Context(), getVersion(suffix))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer func() {
			if err := arciveReader.Close(); err != nil {
				log.Printf("failed to close zip archive reader for %s@%s: %s", path, getVersion(suffix), err)
			}
		}()
		if _, err := io.Copy(w, arciveReader); err != nil {
			log.Printf("failed to write module version zip: %s", err)
		}
	default:
		log.Printf("unsupported suffix %s in %s", suffix, req.URL)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

func getVersion(suffix string, offs ...int) string {
	var offset int
	if len(offs) > 0 {
		offset = offs[0]
	} else {
		offset = 4
	}
	return suffix[:len(suffix)-offset]
}
