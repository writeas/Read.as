package readas

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/gorilla/sessions"
	"github.com/writeas/web-core/auth"
	"github.com/writeas/web-core/converter"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

const (
	serverName      = "Read.as"
	softwareVersion = "0.2"
	configFile      = "config.json"
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
	Host         string `json:"host"`
	Port         int    `json:"port"`
	MySQLConnStr string `json:"mysql_connection"`

	// Instance
	Name string `json:"instance_name"`
}

func Serve() {
	app := &app{
		cfg: &config{},
	}

	var newUser, newPass string
	flag.IntVar(&app.cfg.Port, "p", 8080, "Port to start server on")
	flag.StringVar(&app.cfg.Host, "h", "", "Site's base URL")

	// options for creating a new user
	flag.StringVar(&newUser, "user", "", "New user's username. Should be paired with --pass")
	flag.StringVar(&newPass, "pass", "", "Password for new user. Should be paired with --user")
	flag.Parse()

	if app.cfg.Host == "" || os.Getenv("RA_MYSQL_CONNECTION") == "" {
		log.Printf("Reading %s", configFile)
		// Read configuration if information not passed in via flags or environment vars
		f, err := ioutil.ReadFile(configFile)
		if err != nil {
			log.Fatal("File error: %v\n", err)
		}

		err = json.Unmarshal(f, &app.cfg)
		if err != nil {
			log.Fatalf("Unable to read user config: %v", err)
		}
	}

	userAgent = "Go (" + serverName + "/" + softwareVersion + "; +" + app.cfg.Host + ")"

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
	logInfo("Serving on localhost:%d", app.cfg.Port)
	http.ListenAndServe(fmt.Sprintf(":%d", app.cfg.Port), nil)
}

func initConverter() {
	formDecoder := schema.NewDecoder()
	formDecoder.RegisterConverter(sql.NullString{}, converter.SQLNullString)
}
