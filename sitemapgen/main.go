// The tool to generate []sitemap.Item for any CMS application.

package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/bahna/magazine/webserver/cms"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func main() {
	dbhost := flag.String("dbhost", "192.168.99.100", "database host")
	dbname := flag.String("dbname", "magazine", "database name")
	outpath := flag.String("out", "./", "output directory for sitemap.xml")
	prefix := flag.String("prefix", "", "global prefix to the relative URL document location")
	flag.Parse()

	session, err := mgo.Dial(*dbhost)
	if err != nil {
		log.Fatalf("failed to dial the database server at %s: %v", *dbhost, err)
	}
	defer session.Close()

	items, err := collectItems(session.DB(*dbname), *prefix)
	if err != nil {
		log.Fatalf("failed to collect items: %v", err)
	}

	path := path.Join(*outpath, "sitemap.xml")
	if err = Save(path, items); err != nil {
		log.Fatalf("failed to create an XML: %v", err)
	}
}

func collectItems(dbs *mgo.Database, prefix string) (items []Item, err error) {
	items = []Item{}

	topics, err := cms.AllTopics(dbs, bson.M{"public": true})
	if err != nil {
		return
	}
	for _, v := range topics {
		items = append(items, Item{
			Loc:        fmt.Sprintf("%s/%s/%s", prefix, v.Language, v.Slug),
			ChangeFreq: "daily",
		})
	}

	content, err := cms.AllContent(dbs, bson.M{"public": true})
	if err != nil {
		return
	}
	for _, v := range content {
		if err = cms.GetTopicsForContent(dbs, v); err != nil {
			return
		}

		for _, t := range v.Topics {
			items = append(items, Item{
				Loc:     fmt.Sprintf("%s/%s/%s/%s", prefix, v.Language, t.Slug, v.Slug),
				Lastmod: v.Published.Format("2006-01-02"),
			})
		}
	}

	return
}

// TimeLayout is a layout used for Item.Lastmod formatting.
const TimeLayout = "2006-01-02"

// Item is an item of a  Should be provided by web apps.
type Item struct {
	Loc        string  `xml:"loc"` // required
	Lastmod    string  `xml:"lastmod,omitempty"`
	ChangeFreq string  `xml:"changefreq,omitempty"`
	Priority   float32 `xml:"priority,omitempty"` // 0 <= x <= 1
}

type urlset struct {
	XMLName xml.Name `xml:"urlset"`
	Xmlns   string   `xml:"xmlns,attr"`
	Items   []Item   `xml:"url"`
}

func Save(path string, items []Item) error {
	m := urlset{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		Items: items,
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	f.Write([]byte(xml.Header))
	return xml.NewEncoder(f).Encode(m)
}
