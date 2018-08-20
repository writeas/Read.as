package readas

import (
	"encoding/gob"
	"github.com/gorilla/sessions"
	"net/http"
	"strings"
)

const (
	day           = 86400
	sessionLength = 180 * day
)

// initSession creates the cookie store. It depends on the keychain already
// being loaded.
func initSession(app *app) error {
	// Register complex data types we'll be storing in cookies
	gob.Register(&LocalUser{})

	// Create the cookie store
	app.sStore = sessions.NewCookieStore(app.keys.cookieAuthKey, app.keys.cookieKey)
	app.sStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   sessionLength,
		HttpOnly: true,
		Secure:   strings.HasPrefix(app.cfg.host, "https://"),
	}
	return nil
}

func getUserSession(app *app, r *http.Request) *LocalUser {
	session, err := app.sStore.Get(r, "u")
	if err == nil {
		// Got the currently logged-in user
		val := session.Values["user"]
		var u = &LocalUser{}
		var ok bool
		if u, ok = val.(*LocalUser); ok {
			return u
		}
	}

	return nil
}
