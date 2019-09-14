package goproxy

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/rs/zerolog"
	"github.com/spaolacci/murmur3"

	"github.com/sirkon/goproxy/semver"
)

// Middleware acts as go proxy with given router.
//   transportPrefix is a head part of URL path which refers to address of go proxy before the module info. For example,
// if we serving go proxy at https://0.0.0.0:8081/goproxy/..., transportPrefix will be "/goproxy"
func Middleware(r *Router, transportPrefix string, logger *zerolog.Logger) http.Handler {
	return middleware{
		prefix: transportPrefix,
		router: r,
		logger: logger,
	}
}

// middleware
type middleware struct {
	prefix string
	router *Router
	logger *zerolog.Logger
}

const latestSuffix = "/@latest"

func (m middleware) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	hasher := murmur3.New64()
	_, _ = io.WriteString(hasher, req.URL.String())
	_, _ = io.WriteString(hasher, time.Now().Format(time.RFC3339Nano))
	logger := m.logger.With().Hex("request-id", hasher.Sum(nil)).Str("request", req.URL.String()).Logger()

	path, suffix, err := GetModInfo(req, m.prefix)
	if err != nil {
		logger.Error().Err(err).Str("prefix", m.prefix).Msg("getting mod info")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	logger = logger.With().Str("module", path).Logger()

	factory := m.router.Factory(path)
	if factory == nil {
		logger.Error().Msgf("no plugin registered for %s", path)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	logger = logger.With().Str("plugin", factory.String()).Logger()

	src, err := factory.Module(req, m.prefix)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get a source from plugin")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch {
	case suffix == "list":
		ctx := logger.WithContext(req.Context())
		logger.Debug().Msg("version list requested")
		version, err := src.Versions(ctx, "")
		if err != nil {
			logger.Error().Err(err).Msg("failed to get version list")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		if _, err := io.WriteString(w, strings.Join(version, "\n")); err != nil {
			logger.Error().Err(err).Msg("failed to write list response")
		} else {
			logger.Debug().Msg("version list done")
		}
	case strings.HasSuffix(suffix, ".info"):
		version := getVersion(suffix)
		tmpLogger := logger.With().Str("version", version).Logger()
		ctx := tmpLogger.WithContext(req.Context())
		tmpLogger.Debug().Msg("version info requested")
		info, err := src.Stat(ctx, version)
		if err != nil {
			tmpLogger.Error().Err(err).Msg("failed to get revision info from source beneath")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		je := json.NewEncoder(w)
		if err := je.Encode(info); err != nil {
			tmpLogger.Error().Err(err).Msg("failed to write version info response")
		} else {
			tmpLogger.Debug().Msg("version info done")
		}
	case strings.HasSuffix(suffix, ".mod"):
		version := getVersion(suffix)
		tmpLogger := logger.With().Str("version", version).Logger()
		ctx := tmpLogger.WithContext(req.Context())
		tmpLogger.Debug().Msg("go.mod requested")
		gomod, err := src.GoMod(ctx, version)
		if err != nil {
			tmpLogger.Error().Err(err).Msg("failed to get go.mod from a source beneath")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if _, err := w.Write(gomod); err != nil {
			tmpLogger.Error().Err(err).Msg("failed to return go.mod")
			return
		} else {
			tmpLogger.Debug().Msg("go.mod done")
		}
	case strings.HasSuffix(suffix, ".zip"):
		version := getVersion(suffix)
		tmpLogger := logger.With().Str("version", version).Logger()
		ctx := tmpLogger.WithContext(req.Context())
		tmpLogger.Debug().Msg("zip archive requested")
		archiveReader, err := src.Zip(ctx, version)
		if err != nil {
			tmpLogger.Error().Err(err).Msg("failed to get zip archive")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer func() {
			if err := archiveReader.Close(); err != nil {
				tmpLogger.Error().Err(err).Msgf("failed to close zip reachive reader")
			}
		}()
		if _, err := io.Copy(w, archiveReader); err != nil {
			tmpLogger.Error().Err(err).Msg("failed to return zip archive")
		} else {
			tmpLogger.Debug().Msg("zip done")
		}
	case suffix == "latest":
		ctx := logger.WithContext(req.Context())
		logger.Debug().Msg("latest")
		version, err := src.Versions(ctx, "")
		var revision string
		if err != nil {
			logger.Error().Err(err).Msg("failed to get version list")
			revision = "master"
		} else {
			for _, v := range version {
				if semver.IsValid(v) && (len(revision) == 0 || semver.Compare(v, revision) > 0) {
					revision = v
				}
			}
			if len(revision) == 0 {
				revision = "master"
			}
		}
		tmpLogger := logger.With().Str("version", revision).Logger()
		tmpLogger.Debug().Msg("version info requested")
		info, err := src.Stat(ctx, revision)
		if err != nil {
			tmpLogger.Error().Err(err).Msg("failed to get revision info from source beneath")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		je := json.NewEncoder(w)
		if err := je.Encode(info); err != nil {
			tmpLogger.Error().Err(err).Msg("failed to write version info response")
		} else {
			tmpLogger.Debug().Msgf("latest done")
		}
	default:
		logger.Error().Msgf("unsupported suffix %s", suffix)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

// getVersion we have something like v0.1.2.zip or v0.1.2.info or v0.1.2.zip in the suffix and need to cut the
func getVersion(suffix string) string {
	off := strings.LastIndex(suffix, ".")
	encoding := suffix[:off]

	var buf []byte
	bang := false
	for _, r := range encoding {
		if r >= utf8.RuneSelf {
			return encoding
		}
		if bang {
			bang = false
			if r < 'a' || 'z' < r {
				return encoding
			}
			buf = append(buf, byte(r+'A'-'a'))
			continue
		}
		if r == '!' {
			bang = true
			continue
		}
		if 'A' <= r && r <= 'Z' {
			return encoding
		}
		buf = append(buf, byte(r))
	}
	if bang {
		return encoding
	}
	return string(buf)
}
