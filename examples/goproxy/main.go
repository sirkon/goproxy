//go:generate bash build/generate.sh

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	gitlab2 "github.com/sirkon/gitlab"

	"github.com/sirkon/goproxy"
	"github.com/sirkon/goproxy/router"
	"github.com/sirkon/goproxy/source/gitlab"
	"github.com/sirkon/goproxy/source/vcs"
)

var listen string
var cacheDir string
var gitlabAPIURL string

func init() {
	flag.StringVar(&cacheDir, "cache-dir", "", "go modules cache dir")
	flag.StringVar(&gitlabAPIURL, "gitlab-api-url", "", "gitlab host to get modules from")
	flag.StringVar(&listen, "listen", "0.0.0.0:8081", "service listen address")
	flag.Parse()
}

func main() {
	if len(cacheDir) == 0 {
		fmt.Print("cached dir must be set")
		flag.Usage()
		os.Exit(1)
	}

	writer := zerolog.NewConsoleWriter()
	writer.TimeFormat = time.RFC3339
	writer.FormatMessage = func(i interface{}) string {
		return fmt.Sprintf("\033[1m%v\033[0m", i)
	}
	writer.FormatTimestamp = func(i interface{}) string {
		if i == nil {
			return ""
		}
		return fmt.Sprintf("\033[2m%v\033[0m", i)
	}
	writer.FormatFieldName = func(i interface{}) string {
		return fmt.Sprintf("\033[35m%s\033[0m", i)
	}
	writer.FormatFieldValue = func(i interface{}) string {
		return fmt.Sprintf("[%v]", i)
	}
	writer.FormatErrFieldName = func(i interface{}) string {
		return fmt.Sprintf("\033[31m%s\033[0m", i)
	}
	writer.FormatErrFieldValue =
		func(i interface{}) string {
			return fmt.Sprintf("\033[31m[%v]\033[0m", i)
		}
	log := zerolog.New(writer).Level(zerolog.DebugLevel)

	errCh := make(chan error)

	log.Info().Timestamp().Str("listen", listen).Msg("start listening")

	r, err := router.NewRouter()
	if err != nil {
		log.Fatal().Err(err)
	}

	legacy, err := vcs.NewPlugin(cacheDir)
	if err != nil {
		log.Fatal().Err(err).Msg("exitting")
	}
	if err := r.AddRoute("", legacy); err != nil {
		log.Fatal().Err(err).Msg("exitting")
	}

	if len(gitlabAPIURL) > 0 {
		gl := gitlab.NewPlugin(gitlab2.NewAPIAccess(nil, gitlabAPIURL), true)
		if err := r.AddRoute("gitlab", gl); err != nil {
			log.Fatal().Err(err).Msg("exitting")
		}
	}

	m := goproxy.Middleware(r, "", &log)

	server := http.Server{
		Addr:    listen,
		Handler: m,
	}

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			errCh <- err
		}
	}()

	signCh := make(chan os.Signal)
	signal.Notify(signCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		log.Fatal().Timestamp().Err(err).Msg("exitting")
	case sign := <-signCh:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		log.Info().Timestamp().Str("signal", sign.String()).Msg("server stopped on signal")
	}
}
