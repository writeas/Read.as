package readas

import (
	"encoding/json"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/writeas/activity/streams"
	"github.com/writeas/impart"
	"github.com/writeas/web-core/activitystreams"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func handleFetchUser(app *app, w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Server", serverName)

	vars := mux.Vars(r)
	alias := vars["alias"]
	u, err := app.getLocalUser(alias)
	if err != nil {
		return err
	}

	p := u.AsPerson(app)

	return impart.RenderActivityJSON(w, p, http.StatusOK)
}

func handleFetchOutbox(app *app, w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Server", serverName)

	vars := mux.Vars(r)
	alias := vars["alias"]

	u, err := app.getLocalUser(alias)
	if err != nil {
		return err
	}

	accountRoot := u.AccountRoot(app)
	posts := 0 // app.getUserPostsCount(u.PreferredUsername)

	page := r.FormValue("page")
	p, err := strconv.Atoi(page)
	if err != nil || p < 1 {
		// Return outbox
		oc := activitystreams.NewOrderedCollection(accountRoot, "outbox", posts)
		return impart.RenderActivityJSON(w, oc, http.StatusOK)
	}

	// Return outbox page
	ocp := activitystreams.NewOrderedCollectionPage(accountRoot, "outbox", posts, p)
	ocp.OrderedItems = []interface{}{}

	/*
		posts, err := app.getUserPosts(u.PreferredUsername)
		for _, p := range *posts {
			o := p.ActivityObject()
			a := activitystreams.NewCreateActivity(o)
			ocp.OrderedItems = append(ocp.OrderedItems, *a)
		}
	*/

	return impart.RenderActivityJSON(w, ocp, http.StatusOK)
}

func handleFetchFollowers(app *app, w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Server", serverName)

	vars := mux.Vars(r)
	alias := vars["alias"]

	u, err := app.getLocalUser(alias)
	if err != nil {
		return err
	}

	accountRoot := u.AccountRoot(app)

	page := r.FormValue("page")
	p, err := strconv.Atoi(page)
	if err != nil {
		p = 0
	}

	// TODO: fetch full Users
	folls, err := app.getFollowers(u.ID, p)
	if err != nil {
		return err
	}
	followersCount := len(*folls)

	if p < 1 {
		// Return outbox
		oc := activitystreams.NewOrderedCollection(accountRoot, "followers", followersCount)
		return impart.RenderActivityJSON(w, oc, http.StatusOK)
	}

	// Return outbox page
	ocp := activitystreams.NewOrderedCollectionPage(accountRoot, "followers", followersCount, p)
	ocp.OrderedItems = []interface{}{}
	for _, f := range *folls {
		ocp.OrderedItems = append(ocp.OrderedItems, f)
	}
	return impart.RenderActivityJSON(w, ocp, http.StatusOK)
}

func handleFetchFollowing(app *app, w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Server", serverName)

	vars := mux.Vars(r)
	alias := vars["alias"]

	u, err := app.getLocalUser(alias)
	if err != nil {
		return err
	}

	accountRoot := u.AccountRoot(app)

	page := r.FormValue("page")
	p, err := strconv.Atoi(page)
	if err != nil {
		p = 0
	}

	// TODO: fetch full Users
	folls, err := app.getFollowing(u.ID, p)
	if err != nil {
		return err
	}
	followingCount := len(*folls)

	if p < 1 {
		// Return outbox
		oc := activitystreams.NewOrderedCollection(accountRoot, "following", followingCount)
		return impart.RenderActivityJSON(w, oc, http.StatusOK)
	}

	// Return outbox page
	ocp := activitystreams.NewOrderedCollectionPage(accountRoot, "following", followingCount, p)
	ocp.OrderedItems = []interface{}{}
	for _, f := range *folls {
		ocp.OrderedItems = append(ocp.OrderedItems, f)
	}
	return impart.RenderActivityJSON(w, ocp, http.StatusOK)
}

func handleFetchInbox(app *app, w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Server", serverName)

	vars := mux.Vars(r)
	alias := vars["alias"]
	var u *LocalUser
	var err error
	if alias != "" {
		u, err = app.getLocalUser(alias)
		if err != nil {
			// TODO: return Reject?
			return err
		}
	}

	err = verifyRequest(app, r)
	if err != nil {
		logError("Unable to verify signature: %v", err)
		return err
	}
	logInfo("Signature OK")

	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		logError("Can't dump: %v", err)
	} else {
		logInfo("Rec'd! %q", dump)
	}

	var m map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		return err
	}

	a := streams.NewAccept()
	var to *url.URL
	var isFollow, isUnfollow, isAccept, isCreate bool
	fullActor := &activitystreams.Person{}
	var remoteUser *User

	res := &streams.Resolver{
		FollowCallback: func(f *streams.Follow) error {
			isFollow = true

			// 1) Use the Follow concrete type here
			// 2) Errors are propagated to res.Deserialize call below
			m["@context"] = []string{activitystreams.Namespace}
			b, _ := json.Marshal(m)
			logInfo("Follow: %s", b)

			a.AppendObject(f.Raw())
			_, to = f.GetActor(0)
			obj := f.Raw().GetObjectIRI(0)
			a.AppendActor(obj)

			// First get actor information
			if to == nil {
				return fmt.Errorf("No valid `to` string")
			}
			fullActor, remoteUser, err = fetchActor(app, to.String())
			if err != nil {
				return err
			}
			return impart.RenderActivityJSON(w, nil, http.StatusAccepted)
		},
		UndoCallback: func(u *streams.Undo) error {
			isUnfollow = true

			m["@context"] = []string{activitystreams.Namespace}
			b, _ := json.Marshal(m)
			logInfo("Undo: %s", b)

			a.AppendObject(u.Raw())
			var res streams.Resolution
			res, to = u.GetActor(0)
			// TODO: get actor from object.object, not object
			obj := u.Raw().GetObjectIRI(0)
			a.AppendActor(obj)
			if res != streams.Unresolved {
				// Populate fullActor from DB?
				remoteUser, err = app.getActor(to.String())
				if err != nil {
					if iErr, ok := err.(*impart.HTTPError); ok {
						if iErr.Status == http.StatusNotFound {
							logError("No remoteuser info for Undo event!")
						}
					}
					return err
				} else {
					fullActor = remoteUser.AsPerson()
				}
			} else {
				logError("No to on Undo!")
			}
			return impart.RenderActivityJSON(w, m, http.StatusAccepted)
		},
		AcceptCallback: func(f *streams.Accept) error {
			isAccept = true

			b, _ := json.Marshal(m)
			logInfo("Accept: %s", b)
			_, actorIRI := f.GetActor(0)
			fullActor, remoteUser, err = fetchActor(app, actorIRI.String())
			if u == nil {
				// This is the shared inbox
				logInfo("Shared inbox; fetching local user")

				_, err = f.ResolveObject(&streams.Resolver{
					FollowCallback: func(a *streams.Follow) error {
						_, localActor := a.GetActor(0)
						localUsername := localActor.String()[strings.LastIndex(localActor.String(), "/")+1:]
						logInfo("Is Follow! Getting user: %s", localUsername)
						u, err = app.getLocalUser(localUsername)
						if err != nil {
							return err
						}
						return nil
					},
				}, 0)
				if err != nil {
					return err
				}
			}
			return impart.RenderActivityJSON(w, nil, http.StatusAccepted)
		},
		CreateCallback: func(f *streams.Create) error {
			// TODO: move everything to handleCreateCallback
			isCreate = true

			_, id := f.GetId()
			_, actorIRI := f.GetActor(0)
			var published time.Time
			var postType, name, content string
			var url *url.URL
			artRes := &streams.Resolver{
				ArticleCallback: func(a *streams.Article) error {
					_, published = a.GetPublished()
					_, postType = a.GetType(0)
					_, url = a.GetUrl(0)
					_, name = a.GetName(0)
					_, content = a.GetContent(0)
					return nil
				},
				NoteCallback: func(a *streams.Note) error {
					_, published = a.GetPublished()
					_, postType = a.GetType(0)
					_, url = a.GetUrl(0)
					_, content = a.GetContent(0)
					return nil
				},
			}
			_, err = f.ResolveObject(artRes, 0)
			if err != nil {
				return err
			}
			// FIXME: this won't work if the user doesn't exist locally
			fullActor, remoteUser, err = fetchActor(app, actorIRI.String())
			if err != nil {
				return err
			}

			// Insert post
			err = app.createPost(&Post{
				ActivityID: id.String(),
				Type:       postType,
				Published:  published,
				URL:        url.String(),
				Name:       name,
				Content:    content,
				OwnerID:    remoteUser.ID,
				actorID:    actorIRI.String(),
			})
			if err != nil {
				return err
			}

			return impart.RenderActivityJSON(w, nil, http.StatusAccepted)
		},
		UpdateCallback: func(f *streams.Update) error {
			isCreate = true

			_, id := f.GetId()
			_, actorIRI := f.GetActor(0)
			var name, content string
			var url *url.URL
			artRes := &streams.Resolver{
				ArticleCallback: func(a *streams.Article) error {
					_, url = a.GetUrl(0)
					_, name = a.GetName(0)
					_, content = a.GetContent(0)
					return nil
				},
				NoteCallback: func(a *streams.Note) error {
					_, url = a.GetUrl(0)
					_, name = a.GetName(0)
					_, content = a.GetContent(0)
					return nil
				},
			}
			_, err = f.ResolveObject(artRes, 0)
			if err != nil {
				return err
			}
			// FIXME: this won't work if the user doesn't exist locally
			fullActor, remoteUser, err = fetchActor(app, actorIRI.String())
			if err != nil {
				return err
			}

			// Update post
			err = app.updatePost(&Post{
				ActivityID: id.String(),
				URL:        url.String(),
				Name:       name,
				Content:    content,
			})
			if err != nil {
				return err
			}

			return impart.RenderActivityJSON(w, nil, http.StatusAccepted)
		},
		DeleteCallback: func(f *streams.Delete) error {
			isCreate = true

			_, id := f.GetId()

			// Delete post
			err = app.deletePost(id.String())
			if err != nil {
				return err
			}

			return impart.RenderActivityJSON(w, nil, http.StatusAccepted)
		},
	}
	if err := res.Deserialize(m); err != nil {
		// 3) Any errors from #2 can be handled, or the payload is an unknown type.
		logError("Unable to resolve Follow: %v", err)
		logError("Map: %s", m)
		return err
	}

	if isAccept {
		_, err = app.db.Exec("INSERT INTO follows (follower, followee, created) VALUES (?, ?, NOW())", u.ID, remoteUser.ID)
		if err != nil {
			if mysqlErr, ok := err.(*mysql.MySQLError); ok {
				if mysqlErr.Number != mySQLErrDuplicateKey {
					logError("Couldn't add follower in DB on accept: %v\n", err)
					return err
				}
			} else {
				logError("Couldn't add follower in DB on accept: %v\n", err)
				return err
			}
		}
		fetchUserPosts(app, remoteUser)
	} else if isCreate {
		return nil
	}

	p := u.AsPerson(app)
	go func() {
		time.Sleep(2 * time.Second)
		am, err := a.Serialize()
		if err != nil {
			logError("Unable to serialize Accept: %v", err)
			return
		}
		am["@context"] = []string{activitystreams.Namespace}

		if to == nil {
			logError("No to! %v", err)
			return
		}
		err = makeActivityPost(p, fullActor.Inbox, am)
		if err != nil {
			logError("Unable to make activity POST: %v", err)
			return
		}

		if isFollow {
			var followerID int64
			if remoteUser != nil {
				followerID = remoteUser.ID
			} else {
				followerID, err = app.addUser(fullActor)
				if err != nil {
					return
				}
			}

			// Add follow
			_, err = app.db.Exec("INSERT INTO follows (follower, followee, created) VALUES (?, ?, NOW())", followerID, u.ID)
			if err != nil {
				if mysqlErr, ok := err.(*mysql.MySQLError); ok {
					if mysqlErr.Number != mySQLErrDuplicateKey {
						logError("Couldn't add follower in DB: %v\n", err)
						return
					}
				} else {
					logError("Couldn't add follower in DB: %v\n", err)
					return
				}
			}
		} else if isUnfollow {
			// Remove follower locally
			_, err = app.db.Exec("DELETE FROM follows WHERE followee = ? AND follower = (SELECT id FROM users WHERE actor_id = ?)", u.ID, to.String())
			if err != nil {
				logError("Couldn't remove follower from DB: %v\n", err)
			}
		}
	}()

	return nil
}

func handleFollowUser(app *app, w http.ResponseWriter, r *http.Request) error {
	// Get logged-in user
	cu := getUserSession(app, r)
	if cu == nil {
		return impart.HTTPError{http.StatusUnauthorized, "Not logged in."}
	}

	handle := r.FormValue("user")
	// Make webfinger request
	userItems := strings.Split(handle, "@")
	wfr, err := doWebfinger(userItems[1], userItems[0])
	if err != nil {
		logInfo("Webfinger failed: %v", err)
		return err
	}

	// Save webfinger result
	logInfo("Webfinger success. Saving: %+v", wfr)
	app.addFoundUser(wfr)

	remoteUser, err := app.getActor(wfr.ActorIRI)
	if err != nil {
		if iErr, ok := err.(impart.HTTPError); ok {
			if iErr.Status == http.StatusNotFound {
				// Look up actor
				remotePerson, _, err := fetchActor(app, wfr.ActorIRI)
				if err != nil {
					logInfo("Actor fetch failed: %+v", err)
					return err
				}

				// Save user locally
				logInfo("Actor fetch success")
				//logInfo("Actor fetch success: %+v", remotePerson)
				_, err = app.addUser(remotePerson)
				if err != nil {
					return err
				}
				remoteUser, err = app.getActor(wfr.ActorIRI)
				if err != nil {
					return err
				}
			} else {
				logError("Not NotFound error: %+v", err)
			}
		} else {
			logError("Unable to get actor: %+v", err)
		}
	} else {
		logInfo("Actor is local")
	}

	// Send follow request
	u, err := app.getLocalUser(cu.PreferredUsername)
	if err != nil {
		return err
	}
	followActivity := activitystreams.NewFollowActivity(u.AccountRoot(app), wfr.ActorIRI)
	followActivity.ID = u.AccountRoot(app) + "#follow"
	err = makeActivityPost(u.AsPerson(app), remoteUser.Inbox, followActivity)
	if err != nil {
		logError("Couldn't post! %v", err)
	}
	return impart.WriteSuccess(w, "", http.StatusOK)
}

func fetchUserPosts(app *app, u *User) error {
	return fetchActorOutbox(app, u.Outbox)
}

func handleCreateCallback(app *app, f *streams.Create) error {
	_, id := f.GetId()
	_, actorIRI := f.GetActor(0)
	var published time.Time
	var postType, name, content string
	var url *url.URL
	artRes := &streams.Resolver{
		ArticleCallback: func(a *streams.Article) error {
			_, published = a.GetPublished()
			_, postType = a.GetType(0)
			_, url = a.GetUrl(0)
			_, name = a.GetName(0)
			_, content = a.GetContent(0)
			return nil
		},
		NoteCallback: func(a *streams.Note) error {
			_, published = a.GetPublished()
			_, postType = a.GetType(0)
			_, url = a.GetUrl(0)
			_, content = a.GetContent(0)
			return nil
		},
	}
	_, err := f.ResolveObject(artRes, 0)
	if err != nil {
		return err
	}
	// FIXME: this won't work if the user doesn't exist locally
	_, remoteUser, err := fetchActor(app, actorIRI.String())
	if err != nil {
		return err
	}

	// Insert post
	err = app.createPost(&Post{
		ActivityID: id.String(),
		Type:       postType,
		Published:  published,
		URL:        url.String(),
		Name:       name,
		Content:    content,
		OwnerID:    remoteUser.ID,
		actorID:    actorIRI.String(),
	})
	if err != nil {
		return err
	}

	return nil
}
