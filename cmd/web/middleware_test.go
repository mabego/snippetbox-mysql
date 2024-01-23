package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mabego/snippetbox-mysql/internal/assert"
)

func TestSecureHeaders(t *testing.T) {
	// Initialize a new httptest.ResponseRecorder and an http.Request.
	rr := httptest.NewRecorder()

	r, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a mock HTTP handler to pass to the secureHeaders middleware that writes status code 200
	// and an "OK" response body.
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Pass the mock HTTP handler to the secureHeaders middleware and call its ServeHTTP method with
	// the httptest.ResponseRecorder and the http.Request.
	secureHeaders(next).ServeHTTP(rr, r)

	rs := rr.Result()

	// Run the expected headers tests.

	expectedCSP := "default-src 'self'; style-src 'self' fonts.googleapis.com; font-src fonts.gstatic.com"
	assert.Equal(t, rs.Header.Get("Content-Security-Policy"), expectedCSP)

	expectedRP := "origin-when-cross-origin"
	assert.Equal(t, rs.Header.Get("Referrer-Policy"), expectedRP)

	expectedXCTO := "nosniff"
	assert.Equal(t, rs.Header.Get("X-Content-Type-Options"), expectedXCTO)

	expectedXFO := "deny"
	assert.Equal(t, rs.Header.Get("X-Frame-Options"), expectedXFO)

	expectedXXP := "0"
	assert.Equal(t, rs.Header.Get("X-XSS-Protection"), expectedXXP)

	// Run the handler chain tests

	assert.Equal(t, rs.StatusCode, http.StatusOK)

	defer rs.Body.Close()
	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}
	bytes.TrimSpace(body)

	assert.Equal(t, string(body), "OK")
}
