// Package middleware provides handler wrappers for different general purposes.
package middleware

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/bahna/magazine/appkit"
	"github.com/bahna/magazine/mail"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	mgo "gopkg.in/mgo.v2"
)

// TODO: fast solution, replace the code below with something more configurable
var mailcfg = mail.Config{
	Host:     "localhost",
	Port:     25,
	Insecure: true,
}
var (
	mailErrSubj = "[bahna][error] "
	mailErrFrom = "notify@bahna.ngo"
	mailErrTo   = []string{"ihar.suvorau@gmail.com"}
)

var errTmpl *template.Template

func init() {
	errTmpl = template.Must(template.New("").Parse(errTmplHTML))
}

// AuthorizeAdminsMiddleware uses global variables adminGroup and app to authorize users.
func AuthorizeAdminsMiddleware(app *appkit.Application) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if u, _ := appkit.LoginUser(app, r); u != nil {
				for _, v := range u.Roles {
					for _, rl := range app.Config.AdminGroup {
						if v == rl { // at least one role is satisfied
							next.ServeHTTP(w, r)
							return
						}
					}
				}
			}
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		})
	}
}

func CurrentUserMiddleware(app *appkit.Application) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, err := appkit.LoginUser(app, r)
			if err != nil {
				log.Println(err)
			} else {
				app.CurrentUser = u
			}
			next.ServeHTTP(w, r)
		})
	}
}

func Log(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer func() {
			log.Println(r.Method, r.URL.String(), time.Since(start))
		}()
		next.ServeHTTP(w, r)
	})
}

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e, ok := recover().(error); ok {
				// determine the status code
				code := responseStatusFromErr(e)
				w.WriteHeader(code)

				// error threshold for notification
				if code > 404 {
					// log
					log.Println(e)
					log.Printf("%s\n", debug.Stack())
					// send error message
					subj := mailErrSubj + e.Error()
					if err := mail.SendErrorInsecure(mailcfg, e, r, http.StatusInternalServerError, mailErrFrom, subj, mailErrTo); err != nil {
						log.Println("mail.SendErrorInsecure failed:", err)
					}
				}

				// render an error page
				if err := errTmpl.Execute(w, e.Error()); err != nil {
					log.Printf("failed to execute the error template: %v", err)
				}
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func RedirectTrailingSlash(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && strings.HasSuffix(r.URL.Path, "/") && !strings.HasPrefix(r.URL.Path, "/static") {
			http.Redirect(w, r, strings.TrimRight(r.URL.Path, "/"), http.StatusMovedPermanently)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func CheckAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/static") {
			username, password, ok := r.BasicAuth()
			if !ok || username != "bahna-friend" || password != "let-me-see" {
				w.Header().Set("WWW-Authenticate", fmt.Sprintf("Basic realm=%q", "/"))
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func Authenticate(next http.Handler, sc *securecookie.SecureCookie) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const name = "auth"
		if c, err := r.Cookie(name); err == nil {
			v := make(map[string]string)
			if err = sc.Decode(name, c.Value, &v); err == nil {
				r = r.WithContext(context.WithValue(r.Context(), "uid", v["id"]))
				r = r.WithContext(context.WithValue(r.Context(), "email", v["email"]))
			}
		}
		next.ServeHTTP(w, r)
	})
}

func SetRUCookie(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := r.Cookie("lang")
		// set a cookie once for a new user
		if err == http.ErrNoCookie {
			http.SetCookie(w, &http.Cookie{
				Name:  "lang",
				Value: "ru",
			})
			log.Println("lang cookie set")
		}

		next.ServeHTTP(w, r)
	})
}

func responseStatusFromErr(err error) int {
	if err == mgo.ErrNotFound {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}

var errTmplHTML = `<html>
<head>
<link href="/static/basscss.min.css" rel="stylesheet">
</head>
<body class="flex" style="height: 100%;">
<main class="flex flex-column flex-auto justify-center items-center">
<section class="flex justify-center items-baseline">
<h1 class="m0 regular mr2"><span class="border-bottom">Ошибка</span>:</h1>
<h2 class="m0">{{ . }}</h2>
</section>
<p>
Попробуйте сначала &mdash; <a href="/">на главную</a>
</p>
</main>
</body></html>`
