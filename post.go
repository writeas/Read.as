package readas

import (
	"github.com/gorilla/mux"
	"github.com/microcosm-cc/bluemonday"
	"html/template"
	"net/http"
	"strconv"
	"time"
)

type Post struct {
	ID         int64
	OwnerID    int64
	ActivityID string
	Type       string
	Published  time.Time
	URL        string
	Name       string
	Content    string

	actorID  string
	IsInFeed bool

	Owner *User
}

func (p *Post) SanitaryContent() template.HTML {
	// Strip out bad HTML
	policy := getSanitizationPolicy()
	policy.RequireNoFollowOnLinks(true)
	return template.HTML(policy.Sanitize(p.Content))
}

func (p *Post) DisplayTitle() string {
	t := "A post"
	if p.Name != "" {
		t = p.Name
	}
	return t + " by " + p.Owner.Name
}

func (p *Post) Summary() string {
	// TODO: return truncated summary of post.
	return ""
}

func (p *Post) PublishedDate() string {
	return p.Published.Format("2006-01-02")
}

func (p *Post) Published8601() string {
	return p.Published.Format("2006-01-02T15:04:05Z")
}

func getSanitizationPolicy() *bluemonday.Policy {
	policy := bluemonday.UGCPolicy()
	policy.AllowAttrs("src", "style").OnElements("iframe", "video")
	policy.AllowAttrs("frameborder", "width", "height").Matching(bluemonday.Integer).OnElements("iframe")
	policy.AllowAttrs("allowfullscreen").OnElements("iframe")
	policy.AllowAttrs("controls", "loop", "muted", "autoplay").OnElements("video")
	policy.AllowAttrs("target").OnElements("a")
	policy.AllowAttrs("style", "class", "id").Globally()
	policy.AllowURLSchemes("http", "https", "mailto", "xmpp")
	return policy
}

func handleViewPost(app *app, w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}

	cu := getUserSession(app, r)
	var u *LocalUser
	if cu != nil {
		u, err = app.getLocalUser(cu.PreferredUsername)
		if err != nil {
			return err
		}
	}

	p := struct {
		User         *LocalUser
		Version      string
		InstanceName string
		Post         *Post
	}{
		User:         u,
		Version:      softwareVersion,
		InstanceName: app.cfg.Name,
		Post:         &Post{},
	}
	p.Post, err = app.getPost(int64(id))
	if err != nil {
		return err
	}

	return renderTemplate(w, "post", p)
}
