package versioning

import (
	"net/http"
	"time"
)

// HeaderTimeFormat is the time format that will be used to send DeprecationOptions's DeprectationDate time.
var HeaderTimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

// DeprecationOptions describes the deprecation headers key-values.
// - "X-API-Warn": options.WarnMessage
// - "X-API-Deprecation-Date": time.Now().Format("Mon, 02 Jan 2006 15:04:05 GMT")
// - "X-API-Deprecation-Info": options.DeprecationInfo
type DeprecationOptions struct {
	WarnMessage     string
	DeprecationDate time.Time
	DeprecationInfo string
}

// ShouldHandle reports whether the deprecation headers should be present or no.
func (opts DeprecationOptions) ShouldHandle() bool {
	return opts.WarnMessage != "" || !opts.DeprecationDate.IsZero() || opts.DeprecationInfo != ""
}

// DefaultDeprecationOptions are the default deprecation options,
// it defaults the "X-API-Warn" header to a generic message.
var DefaultDeprecationOptions = DeprecationOptions{
	WarnMessage: "WARNING! You are using a deprecated version of this API.",
}

// Deprecated marks a specific handler as a deprecated.
// Deprecated can be used to tell the clients that
// a newer version of that specific resource is available instead.
func Deprecated(handler http.Handler, options DeprecationOptions) http.Handler {
	if options.WarnMessage == "" {
		options.WarnMessage = DefaultDeprecationOptions.WarnMessage
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-API-Warn", options.WarnMessage)

		if !options.DeprecationDate.IsZero() {
			w.Header().Set("X-API-Deprecation-Date", options.DeprecationDate.Format(HeaderTimeFormat))
		}

		if options.DeprecationInfo != "" {
			w.Header().Set("X-API-Deprecation-Info", options.DeprecationInfo)
		}

		handler.ServeHTTP(w, r)
	})
}
