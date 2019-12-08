package versioning

import (
	"context"
	"net/http"
	"strings"
)

var (
	// contextKey is the context key of the version.
	contextKey interface{} = "api.version"
	// NotFound is the key that can be used inside a `Map` or inside `context.WithValue(r.Context(), versioning.contextKey, versioning.NotFound)`
	// to tell that a version wasn't found, therefore the not found handler should handle the request instead.
	NotFound = contextKey.(string) + ".notfound"
)

const (
	// AcceptVersionHeaderKey is the header key of "Accept-Version".
	AcceptVersionHeaderKey = "Accept-Version"
	// AcceptHeaderKey is the header key of "Accept".
	AcceptHeaderKey = "Accept"
	// AcceptHeaderVersionValue is the Accept's header value search term the requested version.
	AcceptHeaderVersionValue = "version"
)

var versionNotFoundText = []byte("version not found")

// NotFoundHandler is the default version not found handler that
// is executed from `NewMatcher` when no version is registered as available to dispatch a resource.
var NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 303 is an option too,
	// end-dev has the chance to change that behavior by using the NotFound in the map:
	//
	// https://www.w3.org/Protocols/rfc2616/rfc2616-sec10.html
	/*
		10.5.2 501 Not Implemented

		The server does not support the functionality required to fulfill the request.
		This is the appropriate response when the server does not
		recognize the request method and is not capable of supporting it for any resource.
	*/

	w.WriteHeader(http.StatusNotImplemented)
	w.Write(versionNotFoundText)
})

// GetVersion returns the current request version.
//
// By default the `GetVersion` will try to read from:
// - "Accept" header, i.e Accept: "application/json; version=1.0"
// - "Accept-Version" header, i.e Accept-Version: "1.0"
//
// However, the end developer can also set a custom version for a handler trough a middleware by using the request's context's value
// for versions (see `WithVersion` for further details on that).
func GetVersion(r *http.Request) string {
	// firstly by context store, if manually set-ed by a middleware.
	if v := r.Context().Value(contextKey); v != nil {
		if version, ok := v.(string); ok {
			return version
		}
	}

	// secondly by the "Accept-Version" header.
	if version := r.Header.Get(AcceptVersionHeaderKey); version != "" {
		return version
	}

	// thirdly by the "Accept" header which is like"...; version=1.0"
	acceptValue := r.Header.Get(AcceptHeaderKey)
	if acceptValue != "" {
		if idx := strings.Index(acceptValue, AcceptHeaderVersionValue); idx != -1 {
			rem := acceptValue[idx:]
			startVersion := strings.Index(rem, "=")
			if startVersion == -1 || len(rem) < startVersion+1 {
				return NotFound
			}

			rem = rem[startVersion+1:]

			end := strings.Index(rem, " ")
			if end == -1 {
				end = strings.Index(rem, ";")
			}
			if end == -1 {
				end = len(rem)
			}

			if version := rem[:end]; version != "" {
				return version
			}
		}
	}

	return NotFound
}

// WithVersion creates the new context that contains a passed version.
// Example of how you can change the default behavior to extract a requested version (which is by headers)
// from a "version" url parameter instead:
// func(w http.ResponseWriter, r *http.Request) { // &version=1
// 	r = r.WithContext(versioning.WithVersion(r.Context(), r.URL.Query().Get("version")))
// 	nextHandler.ServeHTTP(w,r)
// }
func WithVersion(ctx context.Context, version string) context.Context {
	return context.WithValue(ctx, contextKey, version)
}