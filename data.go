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
	t, err := app.db.Begin()
	if err != nil {
		logError("Unable to start transaction: %v", err)
		return err
	}

	res, err := t.Exec("INSERT INTO users (actor_id, username, password, name, summary, created) VALUES (?, ?, ?, ?, ?, NOW())", u.PreferredUsername, u.PreferredUsername, u.HashedPass, u.Name, u.Summary)
	if err != nil {
		t.Rollback()
		return err
	}

	userID, err := res.LastInsertId()
	if err != nil {
		t.Rollback()
		return err
	}

	_, err = t.Exec("INSERT INTO userkeys (id, user_id, public_key, private_key) VALUES (?, ?, ?, ?)", u.PreferredUsername+"#main-key", userID, pub, priv)
	if err != nil {
		t.Rollback()
		return err
	}

	err = t.Commit()
	if err != nil {
		t.Rollback()
		logError("Rolling back after Commit(): %v\n", err)
		return err
	}
	return nil
}

func (app *app) addFoundUser(wfr *WebfingerResult) error {
	stmt := "INSERT INTO foundusers (username, host, actor_id) VALUES (?, ?, ?)"
	_, err := app.db.Exec(stmt, wfr.Username, wfr.Host, wfr.ActorIRI)
	return err
}

func (app *app) addUser(u *activitystreams.Person) (int64, error) {
	logInfo("Adding follower")
	t, err := app.db.Begin()
	if err != nil {
		logError("Unable to start transaction: %v", err)
		return 0, err
	}

	stmt := "INSERT INTO users (actor_id, username, type, name, summary, created, url, following_iri, followers_iri, inbox_iri, outbox_iri, shared_inbox_iri, avatar, avatar_type) VALUES (?, ?, ?, ?, ?, NOW(), ?, ?, ?, ?, ?, ?, ?, ?)"
	res, err := t.Exec(stmt, u.BaseObject.ID, u.PreferredUsername, u.Type, u.Name, u.Summary, u.URL, u.Following, u.Followers, u.Inbox, u.Outbox, u.Endpoints.SharedInbox, u.Icon.URL, u.Icon.Type)
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

	// Update found users
	stmt = "UPDATE foundusers SET user_id = ? WHERE actor_id = ?"
	_, err = t.Exec(stmt, followerID, u.BaseObject.ID)
	if err != nil {
		t.Rollback()
		return 0, err
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
	stmt := "SELECT u.id, username, password, name, summary, private_key, public_key FROM users u LEFT JOIN userkeys uk ON u.id = uk.user_id WHERE " + condition
	err := app.db.QueryRow(stmt, value).Scan(&u.ID, &u.PreferredUsername, &u.HashedPass, &u.Name, &u.Summary, &u.privKey, &u.pubKey)
	switch {
	case err == sql.ErrNoRows:
		return nil, impart.HTTPError{http.StatusNotFound, "User not found"}
	case err != nil:
		return nil, err
	}

	return &u, nil
}

func (app *app) getFollowers(id int64, page int) (*[]string, error) {
	limitStr := ""
	if page > 0 {
		pagePosts := 10
		start := page*pagePosts - pagePosts
		limitStr = fmt.Sprintf(" LIMIT %d, %d", start, pagePosts)
	}

	rows, err := app.db.Query(`SELECT actor_id
		FROM follows
		LEFT JOIN users
			ON follower = id
		WHERE followee = ?`+limitStr, id)
	if err != nil {
		logError("Failed selecting followers: %v", err)
		return nil, impart.HTTPError{http.StatusInternalServerError, "Couldn't retrieve followers."}
	}
	defer rows.Close()

	users := []string{}
	for rows.Next() {
		u := ""
		err = rows.Scan(&u)
		if err != nil {
			logError("Failed scanning row in getFollowers: %v", err)
			break
		}

		users = append(users, u)
	}
	err = rows.Err()
	if err != nil {
		logError("Error after Next() on rows in getFollowers: %v", err)
	}

	return &users, nil
}

func (app *app) getFollowing(id int64, page int) (*[]string, error) {
	limitStr := ""
	if page > 0 {
		pagePosts := 10
		start := page*pagePosts - pagePosts
		limitStr = fmt.Sprintf(" LIMIT %d, %d", start, pagePosts)
	}

	rows, err := app.db.Query(`SELECT actor_id
		FROM follows
		LEFT JOIN users
			ON followee = id
		WHERE follower = ?`+limitStr, id)
	if err != nil {
		logError("Failed selecting following: %v", err)
		return nil, impart.HTTPError{http.StatusInternalServerError, "Couldn't retrieve following."}
	}
	defer rows.Close()

	users := []string{}
	for rows.Next() {
		u := ""
		err = rows.Scan(&u)
		if err != nil {
			logError("Failed scanning row in getFollowing: %v", err)
			break
		}

		users = append(users, u)
	}
	err = rows.Err()
	if err != nil {
		logError("Error after Next() on rows in getFollowing: %v", err)
	}

	return &users, nil
}

func (app *app) getUsersCount() (uint64, error) {
	var c uint64
	err := app.db.QueryRow("SELECT COUNT(*) FROM users WHERE password IS NOT NULL").Scan(&c)
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

	stmt := `SELECT id, actor_id, u.username, type, name, summary, created, url, following_iri, followers_iri, inbox_iri, outbox_iri, shared_inbox_iri, avatar, avatar_type, host
		FROM users u
			INNER JOIN foundusers
			USING (actor_id)
		WHERE ` + condition
	err := app.db.QueryRow(stmt, value).Scan(&u.ID, &u.BaseObject.ID, &u.PreferredUsername, &u.Type, &u.Name, &u.Summary, &u.Created, &u.URL, &u.Following, &u.Followers, &u.Inbox, &u.Outbox, &u.Endpoints.SharedInbox, &u.Icon.URL, &u.Icon.Type, &u.Host)
	switch {
	case err == sql.ErrNoRows:
		return nil, impart.HTTPError{http.StatusNotFound, "User not found"}
	case err != nil:
		return nil, err
	}

	return &u, nil
}

func (app *app) createPost(p *Post) error {
	_, err := app.db.Exec("INSERT INTO posts (owner_id, activity_id, type, published, url, name, content) VALUES ((SELECT id FROM users WHERE actor_id = ?), ?, ?, ?, ?, ?, ?)", p.actorID, p.ActivityID, p.Type, p.Published, p.URL, p.Name, p.Content)
	return err
}

func (app *app) updatePost(p *Post) error {
	_, err := app.db.Exec("UPDATE posts SET url = ?, name = ?, content = ? WHERE activity_id = ?", p.URL, p.Name, p.Content, p.ActivityID)
	return err
}

func (app *app) deletePost(postID string) error {
	_, err := app.db.Exec("DELETE FROM posts WHERE activity_id = ?", postID)
	return err
}

func (app *app) getUserFeed(id int64, page int) (*[]Post, error) {
	pagePosts := 10
	start := page*pagePosts - pagePosts
	if page == 0 {
		start = 0
		pagePosts = 100
	}

	limitStr := ""
	if page > 0 {
		limitStr = fmt.Sprintf(" LIMIT %d, %d", start, pagePosts)
	}
	rows, err := app.db.Query(`SELECT p.id, owner_id, activity_id, p.type, published, p.url, p.name, content, f.host, u.username, u.name, u.url
		FROM posts p
		INNER JOIN users u
			ON owner_id = u.id
		LEFT JOIN foundusers f
			USING(actor_id)
		WHERE owner_id 
			IN (SELECT followee FROM follows WHERE follower = ?)
		ORDER BY published DESC `+limitStr, id)
	if err != nil {
		logError("Failed selecting from posts: %v", err)
		return nil, impart.HTTPError{http.StatusInternalServerError, "Couldn't retrieve user feed."}
	}
	defer rows.Close()

	// TODO: extract this common row scanning logic for queries using `postCols`
	posts := []Post{}
	for rows.Next() {
		p := Post{
			Owner:    &User{},
			IsInFeed: true,
		}
		err = rows.Scan(&p.ID, &p.OwnerID, &p.ActivityID, &p.Type, &p.Published, &p.URL, &p.Name, &p.Content, &p.Owner.Host, &p.Owner.PreferredUsername, &p.Owner.Name, &p.Owner.URL)
		if err != nil {
			logError("Failed scanning row: %v", err)
			break
		}

		posts = append(posts, p)
	}
	err = rows.Err()
	if err != nil {
		logError("Error after Next() on rows: %v", err)
	}

	return &posts, nil
}

func (app *app) getPost(id int64) (*Post, error) {
	p := Post{
		Owner:    &User{},
		IsInFeed: false,
	}
	stmt := `SELECT p.id, owner_id, activity_id, p.type, published, p.url, p.name, content, f.host, u.username, u.name, u.url
		FROM posts p
		INNER JOIN users u
			ON owner_id = u.id
		LEFT JOIN foundusers f
			USING(actor_id)
		WHERE p.id = ?`
	err := app.db.QueryRow(stmt, id).Scan(&p.ID, &p.OwnerID, &p.ActivityID, &p.Type, &p.Published, &p.URL, &p.Name, &p.Content, &p.Owner.Host, &p.Owner.PreferredUsername, &p.Owner.Name, &p.Owner.URL)
	switch {
	case err == sql.ErrNoRows:
		return nil, impart.HTTPError{http.StatusNotFound, "Post not found"}
	case err != nil:
		return nil, err
	}
	return &p, err
}

func (app *app) getActorKey(id string) ([]byte, error) {
	k := []byte{}

	stmt := "SELECT public_key FROM userkeys WHERE id = ?"
	err := app.db.QueryRow(stmt, id).Scan(&k)
	switch {
	case err == sql.ErrNoRows:
		return nil, impart.HTTPError{http.StatusNotFound, "Key not found"}
	case err != nil:
		return nil, err
	}

	return k, nil
}
