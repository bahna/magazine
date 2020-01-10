package main

import (
	"time"

	"github.com/bahna/magazine/cms"
	"github.com/bahna/magazine/cms/user"
	"gopkg.in/mgo.v2/bson"
)

type userForm struct {
	ID, Email, FirstName, LastName, Password, PasswordConfirm string

	Roles []user.Role
}

type userChangePassForm struct {
	ID, Password, NewPassword, NewPasswordConfirm string
}

type contentForm struct {
	ID       bson.ObjectId `bson:"_id"`
	Weight   int
	Public   bool
	Promoted bool
	Slug     string
	Language string
	Type     cms.ContentType

	// we use it only for schema.Decoder to not complain about invalid path,
	// this field must be always handled automatically and not from a user form
	Created time.Time

	Scheduled time.Time

	PageSlug        string
	PageTitle       string
	PageDescription string

	ParentID *bson.ObjectId

	AuthorIDs []*bson.ObjectId
	TopicIDs  []*bson.ObjectId

	Title, Lede, Body, CoverExternal, CoverInternal string

	Images []struct {
		URL, Caption, LinkTo string
	}

	EventStart time.Time
	Location   string
	LinkTo     string

	Payload map[string]interface{}
}
