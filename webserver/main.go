package main

import (
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Machiel/slugify"
	"github.com/bahna/magazine/webserver/mongo"
	"github.com/bahna/magazine/webserver/slugifier"
	"github.com/bahna/magazine/webserver/user"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/gorilla/securecookie"
	"github.com/nicksnyder/go-i18n/i18n"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
	mgo "gopkg.in/mgo.v2"
)

// a set of environment variable names, those variables must be provided during the startup time by the caller.
const (
	hashKeyEnv  = "BAHNA_HASH_KEY"
	blockKeyEnv = "BAHNA_BLOCK_KEY"
	secretEnv   = "BAHNA_SECRET"
)

// debug specifies if the program is running in the debug mode.
var debug = false

var (
	ErrDependentContentExist = errors.New("delete dependent content first")
)

func main() {
	// cli flags
	addr := flag.String("addr", ":8080", "address to listen on")
	dbhost := flag.String("dbhost", "0.0.0.0", "database host")
	dbname := flag.String("dbname", "magazine", "database name")
	timeout := flag.String("timeout", "10m", "server's timeout")
	logpath := flag.String("log", "~/tmp/log/magazine", "log file path")
	assets := flag.String("assets", "assets/", "assets folder which contains templates/, static/, files/ folders")
	globalAssets := flag.String("gassets", "i18n/", "global assets folder")
	debugflag := flag.Bool("debug", false, "debug mode")
	flag.Parse()

	debug = *debugflag

	// loading UI translations during the package initialization
	i18n.MustLoadTranslationFile(path.Join(*globalAssets, "en-us.all.json"))
	i18n.MustLoadTranslationFile(path.Join(*globalAssets, "ru.all.json"))
	i18n.MustLoadTranslationFile(path.Join(*globalAssets, "be.all.json"))

	// read environment variables
	hashKey := MustGetEnv(hashKeyEnv)
	if len(hashKey) != 32 {
		log.Fatalf("%s length must be 32 bytes, got %d", hashKeyEnv, len(hashKey))
	}
	blockKey := MustGetEnv(blockKeyEnv)
	if len(blockKey) != 32 {
		log.Fatalf("%s length must be 32 bytes, got %d", blockKeyEnv, len(blockKey))
	}
	secret := MustGetEnv(secretEnv)

	// app setup
	scookie := securecookie.New(hashKey, blockKey)
	cfg := configuration{
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
		MailchimpAPI:     "4c7e261c3764067063cce7967b36f498-us14", // TODO: hide this from public and clean the history
		AdminGroup: []user.Role{
			user.Administrator,
			user.Author,
		},
	}

	// app initialization
	app, err := newApplication(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	// middleware
	r := Recover(Authenticate(Log(app.Router), scookie))

	// logger setup
	if w, f, err := LogWriters(*logpath); err != nil {
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

	log.Printf("serving %s at %s", "magazine", *addr)
	s := &http.Server{
		Addr:         *addr,
		Handler:      r,
		ReadTimeout:  t,
		WriteTimeout: t,
	}
	log.Fatal(s.ListenAndServe())
}

type configuration struct {
	Scookie         *securecookie.SecureCookie
	ScookieDuration time.Duration
	// Secret for passwords hashing.
	Secret []byte
	// DbHost and DbName are used to create a database connection.
	DbHost, DbName string
	// StaticDir, FilesDir, TmplDir are pathes for static and user files.
	StaticDir, FilesDir, TmplDir string
	// MaxAge is age of static files cache-control max-age value in seconds.
	MaxAge string
	// MaxUploadSize specifies the maximum size of user files.
	MaxUploadSize int64
	// MailchimpListURI is an URI to register new subscribers.
	MailchimpListURI string
	// MailchimpAPI is an API key.
	MailchimpAPI string
	// AdminGroup unites roles with an access to administration resources.
	AdminGroup []user.Role

	Name, Addr string
	// Timeout is read and write server's timeouts.
	Timeout time.Duration
}

// application shares resources between handlers and stores the global state of the web application.
type application struct {
	Config      *configuration
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

func newApplication(cfg *configuration) (app *application, err error) {
	s, err := mgo.Dial(cfg.DbHost)
	if err != nil {
		return app, fmt.Errorf("failed to dial the database %s: %v", cfg.DbHost, err)
	}

	if err = ensureIndexes(s, cfg.DbName); err != nil {
		return app, fmt.Errorf("failed to create database indexes: %v", err)
	}

	langs := []language.Tag{
		language.English, // first language is used as a fallback
		language.MustParse("be"),
		language.Russian,
	}

	app = &application{
		Config:         cfg,
		Db:             s.DB(cfg.DbName),
		Langs:          langs,
		LangMatcher:    language.NewMatcher(langs),
		LangNamer:      display.English.Languages(),
		FormDecoder:    schema.NewDecoder(),
		Transliterator: slugifier.NewSlugifier(),
	}

	funcs := generateTmplFuncs(app)
	tmpls := generateTmpls(cfg.TmplDir, funcs)

	app.Funcs = funcs
	app.Templates = tmpls

	app.Router = makeRouter(app)

	return
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

func LoginUser(app *application, r *http.Request) (u *user.User, err error) {
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

func Check(err error) {
	if err != nil {
		panic(err)
	}
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

// Page contains all data needed to render a page to a user. Fields
// are exported to be able to use in templates.
type Page struct {
	CurrentUser *user.User
	Language    language.Tag
	// Data is container of variable information collected by a
	// handler and passed to a template.
	Data interface{}
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

// LatestTime returns the latest timestamp from all given timestamps.
func LatestTime(tt ...time.Time) (t time.Time) {
	sort.Sort(ByTime(tt))
	if len(tt) > 0 {
		t = tt[0]
	}
	return
}

// MustGetEnv tries to get the environment variable and fails if it cannot.
func MustGetEnv(v string) []byte {
	s := os.Getenv(v)
	if len(s) == 0 {
		log.Fatal(v, " must be specified")
	}
	return []byte(s)
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

// ByTime sorts a slice of timestamps.
type ByTime []time.Time

func (t ByTime) Len() int           { return len(t) }
func (t ByTime) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t ByTime) Less(i, j int) bool { return t[i].After(t[j]) }
