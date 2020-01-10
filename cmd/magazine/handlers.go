package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"bitbucket.org/iharsuvorau/mongo"
	"github.com/bahna/magazine/appkit"
	"github.com/bahna/magazine/cms"
	"github.com/bahna/magazine/cms/file"
	"github.com/bahna/magazine/cms/user"
	"github.com/bahna/magazine/mail"
	"github.com/gorilla/mux"
	"golang.org/x/text/language"
	"gopkg.in/mgo.v2/bson"
)

// topicWithAmount is a wrapper struct to extend cms.Topic type with
// Amount field.
type topicWithAmount struct {
	*cms.Topic
	Amount int
}

func adminDeleteHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)
		colname := vars["colname"]
		id := vars["id"]

		// TODO: add check for dependent items
		switch colname {
		case "users":
			// get user
			u := new(user.User)
			err := mongo.GetID(app.Db.C("users"), id, u)
			appkit.Check(err)
			// check dependent content
			items, err := cms.AllContentLimited(app.Db, bson.M{"authorids": u.ID}, 0)
			appkit.Check(err)
			if len(items) > 0 {
				err = appkit.ErrDependentContentExist
				appkit.Check(err)
				return
			}
		}

		err := mongo.Delete(app.Db.C(colname), id)
		appkit.Check(err)

		url, err := app.Router.Get(colname).URL("lang", lang.String())
		appkit.Check(err)
		http.Redirect(w, r, url.String(), http.StatusSeeOther)
	})
}

func adminIndexHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)

		page := appkit.Page{
			CurrentUser: app.CurrentUser,
			Language:    lang,
		}
		appkit.Render(app.Templates["admin/index"], lang, w, page)
	})
}

func adminListTopicsHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)

		app.Db.Session.Refresh()
		topics, err := cms.AllTopics(app.Db, nil)
		appkit.Check(err)

		tt := make([]topicWithAmount, len(topics))
		for i, v := range topics {
			n, err := app.Db.C("content").Find(bson.M{"topicids": v.ID}).Count()
			appkit.Check(err)
			tt[i] = topicWithAmount{
				Topic:  v,
				Amount: n,
			}
		}

		page := appkit.Page{
			CurrentUser: app.CurrentUser,
			Language:    lang,
			Data: struct {
				Topics             []topicWithAmount
				AvailableLanguages []language.Tag
			}{
				Topics:             tt,
				AvailableLanguages: app.Langs,
			},
		}

		appkit.Render(app.Templates["admin/topics/index"], lang, w, page)
	})
}

func adminNewTopicHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)
		page := appkit.Page{
			CurrentUser: app.CurrentUser,
			Language:    lang,
			Data: struct {
				AvailableLanguages []language.Tag
			}{
				AvailableLanguages: app.Langs,
			},
		}
		appkit.Render(app.Templates["admin/topics/new"], lang, w, page)
	})
}

func adminEditTopicHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)

		app.Db.Session.Refresh()
		t, err := cms.GetTopic(app.Db, vars["id"])
		appkit.Check(err)

		page := appkit.Page{
			CurrentUser: app.CurrentUser,
			Language:    lang,
			Data: struct {
				Topic              *cms.Topic
				AvailableLanguages []language.Tag
			}{
				Topic:              t,
				AvailableLanguages: app.Langs,
			},
		}
		appkit.Render(app.Templates["admin/topics/edit"], lang, w, page)
	})
}

func adminSaveTopicHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		appkit.Check(err)

		t := new(cms.Topic)
		err = app.FormDecoder.Decode(t, r.PostForm)
		appkit.Check(err)
		// be is unsupported by mongodb and causes language_override error
		if t.Language == "be" {
			t.LanguageOverride = "ru"
		}

		// new item doesn't have an ID
		if t.ID == bson.ObjectId("") {
			t.ID = bson.NewObjectId()
		}

		t.Slug = app.Transliterator.Slugify(t.Title)

		err = mongo.Save(app.Db.C("topics"), bson.M{"_id": t.ID}, t)
		appkit.Check(err)

		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)

		url, err := app.Router.Get("topics").URL("lang", lang.String())
		appkit.Check(err)

		http.Redirect(w, r, url.String(), http.StatusSeeOther)
	})
}

func adminListContentHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)

		err := r.ParseForm()
		appkit.Check(err)

		var t *cms.Topic
		var tt []*cms.Topic
		var currentType cms.ContentType

		if s := r.Form.Get("topic"); len(s) > 0 {
			t, err = cms.GetTopic(app.Db, s)
			appkit.Check(err)
		}

		if s := r.Form.Get("type"); len(s) > 0 {
			n, err := strconv.Atoi(s)
			appkit.Check(err)
			currentType = cms.ContentType(n)
		}

		app.Db.Session.Refresh()
		tt, err = cms.AllTopics(app.Db, nil)
		appkit.Check(err)

		// get content by topic if specified
		var q bson.M
		if t != nil {
			q = bson.M{"topicids": t.ID}
		}
		cc, err := cms.AllContentLimited(app.Db, q, 0)
		appkit.Check(err)

		page := appkit.Page{
			CurrentUser: app.CurrentUser,
			Language:    lang,
			Data: struct {
				CurrentTopic *cms.Topic
				CurrentType  cms.ContentType
				Topics       []*cms.Topic
				Content      []*cms.Content
				Types        []cms.ContentType
			}{
				CurrentTopic: t,
				CurrentType:  currentType,
				Topics:       tt,
				Content:      cc,
				Types:        cms.ContentTypes,
			},
		}
		appkit.Render(app.Templates["admin/content/index"], lang, w, page)
	})
}

func adminFilterContentHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)

		err := r.ParseForm()
		appkit.Check(err)

		var t *cms.Topic
		var tt []*cms.Topic
		var currentType *cms.ContentType

		if s := r.Form.Get("topic"); len(s) > 0 {
			t, err = cms.GetTopic(app.Db, s)
			appkit.Check(err)
		}

		if s := r.Form.Get("type"); len(s) > 0 {
			n, err := strconv.Atoi(s)
			appkit.Check(err)
			ct := cms.ContentType(n)
			currentType = &ct
		}

		app.Db.Session.Refresh()
		tt, err = cms.AllTopics(app.Db, nil)
		appkit.Check(err)

		var q interface{}

		and := []bson.M{}
		if t != nil {
			and = append(and, bson.M{"topicids": t.ID})
		}
		if currentType != nil {
			and = append(and, bson.M{"type": currentType})
		}

		if len(and) > 0 {
			q = bson.M{"$and": and}
		}

		cc, err := cms.AllContentLimited(app.Db, q, 0)
		appkit.Check(err)

		page := appkit.Page{
			CurrentUser: app.CurrentUser,
			Language:    lang,
			Data: struct {
				CurrentTopic *cms.Topic
				CurrentType  *cms.ContentType
				Topics       []*cms.Topic
				Content      []*cms.Content
				Types        []cms.ContentType
			}{
				CurrentTopic: t,
				CurrentType:  currentType,
				Topics:       tt,
				Content:      cc,
				Types:        cms.ContentTypes,
			},
		}
		appkit.Render(app.Templates["admin/content/index"], lang, w, page)
	})
}

func adminNewContentHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)

		app.Db.Session.Refresh()

		uu, err := cms.AllUsers(app.Db.C("users"), nil)
		appkit.Check(err)

		tt, err := cms.AllTopics(app.Db, nil)
		appkit.Check(err)

		series, err := cms.AllContent(app.Db, bson.M{
			"type":   cms.ArticleSeries,
			"public": true,
		})
		appkit.Check(err)

		page := appkit.Page{
			CurrentUser: app.CurrentUser,
			Language:    lang,
			Data: struct {
				Users              []*user.User
				Topics             []*cms.Topic
				AvailableLanguages []language.Tag
				ContentTypes       []cms.ContentType
				ContentParents     []*cms.Content
			}{
				Users:              uu,
				Topics:             tt,
				AvailableLanguages: app.Langs,
				ContentTypes:       cms.ContentTypes,
				ContentParents:     series,
			},
		}
		appkit.Render(app.Templates["admin/content/new"], lang, w, page)
	})
}

func adminEditContentHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)

		if r.Method == "GET" {
			c := new(cms.Content)
			err := mongo.GetID(app.Db.C("content"), vars["id"], c)

			uu, err := cms.AllUsers(app.Db.C("users"), nil)
			appkit.Check(err)

			tt, err := cms.AllTopics(app.Db, nil)
			appkit.Check(err)

			series, err := cms.AllContent(app.Db, bson.M{
				"type":   cms.ArticleSeries,
				"public": true,
			})
			appkit.Check(err)

			log.Printf("series: %+v, query: type %v lang %v", series, cms.ArticleSeries, lang.String())

			page := appkit.Page{
				CurrentUser: app.CurrentUser,
				Language:    lang,
				Data: struct {
					Content            *cms.Content
					Users              []*user.User
					Topics             []*cms.Topic
					AvailableLanguages []language.Tag
					ContentTypes       []cms.ContentType
					ContentParents     []*cms.Content
				}{
					Content:            c,
					Users:              uu,
					Topics:             tt,
					AvailableLanguages: app.Langs,
					ContentTypes:       cms.ContentTypes,
					ContentParents:     series,
				},
			}
			appkit.Render(app.Templates["admin/content/edit"], lang, w, page)
			return
		}

		// POST

		err := r.ParseForm()
		appkit.Check(err)

		// time layout is RFC3339
		// during time conversion faced a confusion like https://github.com/golang/go/issues/9346
		if s := r.PostFormValue("Scheduled"); len(s) != 0 {
			r.PostForm.Set("Scheduled", r.PostFormValue("Scheduled")+":00+03:00")
		}

		if s := r.PostFormValue("EventStart"); len(s) != 0 {
			r.PostForm.Set("EventStart", r.PostFormValue("EventStart")+":00+03:00")
		}

		r.PostForm.Set("Created", r.PostFormValue("Created")+":00+03:00")

		cf := new(contentForm)
		err = app.FormDecoder.Decode(cf, r.PostForm)
		appkit.Check(err)

		slug := app.Transliterator.Slugify(cf.Title)

		// var lede, body string
		// if body, err = typograf.Typogrify(cf.Body); err != nil {
		// 	log.Println(err)
		// 	body = cf.Body
		// }
		// if lede, err = typograf.Typogrify(cf.Lede); err != nil {
		// 	log.Println(err)
		// 	lede = cf.Lede
		// }

		updated := time.Now()
		pubtime := appkit.LatestTime(cf.Created, cf.Scheduled)

		pageTitle := cf.Title
		if len(cf.PageTitle) > 0 {
			pageTitle = cf.PageTitle
		}

		pageDescription := cf.Lede
		if len(cf.PageDescription) > 0 {
			pageDescription = cf.PageDescription
		}

		var parentID *bson.ObjectId
		if cf.ParentID.Valid() {
			parentID = cf.ParentID
		}

		cnt := map[string]interface{}{
			"weight":          cf.Weight,
			"public":          cf.Public,
			"type":            cf.Type,
			"promoted":        cf.Promoted,
			"language":        cf.Language,
			"scheduled":       cf.Scheduled,
			"updated":         updated,
			"published":       pubtime,
			"slug":            slug,
			"pageslug":        cf.PageSlug,
			"pagetitle":       pageTitle,
			"pagedescription": pageDescription,
			"parentid":        parentID,
			"authorids":       cf.AuthorIDs,
			"topicids":        cf.TopicIDs,
			"title":           cf.Title,
			"lede":            cf.Lede,
			"body":            cf.Body,
			"coverexternal":   cf.CoverExternal,
			"coverinternal":   cf.CoverInternal,
			"payload":         cf.Payload,
			"images":          cf.Images,
			"eventstart":      cf.EventStart,
			"location":        cf.Location,
			"linkto":          cf.LinkTo,
		}

		// be is unsupported by mongodb and causes language_override error
		if cf.Language == "be" {
			cnt["language_override"] = "ru"
		}

		c := new(cms.Content)
		err = mongo.GetID(app.Db.C("content"), vars["id"], c)
		appkit.Check(err)
		err = mongo.UpdateID(app.Db.C("content"), vars["id"], cnt, c)
		appkit.Check(err)

		//url, err := app.Router.Get("content").URL("lang", lang.String())
		//appkit.Check(err)
		http.Redirect(w, r, r.URL.String(), http.StatusSeeOther)
	})
}

func adminCreateContentHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		err := r.ParseForm()
		appkit.Check(err)

		if s := r.PostFormValue("Scheduled"); len(s) != 0 {
			// time layout is RFC3339
			// during time conversion faced a confusion like https://github.com/golang/go/issues/9346
			r.PostForm.Set("Scheduled", r.PostFormValue("Scheduled")+":00+03:00")
		}

		if s := r.PostFormValue("EventStart"); len(s) != 0 {
			r.PostForm.Set("EventStart", r.PostFormValue("EventStart")+":00+03:00")
		}

		// TODO: parse cover as image

		c := new(cms.Content)
		err = app.FormDecoder.Decode(c, r.PostForm)
		appkit.Check(err)
		// be is unsupported by mongodb and causes language_override error
		if c.Language == "be" {
			c.LanguageOverride = "ru"
		}

		c.ID = bson.NewObjectId()
		c.Created = time.Now()
		c.Slug = app.Transliterator.Slugify(c.Title)

		c.Published = appkit.LatestTime(c.Created, c.Scheduled)

		if len(c.PageTitle) == 0 {
			c.PageTitle = c.Title
		}

		if len(c.PageDescription) == 0 {
			c.PageDescription = c.Lede
		}

		if !c.ParentID.Valid() {
			c.ParentID = nil
		}

		err = mongo.Save(app.Db.C("content"), bson.M{"_id": c.ID}, c)
		appkit.Check(err)

		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)
		url, err := app.Router.Get("content").URL("lang", lang.String())
		appkit.Check(err)
		http.Redirect(w, r, url.String(), http.StatusSeeOther)
	})
}

func adminUsersHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)

		uu, err := cms.AllUsers(app.Db.C("users"), nil)
		appkit.Check(err)
		page := appkit.Page{
			CurrentUser: app.CurrentUser,
			Language:    lang,
			Data: struct {
				Users []*user.User
			}{
				Users: uu,
			},
		}
		appkit.Render(app.Templates["admin/users/index"], lang, w, page)
	})
}

func adminNewUserHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)

		page := appkit.Page{
			CurrentUser: app.CurrentUser,
			Language:    lang,
			Data: struct {
				Roles []user.Role
			}{
				Roles: user.Roles,
			},
		}
		appkit.Render(app.Templates["admin/users/new"], lang, w, page)
	})
}

func adminCreateUserHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)

		err := r.ParseForm()
		appkit.Check(err)

		uf := new(userForm)
		err = app.FormDecoder.Decode(uf, r.PostForm)
		appkit.Check(err)

		u, err := user.New(uf.Password, uf.Email, uf.FirstName, uf.LastName, uf.Roles, app.Config.Secret)
		appkit.Check(err)

		err = mongo.Save(app.Db.C("users"), bson.M{"_id": u.ID}, u)
		appkit.Check(err)

		url, err := app.Router.Get("users").URL("lang", lang.String())
		appkit.Check(err)
		http.Redirect(w, r, url.String(), http.StatusSeeOther)
	})
}

func adminEditUserHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)

		if r.Method == "GET" {
			u := new(user.User)
			err := mongo.GetID(app.Db.C("users"), vars["id"], u)
			appkit.Check(err)

			page := appkit.Page{
				CurrentUser: app.CurrentUser,
				Language:    lang,
				Data: struct {
					Roles []user.Role
					User  *user.User
				}{
					Roles: user.Roles,
					User:  u,
				},
			}
			appkit.Render(app.Templates["admin/users/edit"], lang, w, page)
			return
		}

		if r.Method == "POST" {
			err := r.ParseForm()
			appkit.Check(err)
			uf := new(userForm)
			err = app.FormDecoder.Decode(uf, r.PostForm)
			appkit.Check(err)

			if uf.Password != uf.PasswordConfirm {
				appkit.Check(user.ErrPasswordMatch)
			}

			u := new(user.User)
			err = mongo.GetID(app.Db.C("users"), uf.ID, u)
			err = mongo.UpdateID(app.Db.C("users"), uf.ID, map[string]interface{}{
				"email.address": uf.Email,
				"firstname":     uf.FirstName,
				"lastname":      uf.LastName,
				"roles":         uf.Roles,
			}, u)
			appkit.Check(err)

			url, err := app.Router.Get("users").URL("lang", lang.String())
			appkit.Check(err)
			http.Redirect(w, r, url.String(), http.StatusSeeOther)
		}
	})
}

func adminUserPassChangeHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)

		u := new(user.User)
		err := mongo.GetID(app.Db.C("users"), vars["id"], u)
		appkit.Check(err)

		if r.Method == "GET" {

			page := appkit.Page{
				CurrentUser: app.CurrentUser,
				Language:    lang,
				Data: struct {
					Roles []user.Role
					User  *user.User
				}{
					Roles: user.Roles,
					User:  u,
				},
			}
			appkit.Render(app.Templates["admin/users/passchange"], lang, w, page)
			return
		}

		if r.Method == "POST" {
			err := r.ParseForm()
			appkit.Check(err)
			uf := new(userChangePassForm)
			err = app.FormDecoder.Decode(uf, r.PostForm)
			appkit.Check(err)

			pass := r.Form.Get("Password")
			newPass := r.Form.Get("NewPassword")
			newPassConfirm := r.Form.Get("NewPasswordConfirm")

			if !user.Verify(pass, u.PasswordHash, app.Config.Secret) {
				appkit.Check(user.ErrWrongPassword)
			}

			if newPass != newPassConfirm {
				appkit.Check(user.ErrPasswordMatch)
			}

			newPassHash, err := user.MakeMAC([]byte(newPass), app.Config.Secret)
			appkit.Check(err)

			updatedUser := new(user.User)
			err = mongo.UpdateID(app.Db.C("users"), uf.ID, map[string]interface{}{
				"passwordhash": newPassHash,
			}, updatedUser)
			appkit.Check(err)

			url, err := app.Router.Get("adminIndex").URL("lang", lang.String())
			appkit.Check(err)
			http.Redirect(w, r, url.String(), http.StatusSeeOther)
		}
	})
}

func indexHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)

		u, err := appkit.LoginUser(app, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		q := r.URL.Query()
		var pageNo int
		if s := q.Get("p"); len(s) > 0 {
			pageNo, err = strconv.Atoi(s)
			appkit.Check(err)
		} else {
			pageNo = 1
		}
		findParams := bson.M{
			"language": lang.String(),
			"public":   true,
			"$and": []bson.M{
				bson.M{"$or": []bson.M{

					bson.M{"scheduled": bson.M{"$lt": time.Now()}},
					bson.M{"scheduled": (time.Time{})},
				}},
				bson.M{"$or": []bson.M{

					bson.M{"type": cms.Photoreport},
					bson.M{"type": cms.Article},
					bson.M{"type": cms.Banner},
				}},
			},
		}
		cc, prev, next, err := cms.AllContentByPage(app.Db.C("content"), findParams, 20, pageNo)
		appkit.Check(err)
		// filter different types of content
		mainThread := []*cms.Content{}

		for _, v := range cc {
			if v.Type == cms.Photoreport && v.ParentID == nil {
				mainThread = append(mainThread, v)
			} else if v.Type == cms.Article || v.Type == cms.Banner {
				mainThread = append(mainThread, v)
			}
		}

		topics, err := getTopics(app.Db, lang)
		appkit.Check(err)

		pages, err := getPages(app.Db, lang)
		appkit.Check(err)

		series, err := getSeries(app.Db, lang)
		appkit.Check(err)

		events, err := getEvents(app.Db, lang)
		appkit.Check(err)

		audio, err := getCertainContent(app.Db, lang, cms.Audio)
		appkit.Check(err)

		research, err := getCertainContent(app.Db, lang, cms.Research)
		appkit.Check(err)

		page := appkit.Page{
			Language:    lang,
			CurrentUser: u,
			Data: struct {
				AvailableLanguages                    []language.Tag
				Topics                                []*cms.Topic
				Topic                                 *cms.Topic
				MainThread                            []*cms.Content
				Events                                []*cms.Content
				Pages                                 []*cms.Content
				Audio                                 []*cms.Content
				Series                                []*cms.Content
				Research                              []*cms.Content
				CurrentPageNo, NextPageNo, PrevPageNo int
				SearchQuery                           string
				Debug                                 bool
			}{
				AvailableLanguages: app.Langs,
				Topics:             topics,
				MainThread:         mainThread,
				Events:             events,
				Pages:              pages,
				Audio:              audio,
				Series:             series,
				Research:           research,
				Topic:              nil,
				CurrentPageNo:      pageNo,
				NextPageNo:         next,
				PrevPageNo:         prev,
				Debug:              debug,
			},
		}
		appkit.Render(app.Templates["index"], lang, w, page)
	})
}

func searchHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)

		u, err := appkit.LoginUser(app, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		q := r.URL.Query()
		searchQuery := q.Get("q")
		if len(searchQuery) == 0 {
			http.Error(w, "empty search query", http.StatusBadRequest)
			return
		}

		cc, err := cms.SearchContent(app.Db, bson.M{
			"language": lang.String(),
			"$text": bson.M{
				"$search": searchQuery,
			},
		}, 0)
		appkit.Check(err)

		tt, err := getTopics(app.Db, lang)
		appkit.Check(err)

		// filter different types of content
		mainThread := []*cms.Content{}
		events := []*cms.Content{}
		audio := []*cms.Content{}
		pages := []*cms.Content{}

		for _, v := range cc {
			if v.Type == cms.Event {
				events = append(events, v)
			} else if v.Type == cms.Page {
				pages = append(pages, v)
			} else if v.Type == cms.Audio {
				audio = append(audio, v)
			} else {
				mainThread = append(mainThread, v)
			}
		}

		page := appkit.Page{
			Language:    lang,
			CurrentUser: u,
			Data: struct {
				AvailableLanguages                    []language.Tag
				Topics                                []*cms.Topic
				Topic                                 *cms.Topic
				MainThread                            []*cms.Content
				Events                                []*cms.Content
				Pages                                 []*cms.Content
				Audio                                 []*cms.Content
				CurrentPageNo, NextPageNo, PrevPageNo int
				SearchQuery                           string
			}{
				AvailableLanguages: app.Langs,
				Topics:             tt,
				MainThread:         mainThread,
				Pages:              pages,
				Audio:              audio,
				Events:             events,
				Topic:              nil,
				SearchQuery:        searchQuery,
			},
		}
		appkit.Render(app.Templates["index"], lang, w, page)
	})
}

func signupHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)
		url, err := app.Router.Get("index").URL("lang", lang.String())
		appkit.Check(err)

		if r.Method == "GET" {
			currentUser, err := appkit.LoginUser(app, r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// already logged in, no need in new user
			if currentUser != nil {
				http.Redirect(w, r, url.String(), http.StatusSeeOther)
				return
			}

			tt, err := cms.AllTopics(app.Db, bson.M{
				"language": lang.String(),
				"public":   true,
				"$or": []bson.M{
					bson.M{"page": false},
					bson.M{"page": nil},
				},
			})
			appkit.Check(err)

			pp, err := getPages(app.Db, lang)
			appkit.Check(err)

			page := appkit.Page{
				Language: lang,
				Data: struct {
					AvailableLanguages []language.Tag
					Topics             []*cms.Topic
					Pages              []*cms.Content
				}{
					AvailableLanguages: app.Langs,
					Topics:             tt,
					Pages:              pp,
				},
			}
			appkit.Render(app.Templates["signup"], lang, w, page)
			return
		}

		// POST
		err = r.ParseForm()
		appkit.Check(err)
		uf := new(userForm)
		err = app.FormDecoder.Decode(uf, r.PostForm)
		appkit.Check(err)

		pass := r.Form.Get("Password")
		passC := r.Form.Get("PasswordConfirm")
		if pass != passC {
			appkit.Check(user.ErrPasswordMatch)
		}

		uf.Roles = []user.Role{user.Visitor}

		u, err := user.New(pass, uf.Email, uf.FirstName, uf.LastName, uf.Roles, app.Config.Secret)
		appkit.Check(err)
		if !user.Validate(u) {
			appkit.Check(user.ErrNotValid)
		}
		err = mongo.Save(app.Db.C("users"), bson.M{"_id": u.ID}, u)
		appkit.Check(err)

		http.Redirect(w, r, url.String(), http.StatusSeeOther)
	})
}

func loginHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)
		url, err := app.Router.Get("index").URL("lang", lang.String())
		appkit.Check(err)

		if r.Method == "GET" {
			u, err := appkit.LoginUser(app, r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if u != nil {
				http.Redirect(w, r, url.String(), http.StatusSeeOther)
				return
			}

			tt, err := cms.AllTopics(app.Db, bson.M{
				"language": lang.String(),
				"public":   true,
				"$or": []bson.M{
					bson.M{"page": false},
					bson.M{"page": nil},
				},
			})
			appkit.Check(err)

			pp, err := getPages(app.Db, lang)
			appkit.Check(err)

			page := appkit.Page{
				Language: lang,
				Data: struct {
					AvailableLanguages []language.Tag
					Topics             []*cms.Topic
					Topic              *cms.Topic
					Pages              []*cms.Content
				}{
					AvailableLanguages: app.Langs,
					Topics:             tt,
					Topic:              nil,
					Pages:              pp,
				},
			}
			appkit.Render(app.Templates["login"], lang, w, page)
			return
		}

		// POST
		email := r.PostFormValue("Email.Address")
		pass := r.PostFormValue("Password")
		if len(email) == 0 || len(pass) == 0 {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		u := new(user.User)
		err = mongo.GetOne(app.Db.C("users"), bson.M{"email.address": email}, u)
		appkit.Check(err)

		if !user.Verify(pass, u.PasswordHash, app.Config.Secret) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		err = user.SetLoginCookie(w, u, app.Config.Scookie, app.Config.ScookieDuration)
		appkit.Check(err)

		// TODO: redirect to next value, implement next value with the HTML tmpl
		http.Redirect(w, r, url.String(), http.StatusSeeOther)
	})
}

func logoutHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)
		url, err := app.Router.Get("index").URL("lang", lang.String())
		appkit.Check(err)

		user.SetLogoutCookie(w)
		http.Redirect(w, r, url.String(), http.StatusSeeOther)
	})
}

func rootHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// workaround for "en-u-rg-uszzzz": https://github.com/golang/go/issues/24211
		//lang := appkit.LangMust(app.LangMatcher, "", r)
		//http.Redirect(w, r, "/"+lang.String()[:2]+"/", http.StatusSeeOther)

		// default
		http.Redirect(w, r, "/ru/", http.StatusSeeOther)
	})
}

func mailchimpHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)
		url, err := app.Router.Get("index").URL("lang", lang.String())
		appkit.Check(err)

		err = r.ParseForm()
		appkit.Check(err)

		email := r.URL.Query().Get("email")
		if len(email) == 0 {
			http.Redirect(w, r, url.String(), http.StatusSeeOther)
			return
		}

		// requesting mailhchimp

		payload := struct {
			Email    string `json:"email_address"`
			Status   string `json:"status"`
			Language string `json:"language"`
		}{
			Email:    email,
			Status:   "subscribed",
			Language: lang.String(),
		}

		var buf bytes.Buffer
		err = json.NewEncoder(&buf).Encode(payload)
		appkit.Check(err)
		req, err := http.NewRequest("POST", app.Config.MailchimpListURI, &buf)
		appkit.Check(err)
		req.SetBasicAuth("anyname", app.Config.MailchimpAPI)
		resp, err := http.DefaultClient.Do(req)
		appkit.Check(err)
		defer resp.Body.Close()

		// reading the response and notifying admins

		m := map[string]interface{}{}
		err = json.NewDecoder(resp.Body).Decode(&m)
		appkit.Check(err)

		err = mailchimpNotificationTmpl.Execute(&buf, m)
		appkit.Check(err)

		msg := mail.Message{
			From:    "notify@bahna.land",
			To:      []string{"bahna.land@gmail.com"},
			Subject: fmt.Sprintf("[bahna.land][new subscriber] %s", m["email_address"]),
			Body:    buf.String(),
			Created: time.Now(),
		}
		err = mail.Send(mail.DefaultConfig, msg)
		appkit.Check(err)

		// collecting info for a page

		u, err := appkit.LoginUser(app, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tt, err := getTopics(app.Db, lang)
		appkit.Check(err)

		pp, err := getPages(app.Db, lang)
		appkit.Check(err)

		page := appkit.Page{
			Language:    lang,
			CurrentUser: u,
			Data: struct {
				AvailableLanguages                    []language.Tag
				Topics                                []*cms.Topic
				Topic                                 *cms.Topic
				Content                               []*cms.Content
				Events                                []*cms.Content
				Pages                                 []*cms.Content
				CurrentPageNo, NextPageNo, PrevPageNo int
				SearchQuery                           string
			}{
				AvailableLanguages: app.Langs,
				Topics:             tt,
				// Content:            cc,
				// Events:             events,
				Pages: pp,
				Topic: nil,
				// CurrentPageNo:      pageNo,
				// NextPageNo:         next,
				// PrevPageNo:         prev,
			},
		}
		appkit.Render(app.Templates["subscription_done"], lang, w, page)
	})
}

func contentHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)
		s1 := vars["topic"]
		s2 := vars["content"]

		tt, err := getTopics(app.Db, lang)
		appkit.Check(err)

		var t cms.Topic
		err = mongo.GetOne(app.Db.C("topics"), bson.M{
			"language": lang.String(),
			"slug":     s1,
		}, &t)
		appkit.Check(err)

		c := new(cms.Content)
		err = mongo.GetOne(app.Db.C("content"), bson.M{"$and": []bson.M{
			{"slug": s2},
			{"topicids": t.ID},
			{"public": true},
			{"$or": []bson.M{
				bson.M{"scheduled": bson.M{"$lt": time.Now()}},
				bson.M{"scheduled": (time.Time{})},
			}},
		}}, c)
		appkit.Check(err)

		c.Topics = []*cms.Topic{&t}

		err = cms.GetAuthorsForContent(app.Db, c)
		appkit.Check(err)

		err = file.GetImagesForContent(app.Db, c)
		appkit.Check(err)

		err = cms.GetChildrenContent(app.Db, c)
		appkit.Check(err)

		u, err := appkit.LoginUser(app, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		pp, err := getPages(app.Db, lang)
		appkit.Check(err)

		page := appkit.Page{
			Language:    lang,
			CurrentUser: u,
			Data: struct {
				AvailableLanguages []language.Tag
				Topics             []*cms.Topic
				Content            *cms.Content
				Pages              []*cms.Content
				Topic              *cms.Topic
			}{
				AvailableLanguages: app.Langs,
				Topics:             tt,
				Topic:              &t,
				Content:            c,
				Pages:              pp,
			},
		}
		appkit.Render(app.Templates["material"], lang, w, page)
	})
}

func topicHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)
		s1 := vars["topic"]

		u, err := appkit.LoginUser(app, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		q := r.URL.Query()
		var pageNo int
		if s := q.Get("p"); len(s) > 0 {
			pageNo, err = strconv.Atoi(s)
			appkit.Check(err)
		} else {
			pageNo = 1
		}

		tt, err := getTopics(app.Db, lang)
		appkit.Check(err)

		t := new(cms.Topic)
		for _, v := range tt {
			if v.Slug == s1 {
				t = v
				break
			}
		}
		if !t.ID.Valid() {
			http.NotFound(w, r)
			return
		}

		findParams := bson.M{
			"language": lang.String(),
			"public":   true,
			"topicids": t.ID,
			"$and": []bson.M{
				bson.M{"$or": []bson.M{

					bson.M{"scheduled": bson.M{"$lt": time.Now()}},
					bson.M{"scheduled": (time.Time{})},
				}},
				bson.M{"$or": []bson.M{

					bson.M{"type": cms.Photoreport},
					bson.M{"type": cms.Article},
					bson.M{"type": cms.Banner},
				}},
			},
		}
		cc, prev, next, err := cms.AllContentByPage(app.Db.C("content"), findParams, 20, pageNo)
		appkit.Check(err)

		pages, err := getPages(app.Db, lang)
		appkit.Check(err)

		// filter different types of content
		mainThread := []*cms.Content{}

		for _, v := range cc {
			if v.Type == cms.Photoreport && v.ParentID == nil {
				mainThread = append(mainThread, v)
			} else if v.Type == cms.Article || v.Type == cms.Banner {
				log.Printf("adding: %s", v.Title)
				mainThread = append(mainThread, v)
			}
		}

		series, err := getSeries(app.Db, lang)
		appkit.Check(err)

		events, err := getEvents(app.Db, lang)
		appkit.Check(err)

		audio, err := getCertainContent(app.Db, lang, cms.Audio)
		appkit.Check(err)

		research, err := getCertainContent(app.Db, lang, cms.Research)
		appkit.Check(err)

		page := appkit.Page{
			Language:    lang,
			CurrentUser: u,
			Data: struct {
				AvailableLanguages                    []language.Tag
				Topics                                []*cms.Topic
				Topic                                 *cms.Topic
				MainThread                            []*cms.Content
				Events                                []*cms.Content
				Pages                                 []*cms.Content
				Audio                                 []*cms.Content
				Series                                []*cms.Content
				Research                              []*cms.Content
				CurrentPageNo, NextPageNo, PrevPageNo int
				SearchQuery                           string
			}{
				AvailableLanguages: app.Langs,
				Topics:             tt,
				Topic:              t,
				MainThread:         mainThread,
				Events:             events,
				Pages:              pages,
				Audio:              audio,
				Series:             series,
				Research:           research,
				CurrentPageNo:      pageNo,
				NextPageNo:         next,
				PrevPageNo:         prev,
			},
		}
		appkit.Render(app.Templates["index"], lang, w, page)
	})
}

func adminFilesHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)
		perpage := 100

		var err error

		// page numbers
		q := r.URL.Query()
		var pageNo int
		if s := q.Get("p"); len(s) > 0 {
			pageNo, err = strconv.Atoi(s)
			appkit.Check(err)
		} else {
			pageNo = 1
		}

		// quering
		files, prev, next, total, err := file.AllFilesByPage(app.Db.C("files"), nil, perpage, pageNo)
		appkit.Check(err)

		page := appkit.Page{
			CurrentUser: app.CurrentUser,
			Language:    lang,
			Data: struct {
				Files         []*file.File
				CurrentPageNo int
				NextPageNo    int
				PrevPageNo    int
				CurrentItems  int
				TotalItems    int
			}{
				Files:         files,
				CurrentPageNo: pageNo,
				NextPageNo:    next,
				PrevPageNo:    prev,
				TotalItems:    total,
				CurrentItems:  pageNo * perpage,
			},
		}
		appkit.Render(app.Templates["admin/files/list"], lang, w, page)
	})
}

func adminCreateFileHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)

		err := r.ParseMultipartForm(app.Config.MaxUploadSize)
		appkit.Check(err)

		var optimize bool

		if len(r.MultipartForm.Value["NeedOptimize"]) > 0 {
			optimize = true
		}
		err = file.UploadFromForm(app.Db.C("files"), r.MultipartForm.File["Files"][0],
			app.Config.FilesDir,
			r.MultipartForm.Value["Title"][0],
			r.MultipartForm.Value["Credits"][0],
			optimize)
		appkit.Check(err)

		url, err := app.Router.Get("files").URL("lang", lang.String())
		appkit.Check(err)
		http.Redirect(w, r, url.String(), http.StatusSeeOther)
	})
}

func adminDeleteFileHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)

		err := file.RemoveFile(app.Db.C("files"), vars["id"], app.Config.FilesDir)
		appkit.Check(err)

		url, err := app.Router.Get("files").URL("lang", lang.String())
		appkit.Check(err)
		http.Redirect(w, r, url.String(), http.StatusSeeOther)
	})
}

func adminEditFileHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)

		if r.Method == "GET" {
			f := new(file.File)
			err := mongo.GetID(app.Db.C("files"), vars["id"], f)
			appkit.Check(err)

			page := appkit.Page{
				CurrentUser: app.CurrentUser,
				Language:    lang,
				Data: struct {
					CurrentFile *file.File
				}{
					CurrentFile: f,
				},
			}
			appkit.Render(app.Templates["admin/files/edit"], lang, w, page)
			return
		}

		// POST

		f := new(file.File)
		col := app.Db.C("files")

		err := mongo.GetID(col, vars["id"], f)
		appkit.Check(err)
		err = mongo.UpdateID(col, vars["id"], map[string]interface{}{
			"title":   r.FormValue("Title"),
			"credits": r.FormValue("Credits"),
		}, f)
		appkit.Check(err)

		url, err := app.Router.Get("files").URL("lang", lang.String())
		appkit.Check(err)
		http.Redirect(w, r, url.String(), http.StatusSeeOther)
	})
}

func restoreUserAccessHandler(app *appkit.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		lang := appkit.LangMust(app.LangMatcher, vars["lang"], r)

		if r.Method == "GET" {
			// pages
			ccpp, err := getPages(app.Db, lang)
			appkit.Check(err)

			// topics
			tt, err := cms.AllTopics(app.Db, bson.M{
				"language": lang.String(),
				"public":   true,
				"$or": []bson.M{
					bson.M{"page": false},
					bson.M{"page": nil},
				},
			})
			appkit.Check(err)

			page := appkit.Page{
				Language: lang,
				Data: struct {
					AvailableLanguages []language.Tag
					Topic              *cms.Topic
					Topics             []*cms.Topic
					Pages              []*cms.Content
				}{
					AvailableLanguages: app.Langs,
					Topics:             tt,
					Pages:              ccpp,
				},
			}
			appkit.Render(app.Templates["restore_access"], lang, w, page)
		}

		if r.Method == "POST" {
			err := r.ParseForm()
			appkit.Check(err)

			email := r.Form.Get("Email.Address")

			u := new(user.User)
			err = mongo.GetOne(app.Db.C("users"), bson.M{"email.address": email}, u)
			appkit.Check(err)

			if u == nil {
				appkit.Check(user.ErrNotValid)
			}

			// generate and reset the password
			pass := user.RandStringRunes(9)
			appkit.Check(err)

			passHash, err := user.MakeMAC([]byte(pass), app.Config.Secret)
			appkit.Check(err)

			updatedUser := new(user.User)
			err = mongo.UpdateID(app.Db.C("users"), u.ID.Hex(), map[string]interface{}{
				"passwordhash": passHash,
			}, updatedUser)
			appkit.Check(err)

			// email to the user
			var buf bytes.Buffer

			tmplData := struct {
				FirstName, LastName, Password string
			}{
				FirstName: u.FirstName,
				LastName:  u.LastName,
				Password:  pass,
			}
			err = passwordResetTmpl.Execute(&buf, tmplData)
			appkit.Check(err)

			msg := mail.Message{
				From:    "no-reply@bahna.land",
				Subject: "Password reset",
				Body:    buf.String(),
				To:      []string{u.Email.Address},
				Created: time.Now(),
			}
			err = mail.Send(mail.DefaultConfig, msg)
			appkit.Check(err)

			// redirect
			url, err := app.Router.Get("login").URL("lang", lang.String())
			appkit.Check(err)
			http.Redirect(w, r, url.String(), http.StatusSeeOther)
		}
	})
}
