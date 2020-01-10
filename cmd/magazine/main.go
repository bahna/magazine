package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path"
	"time"

	"github.com/bahna/magazine/appkit"
	"github.com/bahna/magazine/appkit/middleware"
	"github.com/bahna/magazine/cms/user"
	"github.com/gorilla/securecookie"
	"github.com/nicksnyder/go-i18n/i18n"
)

// a set of environment variable names, those variables must be
// provided during the startup time by the caller.
const (
	hashKeyEnv  = "BAHNA_HASH_KEY"
	blockKeyEnv = "BAHNA_BLOCK_KEY"
	secretEnv   = "BAHNA_SECRET"
)

// debug specifies if the program is running in the debug mode.
var debug = false

func main() {
	// cli flags
	addr := flag.String("addr", ":8080", "address to listen on")
	dbhost := flag.String("dbhost", "0.0.0.0", "database host")
	dbname := flag.String("dbname", "magazine", "database name")
	timeout := flag.String("timeout", "10m", "server's timeout")
	logpath := flag.String("log", "~/tmp/log/magazine", "log file path")
	assets := flag.String("assets", "assets/", "assets folder which contains templates/, static/, files/ folders")
	globalAssets := flag.String("gassets", "assets/", "global assets folder")
	debugflag := flag.Bool("debug", false, "debug mode")
	// tmpldir := flag.String("tmpl", "assets/templates/", "assets folder")
	// static := flag.String("static", "assets/static/", "static folder")
	// files := flag.String("files", "assets/files/", "user uploaded files folder")
	flag.Parse()

	debug = *debugflag

	// loading UI translations during the package initialization
	i18n.MustLoadTranslationFile(path.Join(*globalAssets, "i18n/en-us.all.json"))
	i18n.MustLoadTranslationFile(path.Join(*globalAssets, "i18n/ru.all.json"))
	i18n.MustLoadTranslationFile(path.Join(*globalAssets, "i18n/be.all.json"))

	// read environment variables
	hashKey := appkit.MustGetEnv(hashKeyEnv)
	if len(hashKey) != 32 {
		log.Fatalf("%s length must be 32 bytes, got %d", hashKeyEnv, len(hashKey))
	}
	blockKey := appkit.MustGetEnv(blockKeyEnv)
	if len(blockKey) != 32 {
		log.Fatalf("%s length must be 32 bytes, got %d", blockKeyEnv, len(blockKey))
	}
	secret := appkit.MustGetEnv(secretEnv)

	// app setup
	scookie := securecookie.New(hashKey, blockKey)
	cfg := appkit.Configuration{
		Scookie:          scookie,
		ScookieDuration:  time.Hour * 24 * 28 * 3,
		Secret:           secret,
		DbHost:           *dbhost,
		DbName:           *dbname,
		TmplDir:          path.Join(*assets, "templates/"),
		StaticDir:        path.Join(*assets, "static/"),
		FilesDir:         path.Join(*assets, "files/"),
		MaxAge:           "172800",
		MaxUploadSize:    100 * 1024 * 1024,
		MailchimpListURI: "https://us14.api.mailchimp.com/3.0/lists/6b4f8d648f/members",
		MailchimpAPI:     "4c7e261c3764067063cce7967b36f498-us14",
		AdminGroup: []user.Role{
			user.Administrator,
			user.Author,
		},
	}
	app, err := appkit.New(&cfg, ensureIndexes, makeRouter, generateTmpls, generateTmplFuncs)
	if err != nil {
		log.Fatal(err)
	}
	// free resources during on exit
	defer app.Stop()

	// logger setup
	if w, f, err := appkit.LogWriters(*logpath); err != nil {
		log.Fatal(err)
	} else {
		log.SetOutput(w)
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
		defer f.Close()
	}

	// run
	t, err := time.ParseDuration(*timeout)
	if err != nil {
		log.Fatal(err)
	}

	errors := make(chan error, 1)
	done1 := make(chan struct{})
	cancel := make(chan os.Signal, 1)

	r := middleware.Recover(
		middleware.Authenticate(
			middleware.Log(app.Router),
			scookie))

	go appkit.Run("magazine", *addr, r, t, errors, done1)
	signal.Notify(cancel, os.Interrupt, os.Kill)

	// control
	sig := <-cancel
	log.Printf("%v signal, shutting down", sig)
	close(done1)
	for i := 0; i < cap(errors); i++ {
		err = <-errors
		if err != nil {
			log.Println("shutdown with the error:", err)
		}
	}
}
