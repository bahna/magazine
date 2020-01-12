// Package mongo provides basic API for documents management in a mongo database.
package mongo

import (
	"errors"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	ErrInvalidID = errors.New("invalid ID")
)

func Save(col *mgo.Collection, selector interface{}, item interface{}) error {
	col.Database.Session.Refresh()
	_, err := col.Upsert(selector, item)
	return err
}

func Delete(col *mgo.Collection, idStr string) error {
	id := bson.ObjectIdHex(idStr)
	if !id.Valid() {
		return ErrInvalidID
	}
	col.Database.Session.Refresh()
	return col.RemoveId(id)
}

// GetID fetches a document from a database into the provided interface.
func GetID(col *mgo.Collection, idStr string, dst interface{}) error {
	var id bson.ObjectId
	if id = bson.ObjectIdHex(idStr); !id.Valid() {
		return ErrInvalidID
	}
	col.Database.Session.Refresh()
	return col.FindId(id).One(dst)
}

// GetOne fetches an item from a database.
func GetOne(col *mgo.Collection, query, dst interface{}) error {
	col.Database.Session.Refresh()
	return col.Find(query).One(dst)
}

// UpdateID updates a database by the specified ID with the provided mgo.Change.
func UpdateID(col *mgo.Collection, idStr string, set interface{}, dst interface{}) error {
	id := bson.ObjectIdHex(idStr)
	if !id.Valid() {
		return ErrInvalidID
	}
	col.Database.Session.Refresh()
	_, err := col.FindId(id).Apply(mgo.Change{
		Update:    bson.M{"$set": set},
		ReturnNew: true,
	}, dst)
	return err
}

// UpdateOne updates a record found by a query.
func UpdateOne(col *mgo.Collection, query, set, dst interface{}) error {
	col.Database.Session.Refresh()
	_, err := col.Find(query).Apply(mgo.Change{
		Update:    bson.M{"$set": set},
		ReturnNew: true,
	}, dst)
	return err
}
