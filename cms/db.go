package cms

// The file contains specialized functions for dealing with the
// database. The main package for database operations is
// "bitbucker.org/iharsuvorau/mongo".

// TODO: refactor all other references to use cms/db instead.

import (
	"fmt"
	"time"

	"bitbucket.org/iharsuvorau/mongo"
	"github.com/bahna/magazine/cms/user"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// AllTopics returs topics sorted by weight.
func AllTopics(db *mgo.Database, query interface{}) ([]*Topic, error) {
	db.Session.Refresh()
	items := []*Topic{}
	err := db.C("topics").Find(query).Sort("-weight").All(&items)
	return items, err
}

// AllUsers returns all users from a database.
func AllUsers(col *mgo.Collection, query interface{}) (items []*user.User, err error) {
	col.Database.Session.Refresh()
	err = col.Find(query).All(&items)
	return
}

// AllContent returns all content from a database.
func AllContent(db *mgo.Database, query interface{}) (items []*Content, err error) {
	db.Session.Refresh()
	if err = db.C("content").Find(query).Sort("-weight", "-created").All(&items); err != nil {
		return
	}

	for _, v := range items {
		v.Authors = []*user.User{}
		u := new(user.User)
		for _, id := range v.AuthorIDs {
			if err = mongo.GetID(db.C("users"), id.Hex(), u); err != nil {
				return
			}
		}
		v.Authors = append(v.Authors, u)
	}
	return
}

// AllContentSorted returns all content from a database sorted by the
// specified field: "-eventstart" or "eventstart".
func AllContentSorted(db *mgo.Database, query interface{}, sortField string) (items []*Content, err error) {
	db.Session.Refresh()
	if err = db.C("content").Find(query).Sort(sortField).All(&items); err != nil {
		return
	}

	for _, v := range items {
		v.Authors = []*user.User{}
		u := new(user.User)
		for _, id := range v.AuthorIDs {
			if err = mongo.GetID(db.C("users"), id.Hex(), u); err != nil {
				return
			}
		}
		v.Authors = append(v.Authors, u)
	}
	return
}

// AllContentLimited returns limited amount of content from a database.
func AllContentLimited(db *mgo.Database, query interface{}, limit int) (items []*Content, err error) {
	db.Session.Refresh()
	if limit == 0 {
		if err = db.C("content").Find(query).Sort("-weight", "-published").All(&items); err != nil {
			return
		}
	} else {
		if err = db.C("content").Find(query).Sort("-weight", "-published").Limit(limit).All(&items); err != nil {
			return
		}
	}

	for _, v := range items {
		err = GetAuthorsForContent(db, v)
		if err != nil {
			return
		}
		err = GetTopicsForContent(db, v)
		if err != nil {
			return
		}
	}
	return
}

// SearchContent matches the query only for the Content collection. It sorts results by the mongo score.
func SearchContent(db *mgo.Database, query interface{}, limit int) (items []*Content, err error) {
	db.Session.Refresh()
	if limit == 0 {
		if err = db.C("content").Find(query).Select(bson.M{"score": bson.M{"$meta": "textScore"}}).Sort("$textScore:score").All(&items); err != nil {
			return
		}
	} else {
		if err = db.C("content").Find(query).Select(bson.M{"score": bson.M{"$meta": "textScore"}}).Sort("$textScore:score").Limit(limit).All(&items); err != nil {
			return
		}
	}

	for _, v := range items {
		err = GetAuthorsForContent(db, v)
		if err != nil {
			return
		}
		err = GetTopicsForContent(db, v)
		if err != nil {
			return
		}
	}
	return
}

// AllContentByPage returns content items by page.
func AllContentByPage(col *mgo.Collection, query interface{}, perpage, page int) (items []*Content, prev, next int, err error) {
	col.Database.Session.Refresh()

	// TODO: sort by updated
	q := col.Find(query).Sort("-weight", "-published")
	viewed := (page - 1) * perpage

	var total int
	total, err = q.Count()
	if err != nil {
		return
	}
	if total > (viewed + perpage) {
		next = page + 1
	}

	if page > 1 {
		prev = page - 1
	}

	err = q.Limit(perpage).Skip(viewed).All(&items)
	if err != nil {
		return
	}

	for _, v := range items {
		err = GetAuthorsForContent(col.Database, v)
		if err != nil {
			return
		}
		err = GetTopicsForContent(col.Database, v)
		if err != nil {
			return
		}
	}

	return
}

// GetAuthorsForContent fetches authors for the provided content.
func GetAuthorsForContent(db *mgo.Database, c *Content) (err error) {
	db.Session.Refresh()
	c.Authors = []*user.User{}
	u := new(user.User)
	for _, id := range c.AuthorIDs {
		if err = mongo.GetID(db.C("users"), id.Hex(), u); err != nil {
			return
		}
		c.Authors = append(c.Authors, u)
	}
	return
}

// GetChildrenContent finds all dependent content and updates the parent.
func GetChildrenContent(db *mgo.Database, c *Content) (err error) {
	db.Session.Refresh()

	c.Children, err = AllContentLimited(db, bson.M{
		"parentid": c.ID,
		"public":   true,
		"$or": []bson.M{
			bson.M{"scheduled": bson.M{"$lt": time.Now()}},
			bson.M{"scheduled": (time.Time{})},
		},
	}, 0)

	return
}

// GetTopic returst a single topic by ID.
func GetTopic(db *mgo.Database, idStr string) (*Topic, error) {
	db.Session.Refresh()
	id := bson.ObjectIdHex(idStr)
	if !id.Valid() {
		return nil, fmt.Errorf("invalid ID")
	}
	t := new(Topic)
	err := db.C("topics").FindId(id).One(&t)
	return t, err
}

// GetTopicsForContent retrieves content from the database by .TopicIDs.
func GetTopicsForContent(db *mgo.Database, c *Content) (err error) {
	db.Session.Refresh()
	c.Topics = []*Topic{}
	for _, id := range c.TopicIDs {
		t := new(Topic)
		if err = mongo.GetID(db.C("topics"), id.Hex(), t); err != nil {
			return
		}
		c.Topics = append(c.Topics, t)
	}
	return
}
