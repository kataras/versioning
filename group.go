package versioning

import "net/http"

// Group is a group of version-based routes.
// One version per one or more routes.
type Group struct {
	version string
	routes  map[string]Map // key = path, value = map[version] = handler

	deprecation DeprecationOptions
}

// NewGroup returns a ptr to Group based on the given "version".
//
// See `Handle` and `RegisterGroups` for more.
func NewGroup(version string) *Group {
	return &Group{
		version: version,
		routes:  make(map[string]Map),
	}
}

// Deprecated marks this group and all its versioned routes
// as deprecated versions of that endpoint.
// It can be called in the end just before `RegisterGroups`
// or first by `NewGroup(...).Deprecated(...)`. It returns itself.
func (g *Group) Deprecated(options DeprecationOptions) *Group {
	// if `Deprecated` is called in the end.
	for _, versions := range g.routes {
		versions[g.version] = Deprecated(versions[g.version], options)
	}

	// store the options if called before registering any versioned routes.
	g.deprecation = options

	return g
}

func (g *Group) addVRoute(path string, handler http.Handler) {
	if _, exists := g.routes[path]; !exists {
		g.routes[path] = Map{g.version: handler}
	}
}

// Handle registers a versioned route to the group.
// A call of `RegisterGroups` is necessary in order to register the actual routes
// when the group is complete.
//
// See `RegisterGroups` for more.
func (g *Group) Handle(path string, handler http.Handler) {
	if g.deprecation.ShouldHandle() { // if `Deprecated` called first.
		handler = Deprecated(handler, g.deprecation)
	}

	g.addVRoute(path, handler)
}

// HandleFunc registers a versioned route to the group.
// A call of `RegisterGroups` is necessary in order to register the actual routes
// when the group is complete.
//
// See `RegisterGroups` for more.
func (g *Group) HandleFunc(path string, handlerFn func(w http.ResponseWriter, r *http.Request)) {
	var handler http.Handler = http.HandlerFunc(handlerFn)

	if g.deprecation.ShouldHandle() { // if `Deprecated` called first.
		handler = Deprecated(handler, g.deprecation)
	}

	g.addVRoute(path, handler)
}

// StdMux is an interface which types like `net/http#ServeMux`
// implements in order to register handlers per path.
//
// See `RegisterGroups`.
type StdMux interface{ Handle(string, http.Handler) }

// RegisterGroups registers one or more groups to an `net/http#ServeMux` if not nil, and returns the routes.
// Map's key is the request path from `Group#Handle` and value is the `http.Handler`.
// See `NewGroup` and `NotFoundHandler` too.
func RegisterGroups(mux StdMux, notFoundHandler http.Handler, groups ...*Group) map[string]http.Handler {
	total := make(map[string]Map)
	routes := make(map[string]http.Handler)

	for _, g := range groups {
		for path, versions := range g.routes {
			if _, exists := total[path]; exists {
				total[path][g.version] = versions[g.version]
			} else {
				total[path] = versions
			}
		}
	}

	for path, versions := range total {
		if notFoundHandler != nil {
			versions[NotFound] = notFoundHandler
		}

		matcher := NewMatcher(versions)
		if mux != nil {
			mux.Handle(path, matcher)
		}

		routes[path] = matcher
	}

	return routes
}
