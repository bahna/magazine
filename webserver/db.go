// The file extends bitbucket.org/iharsuvorau/mongo API with private database API.

package main

import (
	"time"

	"github.com/bahna/magazine/webserver/cms"
	"golang.org/x/text/language"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

// ensureIndexes creates mongo indexes.
func ensureIndexes(session *mgo.Session, name string) (err error) {
	// Do not use DefaultLanguage in indexes because it won't
	// let you save documents with outher languages.
	// Just use the .Language attribute.

	content := mgo.Index{
		Key: []string{
			"$text:title",
			"$text:body",
			"$text:description",
		},
		DefaultLanguage:  "ru",
		LanguageOverride: "language_override",
	}

	topics := mgo.Index{
		Key: []string{
			"$text:name",
		},
		Unique:           true,
		DefaultLanguage:  "ru",
		LanguageOverride: "language_override",
	}

	users := mgo.Index{
		Key: []string{
			"$text:email",
		},
		Unique:          true,
		DefaultLanguage: "en",
	}

	err = session.DB(name).C("content").EnsureIndex(content)
	if err != nil {
		return
	}
	err = session.DB(name).C("topics").EnsureIndex(topics)
	if err != nil {
		return
	}
	err = session.DB(name).C("users").EnsureIndex(users)
	return
}

func getPages(db *mgo.Database, lang language.Tag) (pages []*cms.Content, err error) {
	pages, err = cms.AllContent(db, bson.M{
		"language": lang.String(),
		"public":   true,
		// "page":     true,
		"type": cms.Page,
		"$or": []bson.M{
			bson.M{"scheduled": bson.M{"$lt": time.Now()}},
			bson.M{"scheduled": (time.Time{})},
		},
	})
	if err != nil {
		return
	}

	for _, c := range pages {
		err = cms.GetTopicsForContent(db, c)
		if err != nil {
			return
		}
	}

	return
}

func getSeries(db *mgo.Database, lang language.Tag) (cc []*cms.Content, err error) {
	cc, err = cms.AllContent(db, bson.M{
		"language": lang.String(),
		"public":   true,
		"type":     cms.ArticleSeries,
		"$or": []bson.M{
			bson.M{"scheduled": bson.M{"$lt": time.Now()}},
			bson.M{"scheduled": (time.Time{})},
		},
	})
	if err != nil {
		return
	}

	for _, c := range cc {
		err = cms.GetChildrenContent(db, c)
		if err != nil {
			return
		}
		// limit children for the main page to the last 3 items
		if len(c.Children) > 3 {
			c.Children = c.Children[:3]
		}

		err = cms.GetTopicsForContent(db, c)
		if err != nil {
			return
		}

		err = cms.GetAuthorsForContent(db, c)
		if err != nil {
			return
		}
	}

	return
}

func getCertainContent(db *mgo.Database, lang language.Tag, ctype cms.ContentType) (cc []*cms.Content, err error) {
	cc, err = cms.AllContent(db, bson.M{
		"language": lang.String(),
		"public":   true,
		"type":     ctype,
		"$or": []bson.M{
			bson.M{"scheduled": bson.M{"$lt": time.Now()}},
			bson.M{"scheduled": (time.Time{})},
		},
	})
	if err != nil {
		return
	}

	for _, c := range cc {
		err = cms.GetTopicsForContent(db, c)
		if err != nil {
			return
		}

		err = cms.GetAuthorsForContent(db, c)
		if err != nil {
			return
		}
	}

	return
}

func getEvents(db *mgo.Database, lang language.Tag) (cc []*cms.Content, err error) {
	cc, err = cms.AllContentSorted(db, bson.M{
		"language": lang.String(),
		"public":   true,
		"type":     cms.Event,
		"$or": []bson.M{
			bson.M{"scheduled": bson.M{"$lt": time.Now()}},
			bson.M{"scheduled": (time.Time{})},
		},
		"eventstart": bson.M{"$gte": time.Now()},
	},
		"eventstart")
	if err != nil {
		return
	}

	for _, c := range cc {
		err = cms.GetTopicsForContent(db, c)
		if err != nil {
			return
		}

		err = cms.GetAuthorsForContent(db, c)
		if err != nil {
			return
		}
	}

	return
}

func getPosts(db *mgo.Database, lang language.Tag, limit int) (cc []*cms.Content, err error) {
	findParams := bson.M{
		"language": lang.String(),
		"public":   true,
		"$or": []bson.M{
			bson.M{"scheduled": bson.M{"$lt": time.Now()}},
			bson.M{"scheduled": (time.Time{})},
		},
	}

	if limit > 0 {
		cc, err = cms.AllContentLimited(db, findParams, limit)
	} else {
		cc, err = cms.AllContentLimited(db, findParams, 0)
	}
	if err != nil {
		return
	}

	for _, c := range cc {
		err = cms.GetAuthorsForContent(db, c)
		if err != nil {
			return
		}
		err = cms.GetTopicsForContent(db, c)
		if err != nil {
			return
		}
	}

	return
}

func getTopics(db *mgo.Database, lang language.Tag) ([]*cms.Topic, error) {
	return cms.AllTopics(db, bson.M{
		"language": lang.String(),
		"public":   true,
		"page":     false,
	})
}
