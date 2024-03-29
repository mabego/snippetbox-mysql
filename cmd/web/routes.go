package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
	"github.com/mabego/snippetbox-mysql/ui"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	// Set the custom handler for 404 responses through httprouter.
	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.notFound(w)
	})

	// Use an embedded file system instead of reading files from the disk at runtime.
	fileServer := http.FileServer(http.FS(ui.Files))
	router.Handler(http.MethodGet, "/static/*filepath", fileServer)

	// A ping route for testing.
	router.HandlerFunc(http.MethodGet, "/ping", ping)

	// An unprotected middleware chain using alice, specific to 'dynamic' application routes.
	dynamic := alice.New(app.sessionManager.LoadAndSave, noSurf, app.authenticate, app.authorize)

	// 'dynamic' middleware chain routes
	router.Handler(http.MethodGet, "/user/login", dynamic.ThenFunc(app.userLogin))
	router.Handler(http.MethodPost, "/user/login", dynamic.ThenFunc(app.userLoginPost))
	router.Handler(http.MethodGet, "/user/signup", dynamic.ThenFunc(app.userSignup))
	router.Handler(http.MethodPost, "/user/signup", dynamic.ThenFunc(app.userSignupPost))

	// A protected (authenticated-only) and dynamic middleware chain.
	protected := dynamic.Append(app.requireAuthentication)

	// 'protected' middleware chain routes
	router.Handler(http.MethodGet, "/", protected.ThenFunc(app.home))
	router.Handler(http.MethodGet, "/about", protected.ThenFunc(app.aboutView))
	router.Handler(http.MethodGet, "/snippet/view/:id", protected.ThenFunc(app.snippetView))
	router.Handler(http.MethodPost, "/snippet/view/:id", protected.ThenFunc(app.reviewUpdatePost))
	router.Handler(http.MethodPost, "/user/logout", protected.ThenFunc(app.userLogoutPost))
	router.Handler(http.MethodGet, "/account/view", protected.ThenFunc(app.accountView))
	router.Handler(http.MethodGet, "/account/password/update", protected.ThenFunc(app.accountPasswordUpdate))
	router.Handler(http.MethodPost, "/account/password/update", protected.ThenFunc(app.accountPasswordUpdatePost))

	// An authorized-only and dynamic middleware chain.
	owner := dynamic.Append(app.requireAuthorization)

	// 'owner' middleware chain routes
	router.Handler(http.MethodGet, "/snippet/create", owner.ThenFunc(app.snippetCreate))
	router.Handler(http.MethodPost, "/snippet/create", owner.ThenFunc(app.snippetCreatePost))

	// A middleware chain using alice containing the 'standard' middleware used for every application request.
	standard := alice.New(app.recoverPanic, app.logRequest, secureHeaders)

	// Return the 'standard' middleware chain followed by the ServeMux.
	return standard.Then(router)
}
