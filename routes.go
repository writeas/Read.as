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
	api.HandleFunc("/collections/{alias}", app.handler(handleFetchUser)).Methods("GET")
	collectionsAPI := api.PathPrefix("/collections/{alias}").Subrouter()
	collectionsAPI.HandleFunc("/", app.handler(handleFetchUser)).Methods("GET")
	collectionsAPI.HandleFunc("/inbox", app.handler(handleFetchInbox)).Methods("POST")
	collectionsAPI.HandleFunc("/outbox", app.handler(handleFetchOutbox)).Methods("GET")
	collectionsAPI.HandleFunc("/following", app.handler(handleFetchFollowing)).Methods("GET")
	collectionsAPI.HandleFunc("/followers", app.handler(handleFetchFollowers)).Methods("GET")

	api.HandleFunc("/follow", app.handler(handleFollowUser))
}
