package readas

import (
	"github.com/microcosm-cc/bluemonday"
	"html/template"
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

	actorID string

	Owner *User
}

func (p *Post) SanitaryContent() template.HTML {
	// Strip out bad HTML
	policy := getSanitizationPolicy()
	policy.RequireNoFollowOnLinks(true)
	return template.HTML(policy.Sanitize(p.Content))
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
