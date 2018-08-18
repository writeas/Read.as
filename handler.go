package readas

import (
	"github.com/writeas/impart"
	"log"
	"net/http"
	"strings"
)

type handlerFunc func(app *app, w http.ResponseWriter, r *http.Request) error

func (app *app) handler(h handlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleError(w, r, func() error {
			return h(app, w, r)
		}())
	}
}

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

	isAPI := strings.HasPrefix(r.URL.Path, "/api")

	if err, ok := err.(impart.HTTPError); ok {
		log.Printf("Error: %v", err)
		if err.Status >= 300 && err.Status < 400 {
			impart.WriteRedirect(w, err)
			return
		} else if err.Status == http.StatusUnauthorized {
			if isAPI {
				impart.WriteError(w, err)
			} else {
				impart.WriteRedirect(w, impart.HTTPError{http.StatusFound, "/login?to=" + r.URL.Path})
			}
			return
		}
		impart.WriteError(w, err)
		return
	}
	log.Printf("Error: %v", err)

	impart.WriteError(w, impart.HTTPError{http.StatusInternalServerError, "We encountered an error we couldn't handle."})
}
