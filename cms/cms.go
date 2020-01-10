// Package cms contains types and specialized functions for working
// with these.
package cms

import (
	"net/mail"
	"time"

	"github.com/bahna/magazine/cms/user"
	"gopkg.in/mgo.v2/bson"
)

// Content represents a piece of content which belongs to one or several topics
// and one or several authors.
type Content struct {
	ID bson.ObjectId `bson:"_id"`
	// Weight is the priority of the content. The bigger is more important.
	Weight int
	// Public shows if the content is public or not.
	Public bool
	// Promoted helps to define special content, which should be treated specially.
	Promoted bool
	// Page indicates if that piece of content is a separate page.
	// TODO: must be changed for ContentType Page. Update: orgsite, infocenter.
	Page bool
	// Language is two-letter language code of a content's language.
	// https://docs.mongodb.com/manual/tutorial/specify-language-for-text-index/#specify-default-language-text-index
	// two letter code: http://docs.mongodb.org/manual/reference/text-search-languages/#text-search-languages
	Language string

	// LanguageOverride is used for "be". We use .Language attribute in the UI and for searching and navigating
	// content. So we can't specify "none" value for .Language field for mongodb to be fine and not throwing
	// "language override unsupported be" error. So, use .LanguageOverride attribute to set it to "none" for "be" materials.
	LanguageOverride string `bson:"language_override,omitempty"`

	// Type represents a content's type.
	Type ContentType
	// Slug is a transliterated title and is used for URL composition and resultion.
	Slug string

	Created   time.Time
	Updated   time.Time
	Scheduled time.Time
	Published time.Time

	// Page meta information.
	PageSlug        string
	PageTitle       string
	PageDescription string

	// ParentID is used to specifye the parent content.
	ParentID *bson.ObjectId
	// Children contains all dependent content which has .ID as .ParentID in itself.
	Children []*Content `bson:"-"`

	// TopicIDs specifies to which topics this content belongs.
	// For this application we restrict the amount of topics to one
	// so that we can use the only topic to generate a unique URL
	// for a piece of content.
	TopicIDs []bson.ObjectId
	Topics   []*Topic `bson:"-"` // do not store in database

	// AuthorIDs specifies authors of the content.
	AuthorIDs []bson.ObjectId
	Authors   []*user.User `bson:"-"` // do not store in database

	Title string
	Lede  string
	Body  string

	// TODO: URL to external resources must be forbidden
	// CoverExternal is displayed at index pages.
	CoverExternal string
	// CoverInternal is displayed at the material page.
	CoverInternal string

	Images []struct {
		URL, Caption, LinkTo, Credits string // TODO: finish with Credits
	}

	// EventStart is a field for events.
	EventStart time.Time
	Location   string

	// LinkTo is a field for banners. It stores a URL to redirect to after clicking.
	LinkTo string

	// Payload is use for additional structured data.
	Payload map[string]interface{}
}

// ContentType is used to differentiate content of a website to display each content differently.
type ContentType int

const (
	// Article represents a text article.
	Article ContentType = iota
	// Banner is a banner.
	Banner
	// Audio represents podcast or other audio content.
	Audio
	// Video represents video content.
	Video
	// Page is a page.
	Page
	// Event is an event.
	Event

	// ArticleSeries is used to group several articles into a group of articles.
	ArticleSeries

	// Research is like a longread or a simple article but with different subtype
	// to differentiate between simple posts.
	Research

	// Photoreport is an article with photographs as main content.
	Photoreport
)

// ContentTypes is a list of available types.
var ContentTypes = []ContentType{
	Article,
	Banner,
	Audio,
	Video,
	Page,
	Event,
	ArticleSeries,
	Research,
	Photoreport,
}

// go generate fails with "bitbucket.org/iharsuvorau/wander", so we implement String() manually

func (t ContentType) String() string {
	switch t {
	case Article:
		return "Article"
	case Banner:
		return "Banner"
	case Audio:
		return "Audio"
	case Video:
		return "Video"
	case Page:
		return "Page"
	case Event:
		return "Event"
	case ArticleSeries:
		return "ArticleSeries"
	case Research:
		return "Research"
	case Photoreport:
		return "Photoreport"
	}
	return "Unknown ContentType"
}

// Topic represents a section of content grouped by a theme.
type Topic struct {
	ID     bson.ObjectId `bson:"_id"`
	Title  string
	Weight int
	Public bool
	Page   bool

	// Slug is a transliterated title and is used for URL composition and resolution.
	Slug string

	// https://docs.mongodb.com/manual/tutorial/specify-language-for-text-index/#specify-default-language-text-index
	Language string
	// LanguageOverride is used for "be". We use .Language attribute in the UI and for searching and navigating
	// content. So we can't specify "none" value for .Language field for mongodb to be fine and not throwing
	// "language override unsupported be" error. So, use .LanguageOverride attribute to set it to "none" for "be" materials.
	LanguageOverride string `bson:"language_override,omitempty"`
}

// Message represents a message from a website user.
type Message struct {
	ID       bson.ObjectId `bson:"_id"`
	Created  time.Time
	Status   MessageStatus
	FullName string
	Email    mail.Address
	Message  string
}

// MessageStatus represents a message status in the CMS.
type MessageStatus int

const (
	// New means a message is fresh and unprocessed.
	New MessageStatus = iota
	// InWork means a message is being processed.
	InWork
	// Closed means a message is no longer needs attention.
	Closed
)
