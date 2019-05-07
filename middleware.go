package goproxy

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/rs/zerolog"

	"github.com/sirkon/goproxy/router"
	"github.com/sirkon/goproxy/source"
)

// Middleware acts as go proxy with given router.
//   transportPrefix is a head part of URL path which refers to address of go proxy before the module info. For example,
// if we serving go proxy at https://0.0.0.0:8081/goproxy/..., transportPrefix will be "/goproxy"
func Middleware(r *router.Router, transportPrefix string, logger *zerolog.Logger) http.Handler {
	return middleware{
		prefix: transportPrefix,
		router: r,
		logger: logger,
	}
}

// middleware
type middleware struct {
	prefix string
	router *router.Router
	logger *zerolog.Logger
}

func (m middleware) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path, suffix, err := source.GetModInfo(req, m.prefix)
	if err != nil {
		m.logger.Error().Err(err).Str("prefix", m.prefix).Msg("wrong request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	logger := m.logger.With().Str("request", req.URL.Path).Str("module", path)

	factory := m.router.Factory(path)
	if factory == nil {
		tmpLogger := logger.Logger()
		(&tmpLogger).Error().Msgf("no proxy handlers registered for %s", path)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	src, err := factory.Source(req, m.prefix)
	if err != nil {
		tmpLogger := logger.Logger()
		(&tmpLogger).Error().Err(err).Msgf("failed to get a source from plugin %s", factory)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch {
	case suffix == "list":
		tmpLogger := logger.Logger()
		ctx := (&tmpLogger).WithContext(req.Context())
		(&tmpLogger).Debug().Msg("version list requested")
		version, err := src.Versions(ctx, "")
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("failed to get version list")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		if _, err := io.WriteString(w, strings.Join(version, "\n")); err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("failed to write list response")
		}
	case strings.HasSuffix(suffix, ".info"):
		version := getVersion(suffix)
		tmpLogger := logger.Str("version", version).Logger()
		ctx := (&tmpLogger).WithContext(req.Context())
		(&tmpLogger).Debug().Msg("version info requested")
		info, err := src.Stat(ctx, version)
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("failed to get revision info from source beneath")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		je := json.NewEncoder(w)
		if err := je.Encode(info); err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("failed to write version info response")
		}
	case strings.HasSuffix(suffix, ".mod"):
		version := getVersion(suffix)
		tmpLogger := logger.Str("version", version).Logger()
		ctx := (&tmpLogger).WithContext(req.Context())
		(&tmpLogger).Debug().Msg("go.mod requested")
		gomod, err := src.GoMod(ctx, version)
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("failed to get go.mod from a source beneath")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if _, err := w.Write(gomod); err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("failed to return go.mod")
			return
		}
	case strings.HasSuffix(suffix, ".zip"):
		version := getVersion(suffix)
		tmpLogger := logger.Str("version", version).Logger()
		ctx := (&tmpLogger).WithContext(req.Context())
		(&tmpLogger).Debug().Msg("zip archive requested")
		archiveReader, err := src.Zip(ctx, version)
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("failed to get zip archive")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer func() {
			if err := archiveReader.Close(); err != nil {
				zerolog.Ctx(ctx).Error().Err(err).Msgf("failed to close zip reachive reader")
			}
		}()
		if _, err := io.Copy(w, archiveReader); err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("failed to return zip archive")
		}
	default:
		tmpLogger := logger.Logger()
		(&tmpLogger).Error().Msgf("unsupported suffix %s", suffix)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

// getVersion we have something like v0.1.2.zip or v0.1.2.info or v0.1.2.zip in the suffix and need to cut the
func getVersion(suffix string) string {
	off := strings.LastIndex(suffix, ".")
	return suffix[:off]
}
