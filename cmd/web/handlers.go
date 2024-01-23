package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
	"github.com/mabego/snippetbox-mysql/internal/models"
	"github.com/mabego/snippetbox-mysql/internal/validator"
)

const (
	Day           = 1
	MinChars      = 8
	TitleMaxChars = 100
	Week          = 7
	Year          = 365
)

// The struct tags tell the go-playground/form decoder how to map HTML form values into the different struct fields.
// Any type conversions are handled automatically.
// The struct tag `form:"-"` tells the decoder to completely ignore a field during decoding.
type snippetCreateForm struct {
	Title               string `form:"title"`
	Content             string `form:"content"`
	Expires             int    `form:"expires"`
	validator.Validator `form:"-"`
}

type userSignupForm struct {
	Name                string `form:"name"`
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

type userLoginForm struct {
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

type accountPasswordUpdateForm struct {
	CurrentPassword         string `form:"currentPassword"`
	NewPassword             string `form:"newPassword"`
	NewPasswordConfirmation string `form:"newPasswordConfirmation"`
	validator.Validator     `form:"-"`
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	snippets, err := app.snippets.Latest()
	if err != nil {
		app.serverError(w, err)
		return
	}

	data := app.newTemplateData(r)
	data.Snippets = snippets

	app.render(w, http.StatusOK, "home.page.tmpl", data)
}

func (app *application) snippetView(w http.ResponseWriter, r *http.Request) {
	// ParamsFromContext returns a slice containing parameter names and values from the request context.
	params := httprouter.ParamsFromContext(r.Context())

	// ByName gets the value of the "id" named parameter from the slice and validates it as normal.
	id, err := strconv.Atoi(params.ByName("id"))
	if err != nil || id < 1 {
		app.notFound(w)
		return
	}

	snippet, err := app.snippets.Get(id)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			app.notFound(w)
		} else {
			app.serverError(w, err)
		}
		return
	}

	data := app.newTemplateData(r)
	data.Snippet = snippet

	// Retrieve the authenticatedUserID value from the session
	// GetInt will return 0 if no authenticatedUserID value is in the session.
	userID := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")

	review, err := app.reviews.Get(userID, snippet.ID)
	if err != nil {
		app.serverError(w, err)
		return
	}

	data.Review = review

	app.render(w, http.StatusOK, "view.page.tmpl", data)
}

func (app *application) snippetCreate(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)

	// Initialize a new snippetCreateForm instance and pass it to the template
	// so that the templateData.Form field is not nil and set any default values for the form.
	data.Form = snippetCreateForm{Expires: Year}

	app.render(w, http.StatusOK, "create.page.tmpl", data)
}

func (app *application) snippetCreatePost(w http.ResponseWriter, r *http.Request) {
	var form snippetCreateForm

	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Title), "title", "This field cannot be blank")
	form.CheckField(validator.MaxChars(form.Title, TitleMaxChars), "title",
		"This field cannot be more than 100 characters long")
	form.CheckField(validator.NotBlank(form.Content), "content", "This field cannot be blank")
	form.CheckField(validator.PermittedValue(form.Expires, Day, Week, Year), "expires",
		"This field must equal 1, 7 or 365")

	// If there are validation errors, redisplay the creation form along with a 422 status code.
	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "create.page.tmpl", data)
		return
	}

	id, err := app.snippets.Insert(form.Title, form.Content, form.Expires)
	if err != nil {
		app.serverError(w, err)
		return
	}

	// Put adds a key and string value to the session data.
	app.sessionManager.Put(r.Context(), "flash", "Snippet successfully created!")

	// Redirect the user to the newly created snippet page.
	http.Redirect(w, r, fmt.Sprintf("/snippet/view/%d", id), http.StatusSeeOther)
}

func (app *application) userSignup(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userSignupForm{}
	app.render(w, http.StatusOK, "signup.page.tmpl", data)
}

func (app *application) userSignupPost(w http.ResponseWriter, r *http.Request) {
	var form userSignupForm

	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Name), "name", "This field cannot be blank")
	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email",
		"This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")
	form.CheckField(validator.MinChars(form.Password, MinChars), "password",
		"This field must be at least 8 characters long")

	// If there are validation errors, redisplay the signup form along with a 422 status code.
	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "signup.page.tmpl", data)
		return
	}

	err = app.users.Insert(form.Name, form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrDuplicateEmail) {
			form.AddFieldError("email", "Email address is already in use")
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, http.StatusUnprocessableEntity, "signup.page.tmpl", data)
		} else {
			app.serverError(w, err)
		}

		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Your signup was successful. Please log in")

	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

func (app *application) userLogin(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userLoginForm{}
	app.render(w, http.StatusOK, "login.page.tmpl", data)
}

func (app *application) userLoginPost(w http.ResponseWriter, r *http.Request) {
	var form userLoginForm

	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email",
		"This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "login.page.tmpl", data)
		return
	}

	// If the credentials are invalid, add a generic non-field error and redisplay the login form.
	id, err := app.users.Authenticate(form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			form.AddNonFieldError("Email or password is incorrect")
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, http.StatusUnprocessableEntity, "login.page.tmpl", data)
		} else {
			app.serverError(w, err)
		}
		return
	}

	// RenewToken changes the current session ID when the authentication state changes for the user
	// with the login operation.
	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}

	// Add the ID of the current user to the session, so that they are now logged in.
	app.sessionManager.Put(r.Context(), "authenticatedUserID", id)

	// PopString pops the value for the "redirectPathAfterLogin" key from the session data.
	// If there is no matching key in the session data, it will return an empty string.
	urlPath := app.sessionManager.PopString(r.Context(), "redirectPathAfterLogin")
	if urlPath != "" {
		http.Redirect(w, r, urlPath, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/snippet/create", http.StatusSeeOther)
}

func (app *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {
	// RenewToken changes the current session ID when the authentication state changes for the user
	// with the logout operation.
	err := app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}

	// Remove authenticatedUserID from the session data so the user is logged out.
	app.sessionManager.Remove(r.Context(), "authenticatedUserID")

	app.sessionManager.Put(r.Context(), "flash", "You've been logged out successfully!")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) aboutView(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	app.render(w, http.StatusOK, "about.page.tmpl", data)
}

func (app *application) accountView(w http.ResponseWriter, r *http.Request) {
	// GetInt will return 0 if no authenticatedUserID value is in the session.
	userID := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")

	user, err := app.users.Get(userID)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			http.Redirect(w, r, "/user/login", http.StatusSeeOther)
		} else {
			app.serverError(w, err)
		}
		return
	}

	data := app.newTemplateData(r)
	data.User = user
	app.render(w, http.StatusOK, "account.page.tmpl", data)
}

func (app *application) accountPasswordUpdate(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = accountPasswordUpdateForm{}

	app.render(w, http.StatusOK, "password.page.tmpl", data)
}

func (app *application) accountPasswordUpdatePost(w http.ResponseWriter, r *http.Request) {
	var form accountPasswordUpdateForm

	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.CurrentPassword), "currentPassword",
		"This field cannot be blank")
	form.CheckField(validator.NotBlank(form.NewPassword), "newPassword",
		"This field cannot be blank")
	form.CheckField(validator.MinChars(form.NewPassword, MinChars), "newPassword",
		"This field must be at least 8 characters long")
	form.CheckField(validator.NotBlank(form.NewPasswordConfirmation), "newPasswordConfirmation",
		"This field cannot be blank")
	form.CheckField(form.NewPassword == form.NewPasswordConfirmation, "newPasswordConfirmation",
		"Passwords do not match")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form

		app.render(w, http.StatusUnprocessableEntity, "password.page.tmpl", data)
		return
	}

	userID := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")

	err = app.users.PasswordUpdate(userID, form.CurrentPassword, form.NewPassword)
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			form.AddFieldError("currentPassword", "Current password is incorrect")

			data := app.newTemplateData(r)
			data.Form = form

			app.render(w, http.StatusUnprocessableEntity, "password.page.tmpl", data)
		} else {
			app.serverError(w, err)
		}
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Your password has been updated!")

	http.Redirect(w, r, "/account/view", http.StatusSeeOther)
}

func (app *application) reviewUpdatePost(w http.ResponseWriter, r *http.Request) {
	// ParamsFromContext returns a slice containing parameter names and values from the request context.
	params := httprouter.ParamsFromContext(r.Context())

	// ByName gets the value of the "id" named parameter from the slice and validates it as normal.
	snippetID, err := strconv.Atoi(params.ByName("id"))
	if err != nil || snippetID < 1 {
		app.notFound(w)
		return
	}

	// Retrieve the authenticatedUserID value from the session
	// GetInt will return 0 if no authenticatedUserID value is in the session.
	userID := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")

	err = app.reviews.Update(userID, snippetID)
	if err != nil {
		app.serverError(w, err)
		return
	}

	// Put adds a key and string value to the session data.
	app.sessionManager.Put(r.Context(), "flash", "Review successfully submitted!")

	// Redirect the user to the relevant page for the section.
	http.Redirect(w, r, fmt.Sprintf("/snippet/view/%d", snippetID), http.StatusSeeOther)
}

func ping(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	if r.Method == http.MethodGet {
		fmt.Fprintln(w, "OK")
	}
}
