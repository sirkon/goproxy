//go:generate bash build/generate.sh

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirkon/goproxy"
	"github.com/sirkon/goproxy/router"
	"github.com/sirkon/goproxy/source/gitlab"
	"github.com/sirkon/goproxy/source/vcs"
)

var listen string
var cacheDir string
var gitlabHost string

func init() {
	log.SetOutput(os.Stdout)
	flag.StringVar(&cacheDir, "cache-dir", "", "go modules cache dir")
	flag.StringVar(&gitlabHost, "gitlab", "", "gitlab host to get modules from")
	flag.StringVar(&listen, "listen", "0.0.0.0:8081", "service listen address")
	flag.Parse()
}

func main() {
	if len(cacheDir) == 0 {
		fmt.Print("cached dir must be set")
		flag.Usage()
		os.Exit(1)
	}

	errCh := make(chan error)

	log.Printf("goproxy: %s inited. listen on %s\n", time.Now().Format("2006-01-02 15:04:05"), listen)

	r, err := router.NewRouter()
	if err != nil {
		log.Fatal(err)
	}

	legacy, err := vcs.NewFactory(cacheDir)
	if err != nil {
		log.Fatal(err)
	}
	if err := r.AddRoute("", legacy); err != nil {
		log.Fatal(err)
	}

	if len(gitlabHost) > 0 {
		gl := gitlab.NewFactory(gitlabHost, true)
		if err := r.AddRoute("gitlab", gl); err != nil {
			log.Fatal(err)
		}
	}

	m := goproxy.Middleware(r, "")

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
		log.Fatal(err)
	case sign := <-signCh:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		log.Printf("goproxy: Server gracefully %s", sign)
	}
}
