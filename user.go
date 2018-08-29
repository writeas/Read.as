package readas

import (
	"github.com/writeas/impart"
	"github.com/writeas/web-core/activitystreams"
	"github.com/writeas/web-core/auth"
	"net/http"
	"time"
)

// User is a remote user
type User struct {
	activitystreams.Person
	ID      int64     `json:"-"`
	Host    string    `json:"-"`
	Created time.Time `json:"-"`
}

func (u *User) AsPerson() *activitystreams.Person {
	p := u.Person
	p.Context = []interface{}{
		activitystreams.Namespace,
	}
	return &p
}

// LocalUser is a local user
type LocalUser struct {
	ID                int64  `json:"-"`
	PreferredUsername string `json:"preferredUsername"`
	HashedPass        []byte `json:"-"`
	Name              string `json:"name"`
	Summary           string `json:"summary"`
	privKey           []byte
	pubKey            []byte
}

func (u *LocalUser) AsPerson(app *app) *activitystreams.Person {
	accountRoot := u.AccountRoot(app)
	p := activitystreams.NewPerson(accountRoot)
	p.Endpoints.SharedInbox = app.cfg.Host + "/api/inbox"
	p.PreferredUsername = u.PreferredUsername
	p.URL = app.cfg.Host + "/" + u.PreferredUsername
	p.Name = u.Name
	p.Summary = u.Summary

	// Add key
	p.Context = append(p.Context, "https://w3id.org/security/v1")
	p.PublicKey = activitystreams.PublicKey{
		ID:           p.ID + "#main-key",
		Owner:        p.ID,
		PublicKeyPEM: string(u.pubKey),
	}
	p.SetPrivKey(u.privKey)
	return p
}

func (u *LocalUser) AccountRoot(app *app) string {
	return app.cfg.Host + "/api/collections/" + u.PreferredUsername
}

func (u *LocalUser) cookie() LocalUser {
	return LocalUser{
		ID:                u.ID,
		PreferredUsername: u.PreferredUsername,
	}
}

func handleLogin(app *app, w http.ResponseWriter, r *http.Request) error {
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" {
		msg := "A username is required."
		return impart.HTTPError{http.StatusBadRequest, msg}
	}
	if password == "" {
		msg := "A password is required."
		return impart.HTTPError{http.StatusBadRequest, msg}
	}

	to := "/"
	authUser, err := app.getLocalUser(username)
	if err != nil {
		return err
	}

	if !auth.Authenticated(authUser.HashedPass, []byte(password)) {
		return impart.HTTPError{http.StatusUnauthorized, "Incorrect password."}
	}

	// Set cookie
	session, err := app.sStore.Get(r, "u")
	if err != nil {
		// The cookie should still save, even if there's an error.
		logError("Login: Session: %v; ignoring", err)
	}

	// Remove unwanted data
	session.Values["user"] = authUser.cookie()
	err = session.Save(r, w)
	if err != nil {
		logError("Login: Couldn't save session: %v", err)
		// TODO: return error
	}

	if redir := r.FormValue("to"); redir != "" {
		to = redir
	}
	return impart.HTTPError{http.StatusFound, to}
}

func handleLogout(app *app, w http.ResponseWriter, r *http.Request) error {
	session, err := app.sStore.Get(r, "u")
	if err != nil {
		return err
	}

	val := session.Values["user"]
	if _, ok := val.(*LocalUser); !ok {
		logError("Error casting user object on logout. Vals: %+v Resetting cookie.", session.Values)

		err = session.Save(r, w)
		if err != nil {
			logError("Couldn't save session on logout: %v", err)
			return impart.HTTPError{http.StatusInternalServerError, "Unable to save cookie session."}
		}

		return impart.HTTPError{http.StatusFound, "/"}
	}

	session.Options.MaxAge = -1

	err = session.Save(r, w)
	if err != nil {
		logError("Couldn't save session on logout: %v", err)
		return impart.HTTPError{http.StatusInternalServerError, "Unable to save cookie session."}
	}

	return impart.HTTPError{http.StatusFound, "/"}
}
