package readas

import (
	"database/sql"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/gorilla/sessions"
	"github.com/writeas/web-core/auth"
	"github.com/writeas/web-core/converter"
	"log"
	"net/http"
	"os"
)

const (
	serverName      = "Read.as"
	softwareVersion = "0.1"
)

var (
	logInfo   func(format string, v ...interface{})
	logError  func(format string, v ...interface{})
	userAgent string
)

type app struct {
	router *mux.Router
	db     *sql.DB
	cfg    *config
	keys   *keychain
	sStore *sessions.CookieStore
}

type config struct {
	host string
	port int
}

func Serve() {
	app := &app{
		cfg: &config{},
	}

	var newUser, newPass string
	flag.IntVar(&app.cfg.port, "p", 8080, "Port to start server on")
	flag.StringVar(&app.cfg.host, "h", "https://read.as", "Site's base URL")
	flag.StringVar(&newUser, "user", "", "New user's username. Should be paired with --pass")
	flag.StringVar(&newPass, "pass", "", "Password for new user. Should be paired with --user")
	flag.Parse()

	userAgent = "Go (" + serverName + "/" + softwareVersion + "; +" + app.cfg.host + ")"

	logInfo = log.New(os.Stdout, "", log.Ldate|log.Ltime).Printf
	logError = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile).Printf

	err := initDatabase(app)
	if err != nil {
		log.Fatal(err)
	}

	// Do any configuration
	if newUser != "" || newPass != "" {
		if newUser == "" {
			log.Fatal("missing --user parameter")
		} else if newPass == "" {
			log.Fatal("missing --pass parameter")
		}
		logInfo("Creating new user: %s", newUser)
		hashedPass, err := auth.HashPass([]byte(newPass))
		if err != nil {
			log.Fatalf("Unable to hash pass: %v", err)
		}
		app.createUser(&LocalUser{
			PreferredUsername: newUser,
			HashedPass:        hashedPass,
			Name:              newUser,
			Summary:           "It's just me right now.",
		})
		return
	}

	// Check if there's a user
	err = checkData(app)
	if err != nil {
		log.Fatal(err)
	}

	initFederation(app)
	err = initKeys(app)
	if err != nil {
		log.Fatal(err)
	}
	initSession(app)
	initRoutes(app)

	http.Handle("/", app.router)
	logInfo("Serving on localhost:%d", app.cfg.port)
	http.ListenAndServe(fmt.Sprintf(":%d", app.cfg.port), nil)
}

func initConverter() {
	formDecoder := schema.NewDecoder()
	formDecoder.RegisterConverter(sql.NullString{}, converter.SQLNullString)
}
