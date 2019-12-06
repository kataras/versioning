package versioning_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kataras/versioning"
)

func TestDeprecated(t *testing.T) {
	router := http.NewServeMux()

	writeVesion := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(versioning.GetVersion(r)))
	})

	opts := versioning.DeprecationOptions{
		WarnMessage:     "deprecated, see <this link>",
		DeprecationDate: time.Now().UTC(),
		DeprecationInfo: "a bigger version is available, see <this link> for more information",
	}
	router.Handle("/", versioning.Deprecated(writeVesion, opts))

	srv := httptest.NewServer(router)
	defer srv.Close()

	expectedDeprecationDate := opts.DeprecationDate.Format(versioning.HeaderTimeFormat)
	expect(t, http.MethodGet, srv.URL, withHeader(versioning.AcceptVersionHeaderKey, "1.0")).
		statusCode(http.StatusOK).
		headerEq("X-API-Warn", opts.WarnMessage).
		headerEq("X-API-Deprecation-Date", expectedDeprecationDate).
		bodyEq("1.0")
}
