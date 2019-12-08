package versioning_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kataras/versioning"
)

func TestGetVersion(t *testing.T) {
	router := http.NewServeMux()

	writeVesion := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(versioning.GetVersion(r)))
	})

	router.Handle("/", writeVesion)
	router.HandleFunc("/manual", func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(versioning.WithVersion(context.Background(), "11.0.5"))
		writeVesion.ServeHTTP(w, r)
	})

	srv := httptest.NewServer(router)
	defer srv.Close()

	expect(t, http.MethodGet, srv.URL, withHeader(versioning.AcceptVersionHeaderKey, "1.0")).
		statusCode(http.StatusOK).
		bodyEq("1.0")

	expect(t, http.MethodGet, srv.URL, withHeader(versioning.AcceptHeaderKey, "application/vnd.api+json; version=2.1")).
		statusCode(http.StatusOK).
		bodyEq("2.1")

	expect(t, http.MethodGet, srv.URL, withHeader(versioning.AcceptHeaderKey, "application/vnd.api+json; version=2.1 ;other=dsa")).
		statusCode(http.StatusOK).
		bodyEq("2.1")

	expect(t, http.MethodGet, srv.URL, withHeader(versioning.AcceptHeaderKey, "version=2.1")).
		statusCode(http.StatusOK).
		bodyEq("2.1")

	expect(t, http.MethodGet, srv.URL, withHeader(versioning.AcceptHeaderKey, "version=1")).
		statusCode(http.StatusOK).
		bodyEq("1")

		// unknown versions.
	expect(t, http.MethodGet, srv.URL, withHeader(versioning.AcceptVersionHeaderKey, "")).
		statusCode(http.StatusOK).
		bodyEq(versioning.NotFound)

	expect(t, http.MethodGet, srv.URL, withHeader(versioning.AcceptHeaderKey, "application/vnd.api+json; version=")).
		statusCode(http.StatusOK).
		bodyEq(versioning.NotFound)

	expect(t, http.MethodGet, srv.URL, withHeader(versioning.AcceptHeaderKey, "application/vnd.api+json; version= ;other=dsa")).
		statusCode(http.StatusOK).
		bodyEq(versioning.NotFound)

	expect(t, http.MethodGet, srv.URL, withHeader(versioning.AcceptHeaderKey, "version=")).
		statusCode(http.StatusOK).
		bodyEq(versioning.NotFound)

	expect(t, http.MethodGet, srv.URL+"/manual", withHeader(versioning.AcceptHeaderKey, "application/vnd.api+json; version= ;other=dsa")).
		statusCode(http.StatusOK).
		bodyEq("11.0.5")
}
