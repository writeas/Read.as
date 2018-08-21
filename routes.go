package readas

import (
	"github.com/gorilla/mux"
	"github.com/writeas/go-nodeinfo"
	"github.com/writeas/go-webfinger"
	"net/http"
)

func initRoutes(app *app) {
	app.router = mux.NewRouter()

	// Federation endpoints
	wf := webfinger.Default(wfResolver{app})
	wf.NoTLSHandler = nil
	// host-meta
	app.router.HandleFunc("/.well-known/host-meta", app.handler(handleViewHostMeta))
	// webfinger
	app.router.HandleFunc(webfinger.WebFingerPath, http.HandlerFunc(wf.Webfinger))
	// nodeinfo
	niCfg := nodeInfoConfig(app.cfg)
	ni := nodeinfo.NewService(*niCfg, nodeInfoResolver{app})
	app.router.HandleFunc(nodeinfo.NodeInfoPath, http.HandlerFunc(ni.NodeInfoDiscover))
	app.router.HandleFunc(niCfg.InfoURL, http.HandlerFunc(ni.NodeInfo))

	api := app.router.PathPrefix("/api/").Subrouter()
	api.HandleFunc("/auth/login", app.handler(handleLogin)).Methods("POST")
	api.HandleFunc("/collections/{alias}", app.handler(handleFetchUser)).Methods("GET")
	collectionsAPI := api.PathPrefix("/collections/{alias}").Subrouter()
	collectionsAPI.HandleFunc("/", app.handler(handleFetchUser)).Methods("GET")
	collectionsAPI.HandleFunc("/inbox", app.handler(handleFetchInbox)).Methods("POST")
	collectionsAPI.HandleFunc("/outbox", app.handler(handleFetchOutbox)).Methods("GET")
	collectionsAPI.HandleFunc("/following", app.handler(handleFetchFollowing)).Methods("GET")
	collectionsAPI.HandleFunc("/followers", app.handler(handleFetchFollowers)).Methods("GET")

	api.HandleFunc("/follow", app.handler(handleFollowUser))
	api.HandleFunc("/inbox", app.handler(handleFetchInbox))

	app.router.HandleFunc("/logout", app.handler(handleLogout))
	app.router.HandleFunc("/", app.handler(handleViewHome))
	app.router.PathPrefix("/").Handler(http.FileServer(http.Dir("static/")))
}

func handleViewHome(app *app, w http.ResponseWriter, r *http.Request) error {
	cu := getUserSession(app, r)
	var u *LocalUser
	var err error
	if cu != nil {
		u, err = app.getLocalUser(cu.PreferredUsername)
		if err != nil {
			return err
		}
	}

	p := struct {
		User     *LocalUser
		Username string
		Flash    string
		To       string
	}{
		User:     u,
		Username: r.FormValue("username"),
		To:       r.FormValue("to"),
	}

	if err := renderTemplate(w, "index", p); err != nil {
		return err
	}
	return nil
}
