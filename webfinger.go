package readas

import (
	"github.com/writeas/go-webfinger"
	"github.com/writeas/impart"
	"net/http"
)

type wfResolver struct {
	app *app
}

var wfUserNotFoundErr = impart.HTTPError{http.StatusNotFound, "User not found."}

func (wfr wfResolver) FindUser(username string, host, requestHost string, r []webfinger.Rel) (*webfinger.Resource, error) {
	if username != defaultUser {
		return nil, impart.HTTPError{http.StatusNotFound, "User not found."}
	}

	profileURL := wfr.app.cfg.host + "/" + username
	actorURL := wfr.app.cfg.host + "/api/collections/" + username
	res := webfinger.Resource{
		Subject: "acct:" + username + "@" + host,
		Aliases: []string{
			profileURL,
			actorURL,
		},
		Links: []webfinger.Link{
			{
				HRef: profileURL,
				Type: "text/html",
				Rel:  "https://webfinger.net/rel/profile-page",
			},
			{
				HRef: actorURL,
				Type: "application/activity+json",
				Rel:  "self",
			},
		},
	}
	return &res, nil
}

func (wfr wfResolver) DummyUser(username string, hostname string, r []webfinger.Rel) (*webfinger.Resource, error) {
	return nil, impart.HTTPError{http.StatusNotFound, "User not found."}
}

func (wfr wfResolver) IsNotFoundError(err error) bool {
	// TODO: actually check error value
	return err != nil
}
