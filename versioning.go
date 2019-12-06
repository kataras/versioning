package versioning

import (
	"net/http"

	"github.com/hashicorp/go-version"
)

// If reports whether the "version" is a valid match to the "is".
// The "is" should be a version constraint like ">= 1, < 3".
func If(v string, is string) bool {
	ver, err := version.NewVersion(v)
	if err != nil {
		return false
	}

	constraints, err := version.NewConstraint(is)
	if err != nil {
		return false
	}

	return constraints.Check(ver)
}

// Match reports whether the current version matches the "expectedVersion".
func Match(r *http.Request, expectedVersion string) bool {
	return If(GetVersion(r), expectedVersion)
}

// Map is a map of version to handler.
// A handler per version or constraint, the key can be something like ">1, <=2" or just "1".
type Map map[string]http.Handler

// NewMatcher creates a single handler which decides what handler
// should be executed based on the requested version.
//
// Use the `NewGroup` if you want to add many routes under a specific version.
//
// See `Map` and `NewGroup` too.
func NewMatcher(versions Map) http.Handler {
	constraintsHandlers, notFoundHandler := buildConstraints(versions)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		versionString := GetVersion(r)
		if versionString == NotFound {
			notFoundHandler.ServeHTTP(w, r)
			return
		}

		ver, err := version.NewVersion(versionString)
		if err != nil {
			notFoundHandler.ServeHTTP(w, r)
			return
		}

		for _, ch := range constraintsHandlers {
			if ch.constraints.Check(ver) {
				w.Header().Set("X-API-Version", ver.String())
				ch.handler.ServeHTTP(w, r)
				return
			}
		}

		// pass the not matched version so the not found handler can have knowedge about it.
		// ctx.Values().Set(Key, versionString)
		// or let a manual cal of GetVersion(ctx) do that instead.
		notFoundHandler.ServeHTTP(w, r)
	})
}

type constraintsHandler struct {
	constraints version.Constraints
	handler     http.Handler
}

func buildConstraints(versionsHandler Map) (constraintsHandlers []*constraintsHandler, notfoundHandler http.Handler) {
	for v, h := range versionsHandler {
		if v == NotFound {
			notfoundHandler = h
			continue
		}

		constraints, err := version.NewConstraint(v)
		if err != nil {
			panic(err)
		}

		constraintsHandlers = append(constraintsHandlers, &constraintsHandler{
			constraints: constraints,
			handler:     h,
		})
	}

	if notfoundHandler == nil {
		notfoundHandler = NotFoundHandler
	}

	return
}
