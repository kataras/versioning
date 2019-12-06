package main

import (
	"net/http"

	"github.com/kataras/versioning"
)

func main() {
	router := http.NewServeMux()

	examplePerRoute(router)
	exampleRegisterGroups(router)

	println("Listening on: http://localhost:8080")
	http.ListenAndServe(":8080", router)
}

// How to test:
// Open Postman
// GET: localhost:8080/api/cats
// Headers[1] = Accept-Version: "1" and repeat with
// Headers[1] = Accept-Version: "2.5"
// or even "Accept": "application/json; version=2.5"
func examplePerRoute(router *http.ServeMux) {
	router.Handle("/api/cats", versioning.NewMatcher(versioning.Map{
		"1":                 catsVersionExactly1Handler,
		">= 2, < 3":         catsV2Handler,
		versioning.NotFound: versioning.NotFoundHandler,
	}))
}

// How to test:
// Open Postman
// GET: localhost:8080/api/users
// Headers[1] = Accept-Version: "1.9.9" and repeat with
// Headers[1] = Accept-Version: "2.5"
//
// POST: localhost:8080/api/users/new
// Headers[1] = Accept-Version: "1.8.3"
//
// POST: localhost:8080/api/users
// Headers[1] = Accept-Version: "2"
func exampleRegisterGroups(router *http.ServeMux) {
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
}

var catsVersionExactly1Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("v1 exactly resource: /api/cats handler"))
})

var catsV2Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("v2 resource: /api/cats handler"))
})
