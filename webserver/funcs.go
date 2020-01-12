package main

import (
	"errors"
	"fmt"
	"html/template"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Machiel/slugify"
	"github.com/bahna/magazine/webserver/cms"
	"github.com/bahna/magazine/webserver/file"
	"github.com/bahna/magazine/webserver/user"
	"github.com/nicksnyder/go-i18n/i18n"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
	"gopkg.in/mgo.v2/bson"
)

func generateTmplFuncs(app *application) template.FuncMap {
	m := map[string]interface{}{
		"T":            i18n.IdentityTfunc,
		"langName":     LangTagName,
		"langCode":     LangTagStr,
		"idToStr":      ObjectIDToString,
		"plus":         Plus,
		"incr":         Increment,
		"hasRole":      HasRole,
		"fmtTime":      FmtTime,
		"fmtTimeShort": FmtTimeShort,
		"zeroTime":     ZeroTime,
		"pubDate":      PubDate,
		"fmtInputTime": FmtInputTime,
		"inputTimeNow": InputTimeNow,
		"hasID":        HasID,
		"md":           Markdown,
		"translit":     Translit(app.Transliterator),
		"joinUsers":    JoinUsers,
		"joinTopics":   JoinTopics,
		"srcset":       Srcset,
		"bytesToMb":    BytesToMb,
		"dayNumber":    DayNumber,
		"month":        Month,
		"monthShort":   MonthShort,

		"mapTopicsToStyles": mapTopicsToStyles,
		"cutLine":           cutLine,
	}
	return template.FuncMap(m)
}

func cutLine(s string, n int) template.HTML {
	rr := []rune{}
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		rr = append(rr, r)
		s = s[size:]
	}
	if len(rr) > n {
		s = strings.TrimSpace(string(rr[:n])) + "…"
	} else {
		s = string(rr)
	}
	return template.HTML(s)
}

// mapTopicsToStyles maps a topic to its unique CSS-style.
func mapTopicsToStyles(tt []*cms.Topic) map[string]string {
	var topicCardStyles = []string{
		"rounded-a-lot",
		"rounded-rb-corner-a-lot",
		"",
		"rounded-lb-corner-a-lot",
	}
	m := make(map[string]string)
	l := len(topicCardStyles)
	for i, t := range tt {
		m[t.Title] = topicCardStyles[i%l]
	}
	return m
}

// LangTagName returns a language's name.
func LangTagName(tag language.Tag) string {
	return display.Self.Name(tag)
}

// LangTagStr returns a tags string.
func LangTagStr(tag language.Tag) string {
	return tag.String()
}

func Plus(a, b int) int {
	return a + b
}

func Increment(a int) int {
	return a + 1
}

func ObjectIDToString(id bson.ObjectId) string {
	return id.Hex()
}

func HasRole(s []user.Role, a int) bool {
	for _, v := range s {
		if v == user.Role(a) {
			return true
		}
	}
	return false
}

func HasID(s []bson.ObjectId, id bson.ObjectId) bool {
	for _, v := range s {
		if v.Hex() == id.Hex() {
			return true
		}
	}
	return false
}

// FmtTime formats time to a simple layout.
func FmtTime(t time.Time) string {
	if t == (time.Time{}) {
		return ""
	}
	return t.Format("2006-01-02 15:04")
}

// FmtTimeShort formats time to a simple layout.
func FmtTimeShort(t time.Time) string {
	if t == (time.Time{}) {
		return ""
	}
	return t.Format("2006-01-02")
}

// ZeroTime checks is the given time is empty.
func ZeroTime(t time.Time) bool {
	if t == (time.Time{}) {
		return true
	}
	return false
}

func FmtInputTime(t time.Time) string {
	// yyyy-MM-ddThh:mm
	return t.Format("2006-01-02T15:04")
}

func InputTimeNow() string {
	return FmtInputTime(time.Now())

}

// PubDate defines a content's publication date from Scheduled,
// Updated and Created fields.
func PubDate(c *cms.Content) string {
	const layout = "02.01.2006, 15:04"
	// if c.Scheduled != (time.Time{}) {
	// 	return c.Scheduled.Format(layout)
	// } else if c.Updated != (time.Time{}) {
	// 	return c.Updated.Format(layout)
	// }
	return c.Published.Format(layout)
}

func Translit(slug *slugify.Slugifier) func(s string) string {
	return func(s string) string {
		return slug.Slugify(s)
	}
}

// Srcset returns strings for HTML srcset attribute.
func Srcset(f *file.File) ([]string, error) {
	if f.Kind != file.ImageKind {
		return nil, errors.New("the file must be an image")
	}

	if len(f.Optimized) == 0 {
		return []string{f.URL}, nil
	}

	//log.Printf("file img: %+v", img)

	set := []string{}
	comma := ", "
	suffix := ""
	for i, v := range f.Optimized {
		if i == len(f.Optimized)-1 {
			comma = ""
		}
		if strings.Contains(v.URL, "@2x") {
			suffix = " 2x"
		}
		set = append(set, v.URL+suffix+comma)
	}

	//log.Printf("srcset: %+v", set)
	return set, nil
}

func BytesToMb(i int64) string {
	r := float64(i) / math.Pow(float64(1024), float64(2))
	return fmt.Sprintf("%.2f MB", r)
}

func DayNumber(t time.Time) string {
	return fmt.Sprintf("%d", t.Day())
}

func Month(t time.Time) string {
	return t.Month().String()
}

func MonthShort(t time.Time) string {
	return t.Month().String()[:3]
}

func JoinUsers(users []*user.User, delim string) string {
	s := []string{}
	for _, v := range users {
		n := fmt.Sprintf("%s %s", v.FirstName, v.LastName)
		s = append(s, n)
	}
	return strings.Join(s, delim)
}

func JoinTopics(topics []*cms.Topic, delim string) string {
	s := make([]string, len(topics))
	for i, v := range topics {
		s[i] = v.Title
	}
	return strings.Join(s, delim)
}
