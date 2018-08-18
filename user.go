package readas

import (
	"github.com/writeas/web-core/activitystreams"
)

// User is a remote user
type User struct {
	activitystreams.Person
	ID int64 `json:"-"`
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
	p.Endpoints.SharedInbox = app.cfg.host + "/inbox"
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
	return app.cfg.host + "/api/collections/" + u.PreferredUsername
}
