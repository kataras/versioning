package versioning_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/kataras/versioning"
)

var notFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
})

const (
	v10Response = "v1.0 handler"
	v2Response  = "v2.x handler"
)

func sendHandler(contents string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(contents))
	}
}

func TestIf(t *testing.T) {
	if expected, got := true, versioning.If("1.0", ">=1"); expected != got {
		t.Fatalf("expected %s to be %s", "1.0", ">= 1")
	}
	if expected, got := true, versioning.If("1.2.3", "> 1.2"); expected != got {
		t.Fatalf("expected %s to be %s", "1.2.3", "> 1.2")
	}
}

func TestNewMatcher(t *testing.T) {
	router := http.NewServeMux()
	router.Handle("/api/user", versioning.NewMatcher(versioning.Map{
		"1.0":               sendHandler(v10Response),
		">= 2, < 3":         sendHandler(v2Response),
		versioning.NotFound: notFoundHandler,
	}))

	// middleware as usual.
	myMiddleware := func(next http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Custom", "something")
			next.ServeHTTP(w, r)
		}
	}
	myVersions := versioning.Map{
		"1.0": sendHandler(v10Response),
	}

	router.Handle("/api/user/with_middleware", myMiddleware(versioning.NewMatcher(myVersions)))

	srv := httptest.NewServer(router)
	defer srv.Close()

	expect(t, http.MethodGet, srv.URL+"/api/user", withHeader(versioning.AcceptVersionHeaderKey, "1")).
		statusCode(http.StatusOK).
		bodyEq(v10Response)
	expect(t, http.MethodGet, srv.URL+"/api/user", withHeader(versioning.AcceptVersionHeaderKey, "2.0")).
		statusCode(http.StatusOK).
		bodyEq(v2Response)
	expect(t, http.MethodGet, srv.URL+"/api/user", withHeader(versioning.AcceptVersionHeaderKey, "2.1")).
		statusCode(http.StatusOK).
		bodyEq(v2Response)
	expect(t, http.MethodGet, srv.URL+"/api/user", withHeader(versioning.AcceptVersionHeaderKey, "2.9.9")).
		statusCode(http.StatusOK).
		bodyEq(v2Response)

	// middleware as usual.
	expect(t, http.MethodGet, srv.URL+"/api/user/with_middleware", withHeader(versioning.AcceptVersionHeaderKey, "1.0")).
		statusCode(http.StatusOK).
		bodyEq(v10Response).headerEq("X-Custom", "something")
	expect(t, http.MethodGet, srv.URL+"/api/user", withHeader(versioning.AcceptVersionHeaderKey, "3.0")).
		statusCode(http.StatusNotFound).
		bodyEq("Not Found\n")
}

func TestNewGroup(t *testing.T) {
	router := http.NewServeMux()

	userAPIV1 := versioning.NewGroup("1.0").Deprecated(versioning.DefaultDeprecationOptions)
	userAPIV1.Handle("/", sendHandler(v10Response))

	userAPIV2 := versioning.NewGroup(">= 2, < 3")
	userAPIV2.Handle("/", sendHandler(v2Response))
	userAPIV2.Handle("/other", sendHandler(v2Response))

	versioning.RegisterGroups(router, versioning.NotFoundHandler, userAPIV1, userAPIV2)

	srv := httptest.NewServer(router)
	defer srv.Close()

	expect(t, http.MethodGet, srv.URL, withHeader(versioning.AcceptVersionHeaderKey, "1")).
		statusCode(http.StatusOK).
		bodyEq(v10Response).
		headerEq("X-API-Warn", versioning.DefaultDeprecationOptions.WarnMessage)

	expect(t, http.MethodGet, srv.URL, withHeader(versioning.AcceptVersionHeaderKey, "2.1")).
		statusCode(http.StatusOK).
		bodyEq(v2Response)

	expect(t, http.MethodGet, srv.URL, withHeader(versioning.AcceptVersionHeaderKey, "2.9.9")).
		statusCode(http.StatusOK).
		bodyEq(v2Response)

	expect(t, http.MethodGet, srv.URL+"/other", withHeader(versioning.AcceptVersionHeaderKey, "2.9")).
		statusCode(http.StatusOK).
		bodyEq(v2Response)

	expect(t, http.MethodGet, srv.URL, withHeader(versioning.AcceptVersionHeaderKey, "3.0")).
		statusCode(http.StatusNotImplemented).
		bodyEq("version not found")
}

// Small test suite for this package follows.

func expect(t *testing.T, method, url string, testieOptions ...func(*http.Request)) *testie {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, opt := range testieOptions {
		opt(req)
	}

	return testReq(t, req)
}

func withHeader(key string, value string) func(*http.Request) {
	return func(r *http.Request) {
		r.Header.Add(key, value)
	}
}

func withQuery(key string, value string) func(*http.Request) {
	return func(r *http.Request) {
		q := r.URL.Query()
		q.Add(key, value)

		enc := strings.NewReader(q.Encode())
		r.Body = ioutil.NopCloser(enc)
		r.GetBody = func() (io.ReadCloser, error) { return http.NoBody, nil }

		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
}

func withFormField(key string, value string) func(*http.Request) {
	return func(r *http.Request) {
		if r.Form == nil {
			r.Form = make(url.Values)
		}
		r.Form.Add(key, value)

		enc := strings.NewReader(r.Form.Encode())
		r.Body = ioutil.NopCloser(enc)
		r.ContentLength = int64(enc.Len())

		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
}

func expectWithBody(t *testing.T, method, url string, body string, headers http.Header) *testie {
	req, err := http.NewRequest(method, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}

	if len(headers) > 0 {
		req.Header = http.Header{}
		for k, v := range headers {
			req.Header[k] = v
		}
	}

	return testReq(t, req)
}

func testReq(t *testing.T, req *http.Request) *testie {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	resp.Request = req
	return &testie{t: t, resp: resp}
}

func testHandler(t *testing.T, handler http.Handler, method, url string) *testie {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, url, nil)
	handler.ServeHTTP(w, req)
	resp := w.Result()
	resp.Request = req
	return &testie{t: t, resp: resp}
}

type testie struct {
	t    *testing.T
	resp *http.Response
}

func (te *testie) statusCode(expected int) *testie {
	if got := te.resp.StatusCode; expected != got {
		te.t.Fatalf("%s: expected status code: %d but got %d", te.resp.Request.URL, expected, got)
	}

	return te
}

func (te *testie) bodyEq(expected string) *testie {
	te.t.Helper()

	b, err := ioutil.ReadAll(te.resp.Body)
	te.resp.Body.Close()
	if err != nil {
		te.t.Fatal(err)
	}

	if got := string(b); expected != got {
		te.t.Fatalf("%s: expected to receive '%s' but got '%s'", te.resp.Request.URL, expected, got)
	}

	return te
}

func (te *testie) headerEq(key, expected string) *testie {
	if got := te.resp.Header.Get(key); expected != got {
		te.t.Fatalf("%s: expected header value of %s to be: '%s' but got '%s'", te.resp.Request.URL, key, expected, got)
	}

	return te
}
