package main

import (
	"bytes"
	"html"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"
	"github.com/mabego/snippetbox-mysql/internal/models/mocks"
)

// csrfTokenRX captures the CSRF token value from the user signup page.
var csrfTokenRX = regexp.MustCompile(`<input type="hidden" name="csrf_token" value="(.+)">`)

func extractCSRFToken(t *testing.T, body string) string {
	t.Helper()

	// Extract the token from the HTML body.
	// FindStringSubmatch returns an array with the entire matched pattern at index 0,
	// and the values of any captured data in the subsequent indices.
	matches := csrfTokenRX.FindStringSubmatch(body)
	if len(matches) < 2 {
		t.Fatal("no csrf token found in body")
	}

	return html.UnescapeString(matches[1])
}

// newTestApplication creates an instance of the application struct with mock data.
func newTestApplication(t *testing.T) *application {
	t.Helper()

	templateCache, err := newTemplateCache()
	if err != nil {
		t.Fatal(err)
	}

	formDecoder := form.NewDecoder()

	sessionManager := scs.New()
	sessionManager.Lifetime = SessionLifetime
	sessionManager.Cookie.Secure = true

	return &application{
		errorLog:       log.New(io.Discard, "", 0),
		infoLog:        log.New(io.Discard, "", 0),
		snippets:       &mocks.SnippetModel{},
		users:          &mocks.UserModel{},
		reviews:        &mocks.ReviewModel{},
		templateCache:  templateCache,
		formDecoder:    formDecoder,
		sessionManager: sessionManager,
	}
}

// A custom testServer type that embeds an httptest.Server instance.
type testServer struct {
	*httptest.Server
}

func newTestServer(t *testing.T, h http.Handler) *testServer {
	t.Helper()

	ts := httptest.NewTLSServer(h)

	// Initialize a new cookie jar.
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}

	// Add the cookie jar to the test server client.
	// Any response cookies will now be stored and sent with test server client requests.
	ts.Client().Jar = jar

	// Disable redirect-following for the test server client by setting a custom CheckRedirect function that uses an
	// http.ErrUseLastResponse error to force the client to return the received response.
	ts.Client().CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	return &testServer{ts}
}

// ts.get makes a GET request to a given url path using the test server client and returns the response
// status code, headers, and body.
func (ts *testServer) get(t *testing.T, urlPath string) (int, http.Header, string) {
	t.Helper()
	// The network address that the test server is listening to is contained in the ts.URL field.
	rs, err := ts.Client().Get(ts.URL + urlPath)
	if err != nil {
		t.Fatal(err)
	}

	defer rs.Body.Close()
	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}
	bytes.TrimSpace(body)

	return rs.StatusCode, rs.Header, string(body)
}

// postForm sends POST requests to the test server.
// The "form" parameter is a url.Values object that can contain any form data to send to the request body.
func (ts *testServer) postForm(t *testing.T, urlPath string, form url.Values) (int, http.Header, string) {
	t.Helper()

	rs, err := ts.Client().PostForm(ts.URL+urlPath, form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}
	bytes.TrimSpace(body)

	return rs.StatusCode, rs.Header, string(body)
}
