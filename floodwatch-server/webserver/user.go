package webserver

import (
	"net/http"
	"strings"
	"time"

	"github.com/O-C-R/auth/id"

	"github.com/O-C-R/floodwatch-server/floodwatch-server/backend"
	"github.com/O-C-R/floodwatch-server/floodwatch-server/data"
)

func Logout(options *Options) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		cookie, err := req.Cookie(CookieName)
		if err == http.ErrNoCookie {
			w.WriteHeader(204)
			return
		}

		if err != nil {
			Error(w, err, 500)
			return
		}

		// Get the session ID from the cookie.
		sessionIDString := cookie.Value

		// Delete the cookie.
		cookie.Value = ""
		cookie.MaxAge = -1
		http.SetCookie(w, cookie)

		sessionID := id.ID{}
		if err := sessionID.UnmarshalText([]byte(sessionIDString)); err != nil {
			Error(w, err, 400)
			return
		}

		if err := options.SessionStore.DeleteSession(sessionID); err != nil {
			Error(w, err, 500)
			return
		}

		w.WriteHeader(204)
	})
}

func Login(options *Options) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		person, err := options.Backend.UserByUsername(req.FormValue("username"))
		if err == backend.NotFoundError {
			Error(w, err, 401)
			return
		}

		if err != nil {
			Error(w, err, 500)
			return
		}

		if err := person.CheckPassword(req.FormValue("password")); err != nil {
			Error(w, err, 401)
			return
		}

		person.LastSeen = time.Now()
		if err := options.Backend.UpsertPerson(person); err != nil {
			Error(w, err, 500)
			return
		}

		sessionID, err := id.New()
		if err != nil {
			Error(w, err, 500)
			return
		}

		session := data.NewSession(person.ID)
		if err := options.SessionStore.SetSession(sessionID, session); err != nil {
			Error(w, err, 500)
			return
		}

		cookie := &http.Cookie{
			Name:     CookieName,
			Value:    sessionID.String(),
			Domain:   req.URL.Host,
			HttpOnly: true,
			Secure:   req.TLS != nil,
		}
		http.SetCookie(w, cookie)

		w.WriteHeader(http.StatusNoContent)
	})
}

func Register(options *Options) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		errs := make(map[string]string)

		username := req.FormValue("username")
		switch {
		case len(username) < 3:
			errs["username"] = "Usernames must be at least 3 characters long."
		case len(username) > 120:
			errs["username"] = "Usernames cannot be longer than 120 characters."
		case strings.ContainsAny(username, "\t\n\f\r "):
			errs["username"] = "Usernames cannot include spaces."
		}

		password := req.FormValue("password")
		switch {
		case len(password) < 10:
			errs["password"] = "Passwords must be at least 10 characters long."
		case len(password) > 120:
			errs["password"] = "Passwords cannot be longer than 120 characters."
		}

		if len(errs) > 0 {
			InvalidForm(w, errs)
			return
		}

		userID, err := id.New()
		if err != nil {
			Error(w, err, 500)
			return
		}

		person := &data.Person{
			ID:       userID,
			Username: username,
			Email:    req.FormValue("email"),
			LastSeen: time.Now(),
		}

		if err := person.SetPassword(password); err != nil {
			Error(w, err, 500)
			return
		}

		if err := options.Backend.AddPerson(person); err == backend.UsernameInUseError {
			errs["username"] = "Username is already in use."
			InvalidForm(w, errs)
			return
		}

		if err != nil {
			Error(w, err, 500)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
}

func PersonCurrent(options *Options) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		session := ContextSession(req.Context())
		if session == nil {
			Error(w, nil, 500)
		}

		person, err := options.Backend.Person(session.UserID)
		if err != nil {
			Error(w, err, 500)
		}

		WriteJSON(w, person)
	})
}