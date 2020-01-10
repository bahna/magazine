package main

import (
	"net/http"
	"path"

	"github.com/bahna/magazine/appkit"
	"github.com/bahna/magazine/appkit/handlers"
	"github.com/bahna/magazine/appkit/middleware"
	"github.com/gorilla/mux"
)

func makeRouter(a *appkit.Application) *mux.Router {
	r := mux.NewRouter()
	r.StrictSlash(true)

	// lang handler is a parent to admin and user handlers
	withLang := r.PathPrefix("/{lang:en|ru|be}").Subrouter()

	// admin handlers
	admin := withLang.PathPrefix("/admin").Subrouter()
	admin.Use(middleware.AuthorizeAdminsMiddleware(a))
	admin.Use(middleware.CurrentUserMiddleware(a))
	admin.Handle("/{colname}/delete/{id}", adminDeleteHandler(a)).Methods("GET") // general delete
	admin.Handle("/topics/edit/{id}", adminEditTopicHandler(a)).Methods("GET")
	admin.Handle("/topics/new", adminNewTopicHandler(a)).Methods("GET")
	admin.Handle("/topics/", adminListTopicsHandler(a)).Methods("GET").Name("topics")
	admin.Handle("/topics/", adminSaveTopicHandler(a)).Methods("POST")
	admin.Handle("/content/filter", adminFilterContentHandler(a)).Methods("GET", "POST")
	admin.Handle("/content/edit/{id}", adminEditContentHandler(a)).Methods("GET", "POST")
	admin.Handle("/content/new", adminNewContentHandler(a)).Methods("GET")
	admin.Handle("/content/", adminCreateContentHandler(a)).Methods("POST")
	admin.Handle("/content/", adminListContentHandler(a)).Methods("GET", "POST").Name("content")
	admin.Handle("/users/passchange/{id}", adminUserPassChangeHandler(a)).Methods("GET", "POST")
	admin.Handle("/users/edit/{id}", adminEditUserHandler(a)).Methods("GET", "POST")
	admin.Handle("/users/new", adminNewUserHandler(a)).Methods("GET")
	admin.Handle("/users/", adminUsersHandler(a)).Methods("GET").Name("users")
	admin.Handle("/users/", adminCreateUserHandler(a)).Methods("POST")
	admin.Handle("/files/delete_/{id}", adminDeleteFileHandler(a)).Methods("GET")
	admin.Handle("/files/edit/{id}", adminEditFileHandler(a)).Methods("GET", "POST")
	admin.Handle("/files/", adminFilesHandler(a)).Methods("GET").Name("files")
	admin.Handle("/files/", adminCreateFileHandler(a)).Methods("POST")
	admin.Handle("/", adminIndexHandler(a)).Methods("GET").Name("adminIndex")

	// user handlers
	withLang.Handle("/signup", signupHandler(a)).Methods("GET", "POST")
	withLang.Handle("/login", loginHandler(a)).Methods("GET").Name("login")
	withLang.Handle("/login", loginHandler(a)).Methods("POST")
	withLang.Handle("/logout", logoutHandler(a)).Methods("GET")
	withLang.Handle("/restore", restoreUserAccessHandler(a)).Methods("GET", "POST")
	withLang.Handle("/mailchimp", mailchimpHandler(a))
	withLang.Handle("/search", searchHandler(a))
	withLang.Handle("/{topic}/{content}", contentHandler(a)).Methods("GET")
	withLang.Handle("/{topic}", topicHandler(a)).Methods("GET")
	withLang.Handle("/", indexHandler(a)).Name("index")

	// static files
	r.Handle("/static/{key:.*}", handlers.StaticFolder(a.Config.StaticDir, a.Config.MaxAge)).Methods("GET")
	r.Handle("/files/{key:.*}", handlers.StaticFolderDebug(
		a.Config.FilesDir, a.Config.MaxAge, debug, "https://bahna.land/files/")).Methods("GET")
	r.Handle("/sitemap.xml", handlers.ServeFile(path.Join(a.Config.StaticDir, "sitemap.xml"), "application/xml"))
	r.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(robotsTxt))
	})
	for name, contentType := range favicons {
		r.Handle("/"+name, handlers.ServeFile(path.Join(a.Config.StaticDir, name), contentType))
	}

	// index catchall handler
	r.Handle("/", rootHandler(a)).Name("root")

	return r
}

// favicons generated with https://realfavicongenerator.net.
var favicons = map[string]string{
	"apple-touch-icon.png":       "image/png",
	"favicon-32x32.png":          "image/png",
	"favicon-16x16.png":          "image/png",
	"site.webmanifest":           "application/json",
	"safari-pinned-tab.svg":      "image/svg+xml",
	"mstile-150x150.png":         "image/png",
	"favicon.ico":                "image/x-icon",
	"browserconfig.xml":          "image/png",
	"android-chrome-256x256.png": "image/png",
	"android-chrome-192x192.png": "image/png",
	"og-image.png":               "image/png",
}

// robotsTxt is served at /robots.txt endpoint.
var robotsTxt = `User-agent: *
Disallow: /*/admin
Disallow: /*/login
Disallow: /*/restore

User-agent: Yandex
Disallow: /*/admin
Disallow: /*/login
Disallow: /*/restore
`
