// tmplfuncs provides a common set of template operations.
package tmplfuncs

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/bahna/magazine/appkit"
	"github.com/bahna/magazine/cms"
	"github.com/bahna/magazine/cms/file"
	"github.com/bahna/magazine/cms/user"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
	"gopkg.in/mgo.v2/bson"
)

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

func Translit(app *appkit.Application) func(s string) string {
	return func(s string) string {
		return app.Transliterator.Slugify(s)
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
