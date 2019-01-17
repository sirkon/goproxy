# GoProxy middleware

Middleware for go modules proxy, inspired by https://github.com/goproxyio/goproxy

## Usage
How to set up a middleware:

1. Initiate router:
    ```go
    r, err := router.NewRouter()
    ```
2. Add needed handlers via
    ```go
    err := r.AddRoute(proxyFactory)
    ```
    for example:
    ```go
    regular, err := vcs.NewFactory(cacheDir)
    if err != nil {
        log.Fatal(err)
    }
    if err := r.AddRoute("", regular); err != nil {
        log.Fatal(err)
    }
    ```
    this library currently supports `vcs` which is pretty much like regular `go get` (`regular` in the example), `gitlab` which works
    upon gitlab's v4 API and delegation to another go proxy, see `source/...`
3. Generate middleware:
    ```go
    var m http.Handler = goproxy.Middleware(r)
    ```
    Now you can  use it in your HTTP server
    See `examples/goproxy` for details



## Example
There's an example which supports regular `go get`-like module retrieval and gitlab.

1. Build it:
    ```bash
    go build ./examples/goproxy
    ```
2. Start it:
    ```bash
    ./goproxy -cache-dir . -gitlab https://gitlab.com/api/v4 -listen 0.0.0.0:8081
    ```
3. Set up environment:
    ```bash
    export GO111MODULE=on
    export GOPROXY=http://<gitlab private token>@localhost:8081
    ```
    Remember, you must generate gitlab private token first otherwise gitlab may reject your requests
