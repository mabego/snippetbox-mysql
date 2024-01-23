package main

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-playground/form/v4"
	"github.com/justinas/nosurf"
)

var ErrNoTmpl = errors.New("template does not exist")

// serverError helper writes an error message and a stack trace to the errorLog,
// then sends a generic 500 Internal Server Error response to the user.
func (app *application) serverError(w http.ResponseWriter, err error) {
	trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
	app.errorLog.Output(2, trace)

	if app.debug {
		http.Error(w, trace, http.StatusInternalServerError)
		return
	}

	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

// clientError helper sends a specific status code and its description to the user.
func (app *application) clientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

// notFound helper is a wrapper around clientError that sends a 404 Not Found response to the user.
func (app *application) notFound(w http.ResponseWriter) {
	app.clientError(w, http.StatusNotFound)
}

func (app *application) render(w http.ResponseWriter, status int, page string, data *templateData) {
	// Use the page name as the map key to retrieve a template set from the cache
	ts, ok := app.templateCache[page]
	if !ok {
		err := fmt.Errorf("%w: %s", ErrNoTmpl, page)
		app.serverError(w, err)
		return
	}

	buf := new(bytes.Buffer)

	// Execute the template set and write the template to the buffer instead of the response body.
	err := ts.ExecuteTemplate(buf, "base", data)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.WriteHeader(status)

	// Write the contents of the buffer to http.ResponseWriter.
	_, err = buf.WriteTo(w)
	if err != nil {
		app.serverError(w, err)
		return
	}
}

func (app *application) newTemplateData(r *http.Request) *templateData {
	return &templateData{
		IsAuthenticated: app.isAuthenticated(r),
		IsAuthorized:    app.isAuthorized(r),
		CurrentYear:     time.Now().Year(),
		Flash:           app.sessionManager.PopString(r.Context(), "flash"),
		CSRFToken:       nosurf.Token(r),
	}
}

func (app *application) decodePostForm(r *http.Request, dst any) error {
	err := r.ParseForm()
	if err != nil {
		return err
	}

	err = app.formDecoder.Decode(dst, r.PostForm)
	if err != nil {
		// Check for a non-nil pointer through the error InvalidDecoderError
		var invalidDecoderError *form.InvalidDecoderError

		if errors.As(err, &invalidDecoderError) {
			panic(err)
		}

		return fmt.Errorf("form decoding error: %w", err)
	}

	return nil
}

func (app *application) isAuthenticated(r *http.Request) bool {
	isAuthenticated, ok := r.Context().Value(isAuthenticatedContextKey).(bool)
	if !ok {
		return false
	}

	return isAuthenticated
}

func (app *application) isAuthorized(r *http.Request) bool {
	isAuthorized, ok := r.Context().Value(isAuthorizedContextKey).(bool)
	if !ok {
		return false
	}

	return isAuthorized
}
