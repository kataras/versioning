# API Versioning (Go)

[![build status](https://img.shields.io/travis/kataras/versioning/master.svg?style=for-the-badge&logo=travis)](https://travis-ci.org/kataras/versioning) [![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/kataras/versioning) [![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://godoc.org/github.com/kataras/versioning) [![donate on PayPal](https://img.shields.io/badge/support-PayPal-blue.svg?style=for-the-badge)](https://www.paypal.me/kataras)

[Semver](https://semver.org/) versioning for your APIs. It implements all the suggestions written at [api-guidelines](https://github.com/byrondover/api-guidelines/blob/master/Guidelines.md#versioning) and more.

The version comparison is done by the [go-version](https://github.com/hashicorp/go-version) package. It supports matching over patterns like `">= 1.0, < 3"` and e.t.c.

## Getting started

The only requirement is the [Go Programming Language](https://golang.org/dl).

```sh
$ go get github.com/kataras/versioning
```

## Features

- Per route version matching, an `http.Handler` with "switch" cases via [versioning.Map](https://github.com/kataras/versioning/blob/master/versioning.go#L33) for version => handler
- Per group versioned routes and deprecation API
- Version matching like ">= 1.0, < 2.0" or just "2.0.1" and e.t.c.
- Version not found handler (can be customized by simply adding the `versioning.NotFound`: customNotMatchVersionHandler on the Map)
- Version is retrieved from the "Accept" and "Accept-Version" headers (can be customized through request's context key)
- Respond with "X-API-Version" header, if version found.
- Deprecation options with customizable "X-API-Warn", "X-API-Deprecation-Date", "X-API-Deprecation-Info" headers via `Deprecated` wrapper.

## Compare Versions

```go
// If reports whether the "version" is a valid match to the "is".
// The "is" can be a version constraint like ">= 1, < 3".
If(version string, is string) bool
```

```go
// Match reports whether the current version matches the "expectedVersion".
Match(r *http.Request, expectedVersion string) bool
```

Example

```go
router.HandleFunc("/api/user", func(w http.ResponseWriter, r *http.Request) {
    if versioning.Match(r, ">= 2.2.3") {
        // [logic for >= 2.2.3 version of your handler goes here]
        return
    }
})
```

## Determining The Current Version

Current request version is retrieved by `versioning.GetVersion(r *http.Request)`.

By default the `GetVersion` will try to read from:
- `Accept` header, i.e `Accept: "application/json; version=1.0"`
- `Accept-Version` header, i.e `Accept-Version: "1.0"`

```go
func handler(w http.ResponseWriter, r *http.Request){
    currentVersion := versioning.GetVersion(r)
}
```

You can also **set a custom version** to a handler trough a middleware by setting a request context's value.
For example:
```go
import (
    "context"
    "net/http"

    "github.com/kataras/versioning"
)

func urlParamVersion(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
        version := r.URL.Query().Get("v") // ?v=2.3.5
        if version == "" {
            // set a default version, e.g. 1.0
            version = "1.0"
        }
        r = r.WithContext(context.WithValue(r.Context(), versioning.ContextKey, version))
        next.ServeHTTP(w, r)
    })
}
```

## Map Versions to Handlers

The `versioning.NewMatcher(versioning.Map) http.Handler` creates a single handler which decides what handler need to be executed based on the requested version.

```go
// middleware for all versions.
func myMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
        // [...]
        next.ServeHTTP(w, r)
    })
}

func myCustomVersionNotFound(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(404)
    fmt.Fprintf(w, "%s version not found", versioning.GetVersion(r))
}

router := http.NewServeMux()
router.Handle("/", myMiddleware, versioning.NewMatcher(versioning.Map{
    // v1Handler is a handler of yuors that will be executed only on version 1.
    "1.0":               v1Handler, 
    ">= 2, < 3":         v2Handler,
    versioning.NotFound: http.HandlerFunc(myCustomNotVersionFound),
}))
```

### Deprecation

Using the `versioning.Deprecated(handler http.Handler, options versioning.DeprecationOptions) http.Handler` function you can mark a specific handler version as deprecated.

```go
v1Handler = versioning.Deprecated(v1Handler, versioning.DeprecationOptions{
    // if empty defaults to: "WARNING! You are using a deprecated version of this API."
    WarnMessage string
    DeprecationDate time.Time
    DeprecationInfo string
})

router.Handle("/", versioning.NewMatcher(versioning.Map{
    "1.0": v1Handler,
    // [...]
}))
```

This will make the handler to send these headers to the client:

- `"X-API-Warn": options.WarnMessage`
- `"X-API-Deprecation-Date": options.DeprecationDate`
- `"X-API-Deprecation-Info": options.DeprecationInfo`

> versioning.DefaultDeprecationOptions can be passed instead if you don't care about Date and Info.

## Grouping Routes By Version

Grouping routes by version is possible as well.

Using the `versioning.NewGroup(version string) *versioning.Group` function you can create a group to register your versioned routes.
The `versioning.RegisterGroups(r *http.ServeMux, versionNotFoundHandler http.Handler, groups ...*versioning.Group)` must be called in the end in order to register the routes to a specific `StdMux`.

```go
router := http.NewServeMux()

// version 1.
usersAPIV1 := versioning.NewGroup(">= 1, < 2")
usersAPIV1.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
        return
    }

    w.Write([]byte("v1 resource: /api/users handler"))
})
usersAPIV1.HandleFunc("/api/users/new", func(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
        return
    }

    w.Write([]byte("v1 resource: /api/users/new post handler"))
})

// version 2.
usersAPIV2 := versioning.NewGroup(">= 2, < 3")
usersAPIV2.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
        return
    }

    w.Write([]byte("v2 resource: /api/users handler"))
})
usersAPIV2.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
        return
    }

    w.Write([]byte("v2 resource: /api/users post handler"))
})

versioning.RegisterGroups(router, versioning.NotFoundHandler, usersAPIV1, usersAPIV2)
```

> A middleware can be registered, using the methods we learnt above, i.e by using the `versioning.Match` in order to detect what code/handler you want to be executed when "x" or no version is requested.

### Deprecation for Group

Just call the `Group#Deprecated(versioning.DeprecationOptions)` on the group you want to notify your API consumers that this specific version is deprecated.

```go
userAPIV1 := versioning.NewGroup("1.0").Deprecated(versioning.DefaultDeprecationOptions)
```

For a more detailed technical documentation you can head over to our [godocs](https://godoc.org/github.com/kataras/versioning). And for executable code you can always visit the [_examples](_examples) repository's subdirectory.

## License

kataras/versioning is free and open-source software licensed under the [MIT License](https://tldrlegal.com/license/mit-license).
