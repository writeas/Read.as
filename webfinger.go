package readas

import (
	"github.com/writeas/go-webfinger"
	"github.com/writeas/impart"
	"net/http"
	"strings"
)

type wfResolver struct {
	app *app
}

var wfUserNotFoundErr = impart.HTTPError{http.StatusNotFound, "User not found."}

func (wfr wfResolver) FindUser(username string, host, requestHost string, r []webfinger.Rel) (*webfinger.Resource, error) {
	realHost := wfr.app.cfg.Host[strings.LastIndexByte(wfr.app.cfg.Host, '/')+1:]
	if host != realHost {
		return nil, impart.HTTPError{http.StatusBadRequest, "Host doesn't match"}
	}

	u, err := wfr.app.getLocalUser(username)
	if err != nil {
		return nil, err
	}

	profileURL := wfr.app.cfg.Host + "/" + username
	res := webfinger.Resource{
		Subject: "acct:" + username + "@" + host,
		Aliases: []string{
			profileURL,
			u.AccountRoot(wfr.app),
		},
		Links: []webfinger.Link{
			{
				HRef: profileURL,
				Type: "text/html",
				Rel:  "https://webfinger.net/rel/profile-page",
			},
			{
				HRef: u.AccountRoot(wfr.app),
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
