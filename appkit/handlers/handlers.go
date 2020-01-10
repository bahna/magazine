package handlers

import (
	"math/rand"
	"net/http"
	"os"
	"path"
	"strconv"

	"github.com/bahna/magazine/appkit"
	"github.com/gorilla/mux"
)

func ServeFile(path, contentType string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", contentType)
		http.ServeFile(w, r, path)
	})
}

func StaticFolder(staticDir, maxAge string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		k := vars["key"]

		// open the file
		filepath := path.Join(staticDir, k)
		f, err := os.Open(filepath)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			} else if os.IsPermission(err) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			// otherwise, maybe the server is under load and ran out of file descriptors?
			backoff := int(3 + rand.Int31()%3) // 3–5 seconds to prevent a stampede
			w.Header().Set("Retry-After", strconv.Itoa(backoff))
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
		defer f.Close()

		// get information about the file
		d, err := f.Stat()
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			} else if os.IsPermission(err) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			// return a different status code than above to distinguish these cases
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if d.IsDir() {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		etag := appkit.CalculateEtag(d)

		// TODO: guess content-type
		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "max-age="+maxAge)
		http.ServeFile(w, r, filepath)
	})
}

func StaticFolderDebug(staticDir, maxAge string, debug bool, remoteStaticDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		k := vars["key"]

		// for debug purposes use files uploaded to the
		// production server to avoid copying gigabytes b/w
		// machines
		if debug {
			http.Redirect(w, r, remoteStaticDir+k, 301)
			return
		}

		// open the file
		filepath := path.Join(staticDir, k)
		f, err := os.Open(filepath)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			} else if os.IsPermission(err) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			// otherwise, maybe the server is under load and ran out of file descriptors?
			backoff := int(3 + rand.Int31()%3) // 3–5 seconds to prevent a stampede
			w.Header().Set("Retry-After", strconv.Itoa(backoff))
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
		defer f.Close()

		// get information about the file
		d, err := f.Stat()
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			} else if os.IsPermission(err) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			// return a different status code than above to distinguish these cases
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if d.IsDir() {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		etag := appkit.CalculateEtag(d)

		// TODO: guess content-type
		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "max-age="+maxAge)
		http.ServeFile(w, r, filepath)
	})
}
