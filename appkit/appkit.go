// appkit provides common interface for configuring, starting and
// maintaining a website.
package appkit

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/iharsuvorau/mongo"
	"bitbucket.org/iharsuvorau/slugifier"
	"github.com/Machiel/slugify"
	"github.com/bahna/magazine/cms"
	"github.com/bahna/magazine/cms/user"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/gorilla/securecookie"
	"github.com/nicksnyder/go-i18n/i18n"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"

	mgo "gopkg.in/mgo.v2"
)

// application-wide errors
var (
	ErrDependentContentExist = errors.New("delete dependent content first")
)

// Configuration is a set of resources provided by the caller needed
// to create an application.
type Configuration struct {
	Scookie         *securecookie.SecureCookie
	ScookieDuration time.Duration
	// Secret for passwords hashing.
	Secret []byte
	// DbHost and DbName are used to create a database connection.
	DbHost, DbName string
	// StaticDir, FilesDir, TmplDir are pathes for static and user files.
	StaticDir, FilesDir, TmplDir string
	// MaxAge is age of static files cache-control max-age value
	// in seconds.
	MaxAge string
	// MaxUploadSize specifies the maximum size of user files.
	MaxUploadSize int64
	// MailchimpListURI is an URI to register new subscribers.
	MailchimpListURI string
	// MailchimpAPI is an API key.
	MailchimpAPI string
	// AdminGroup unites roles with an access to administration
	// resources.
	AdminGroup []user.Role

	Name, Addr string
	// Timeout is read and write server's timeouts.
	Timeout time.Duration
}

// Application shares resources between handlers and stores the global
// state of the web application.
type Application struct {
	Config      *Configuration
	Db          *mgo.Database
	Templates   map[string]*template.Template
	Funcs       template.FuncMap
	Langs       []language.Tag
	LangMatcher language.Matcher
	LangNamer   display.Namer
	FormDecoder *schema.Decoder
	Router      *mux.Router
	CurrentUser *user.User
	// transliterator manages slugs generation from titles.
	Transliterator *slugify.Slugifier
}

// Close should free all resources before application shutdown.
func (a *Application) Stop() error {
	a.Db.Session.Close()
	return nil
}

func (a *Application) Run() error {
	log.Printf("serving %s at %s", a.Config.Name, a.Config.Addr)
	s := &http.Server{
		Addr:         a.Config.Addr,
		Handler:      a.Router,
		ReadTimeout:  a.Config.Timeout,
		WriteTimeout: a.Config.Timeout,
	}
	return s.ListenAndServe()
}

// Page contains all data needed to render a page to a user. Fields
// are exported to be able to use in templates.
type Page struct {
	CurrentUser *user.User
	Language    language.Tag
	// Data is container of variable information collected by a
	// handler and passed to a template.
	Data interface{}
}

// CreateMongoIndexes creates database indexes during the startup.
type CreateMongoIndexes func(*mgo.Session, string) error

// CreateRouter creates the root router for the application which
// handles all endpoints. Each application provides its own routes, so
// it's application dependent.
type CreateRouter func(*Application) *mux.Router

// CreateTemplates creates a map of templates for a website.
type CreateTemplates func(tmplDir string, fm template.FuncMap) map[string]*template.Template

// CreateFuncMap creates template helpers to use inside
// templates. Application is provided for optional usage by helper
// functions.
type CreateFuncMap func(*Application) template.FuncMap

func New(cfg *Configuration, makeIndexes CreateMongoIndexes, makeRouter CreateRouter, makeTmpls CreateTemplates, makeFM CreateFuncMap) (app *Application, err error) {
	s, err := mgo.Dial(cfg.DbHost)
	if err != nil {
		return app, fmt.Errorf("failed to dial the database %s: %v", cfg.DbHost, err)
	}
	if err = makeIndexes(s, cfg.DbName); err != nil {
		return app, fmt.Errorf("failed to create database indexes: %v", err)
	}

	funcs := makeFM(app)
	tmpls := makeTmpls(cfg.TmplDir, funcs)

	langs := []language.Tag{
		language.English, // first language is used as a fallback
		language.MustParse("be"),
		language.Russian,
	}

	app = &Application{
		Config:         cfg,
		Db:             s.DB(cfg.DbName),
		Funcs:          funcs,
		Templates:      tmpls,
		Langs:          langs,
		LangMatcher:    language.NewMatcher(langs),
		LangNamer:      display.English.Languages(),
		FormDecoder:    schema.NewDecoder(),
		Transliterator: slugifier.NewSlugifier(),
	}

	app.Router = makeRouter(app)

	return
}

func LoginUser(app *Application, r *http.Request) (u *user.User, err error) {
	id, ok := r.Context().Value("uid").(string)
	if !ok {
		return nil, nil
	}
	u = new(user.User)
	if err = mongo.GetID(app.Db.C("users"), id, u); err != nil {
		err = fmt.Errorf("cannot log in user: id: %s error: %v", id, err)
		return
	}
	return
}

// CalculateEtag produces a strong etag by default, although, for
// efficiency reasons, it does not actually consume the contents of
// the file to make a hash of all the bytes. ¯\_(ツ)_/¯ Prefix the
// etag with "W/" to convert it into a weak etag.  See:
// https://tools.ietf.org/html/rfc7232#section-2.3
func CalculateEtag(d os.FileInfo) string {
	t := strconv.FormatInt(d.ModTime().Unix(), 36)
	s := strconv.FormatInt(d.Size(), 36)
	return `"` + t + s + `"`
}

// PubTime defines the publication date of content.
func PubTime(c *cms.Content) (t time.Time) {
	if c.Created.After(c.Updated) {
		t = c.Updated
	} else {
		t = c.Created
	}

	if c.Scheduled != (time.Time{}) {
		if t.After(c.Scheduled) {
			t = c.Scheduled
		}
	}

	return
}

// ByTime sorts a slice of timestamps.
type ByTime []time.Time

func (t ByTime) Len() int           { return len(t) }
func (t ByTime) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t ByTime) Less(i, j int) bool { return t[i].After(t[j]) }

// LatestTime returns the latest timestamp from all given timestamps.
func LatestTime(tt ...time.Time) (t time.Time) {
	sort.Sort(ByTime(tt))
	if len(tt) > 0 {
		t = tt[0]
	}
	return
}

// LangMust must return a language.Tag in any case. It collects information about
// preferred languages from cookies, HTTP headers and a route (e.g: /en/page).
func LangMust(matcher language.Matcher, routeLang string, r *http.Request) language.Tag {
	langs := []string{routeLang}
	userPrefs := UserLangs(r)
	langs = append(langs, userPrefs...)
	t, _ := language.MatchStrings(matcher, langs...)
	return t
}

// LogWriters takes a filepath, modifies it appending current time, creates
// a file and returns a multiwriter to the file and stdout with a file
// for closing later and an error.
func LogWriters(path string) (io.Writer, *os.File, error) {
	t := time.Now()
	ext := filepath.Ext(path)
	path = strings.TrimSuffix(path, ext)
	path = fmt.Sprintf("%s_%s%s", path, t.Format("2006-01-02_15:04:05"), ext)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, f, err
	}
	return io.MultiWriter(os.Stdout, f), f, nil
}

// MustGetEnv tries to get the environment variable and fails if it cannot.
func MustGetEnv(v string) []byte {
	s := os.Getenv(v)
	if len(s) == 0 {
		log.Fatal(v, " must be specified")
	}
	return []byte(s)
}

// Run is a package-wide run (instead of Application.Run) which uses
// channels to communicate errors. It's the old way of running an
// application. Added for backward compatibility.
func Run(name, addr string, h http.Handler, timeout time.Duration, errors chan<- error, done chan struct{}) {
	log.Printf("serving %s at %s", name, addr)
	s := &http.Server{
		Addr:         addr,
		Handler:      h,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		errors <- s.Serve(ln)
	}()

	<-done
	errors <- s.Close()
	log.Printf("%s server is closed", name)
}

func RunAlt(n, a string, h http.Handler, t time.Duration) error {
	log.Printf("serving %s at %s", n, a)
	s := &http.Server{
		Addr:         a,
		Handler:      h,
		ReadTimeout:  t,
		WriteTimeout: t,
	}
	return s.ListenAndServe()
}

func Check(err error) {
	if err != nil {
		panic(err)
	}
}

func Render(tmpl *template.Template, lang language.Tag, w http.ResponseWriter, data interface{}) {
	T, err := i18n.Tfunc(lang.String())
	Check(err)
	tmpl = tmpl.Funcs(map[string]interface{}{
		"T": T,
	})

	if err := tmpl.Execute(w, data); err != nil {
		log.Println(err)
	}
}

func UserLangs(r *http.Request) []string {
	langs := []string{}
	if c, err := r.Cookie("lang"); err == nil {
		langs = append(langs, c.Value)
	}
	if s := r.Header.Get("Accept-Language"); len(s) > 0 {
		s = strings.Split(s, ";")[0]
		slice := strings.Split(s, ",")
		for _, v := range slice {
			langs = append(langs, v)
		}
	}
	//langs = append(langs, defaultLanguage) // default fallback language
	return langs
}

// Do not remember why these two pieces below exist but they look like
// important funcs. Leaving it for now.

func updateTFunc(tmpl *template.Template, r *http.Request) error {
	T, err := tfunc(r)
	if err != nil {
		return err
	}
	tmpl.Funcs(map[string]interface{}{"T": T})
	return nil
}

func tfunc(r *http.Request) (i18n.TranslateFunc, error) {
	langs := []string{}
	if c, err := r.Cookie("lang"); err == nil {
		langs = append(langs, c.Value)
	}
	if s := r.Header.Get("Accept-Language"); len(s) > 0 {
		s = strings.Split(s, ";")[0]
		slice := strings.Split(s, ",")
		langs = append(langs, slice...)
	}
	langs = append(langs, "en-US") // default language
	return i18n.Tfunc("", langs...)
}
