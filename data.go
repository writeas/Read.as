package readas

import (
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/writeas/impart"
	"github.com/writeas/web-core/activitypub"
	"github.com/writeas/web-core/activitystreams"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

const (
	mySQLErrDuplicateKey = 1062
)

func initDatabase(app *app) error {
	mysqlConnStr := os.Getenv("RA_MYSQL_CONNECTION")
	if mysqlConnStr == "" {
		return fmt.Errorf("No database configuration. Provide RA_MYSQL_CONNECTION environment variable.")
	}
	var err error
	app.db, err = sql.Open("mysql", mysqlConnStr+"?charset=utf8mb4&parseTime=true")
	if err != nil {
		return err
	}

	return nil
}

func checkData(app *app) error {
	users, err := app.getUsersCount()
	if err != nil {
		return fmt.Errorf("Unable to get users count: %v", err)
	}
	if users == 0 {
		return fmt.Errorf("No users exist. Create one with: readas --user [username] --pass [password]")
	}

	return nil
}

func (app *app) createUser(u *LocalUser) error {
	pub, priv := activitypub.GenerateKeys()
	_, err := app.db.Exec("INSERT INTO localusers (username, password, name, summary, private_key, public_key, created) VALUES (?, ?, ?, ?, ?, ?, NOW())", u.PreferredUsername, u.HashedPass, u.Name, u.Summary, priv, pub)
	return err
}

func (app *app) addUser(u *activitystreams.Person) (int64, error) {
	logInfo("Adding follower")
	t, err := app.db.Begin()
	if err != nil {
		logError("Unable to start transaction: %v", err)
		return 0, err
	}

	stmt := "INSERT INTO users (actor_id, username, type, name, summary, following_iri, followers_iri, inbox_iri, outbox_iri, shared_inbox_iri, avatar, avatar_type) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	res, err := t.Exec(stmt, u.ID, u.PreferredUsername, u.Type, u.Name, u.Summary, u.Following, u.Followers, u.Inbox, u.Outbox, u.Endpoints.SharedInbox, u.Icon.URL, u.Icon.Type)
	if err != nil {
		t.Rollback()
		return 0, err
	}

	followerID, err := res.LastInsertId()
	if err != nil {
		t.Rollback()
		logError("No lastinsertid for followers, rolling back: %v", err)
		return 0, err
	}

	// Add in key
	_, err = t.Exec("INSERT INTO userkeys (id, user_id, public_key) VALUES (?, ?, ?)", u.PublicKey.ID, followerID, u.PublicKey.PublicKeyPEM)
	if err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok {
			if mysqlErr.Number != mySQLErrDuplicateKey {
				t.Rollback()
				logError("Couldn't add follower keys in DB: %v\n", err)
				return 0, err
			}
		} else {
			t.Rollback()
			logError("Couldn't add follower keys in DB: %v\n", err)
			return 0, err
		}
	}

	err = t.Commit()
	if err != nil {
		t.Rollback()
		logError("Rolling back after Commit(): %v\n", err)
		return 0, err
	}

	return followerID, nil
}

func (app *app) getLocalUser(username string) (*LocalUser, error) {
	u := LocalUser{}

	condition := "username = ?"
	value := username
	stmt := "SELECT id, username, password, name, summary, private_key, public_key FROM localusers WHERE " + condition
	err := app.db.QueryRow(stmt, value).Scan(&u.ID, &u.PreferredUsername, &u.HashedPass, &u.Name, &u.Summary, &u.privKey, &u.pubKey)
	switch {
	case err == sql.ErrNoRows:
		return nil, impart.HTTPError{http.StatusNotFound, "User not found"}
	case err != nil:
		return nil, err
	}

	return &u, nil
}

func (app *app) getUsersCount() (uint64, error) {
	var c uint64
	err := app.db.QueryRow("SELECT COUNT(*) FROM localusers").Scan(&c)
	if err != nil {
		logError("Couldn't get users count: %v", err)
		return 0, err
	}

	return c, nil
}

func (app *app) getActor(id string) (*User, error) {
	return app.getUserBy("actor_id = ?", id)
}

func (app *app) getUserBy(condition string, value interface{}) (*User, error) {
	u := User{}

	stmt := "SELECT id, actor_id, username, type, name, summary, following_iri, followers_iri, inbox_iri, outbox_iri, shared_inbox_iri, avatar, avatar_type FROM users WHERE " + condition
	err := app.db.QueryRow(stmt, value).Scan(&u.ID, &u.BaseObject.ID, &u.PreferredUsername, &u.Type, &u.Name, &u.Summary, &u.Following, &u.Followers, &u.Inbox, &u.Outbox, &u.Endpoints.SharedInbox, &u.Icon.URL, &u.Icon.Type)
	switch {
	case err == sql.ErrNoRows:
		return nil, impart.HTTPError{http.StatusNotFound, "User not found"}
	case err != nil:
		return nil, err
	}

	return &u, nil
}
