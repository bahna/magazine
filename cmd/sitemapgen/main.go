// The tool to generate []sitemap.Item for any CMS application.

package main

import (
	"flag"
	"fmt"
	"log"
	"path"

	"github.com/bahna/magazine/appkit/sitemap"
	"github.com/bahna/magazine/cms"
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
	if err = sitemap.Save(path, items); err != nil {
		log.Fatalf("failed to create an XML: %v", err)
	}
}

func collectItems(dbs *mgo.Database, prefix string) (items []sitemap.Item, err error) {
	items = []sitemap.Item{}

	topics, err := cms.AllTopics(dbs, bson.M{"public": true})
	if err != nil {
		return
	}
	for _, v := range topics {
		items = append(items, sitemap.Item{
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
			items = append(items, sitemap.Item{
				Loc:     fmt.Sprintf("%s/%s/%s/%s", prefix, v.Language, t.Slug, v.Slug),
				Lastmod: v.Published.Format("2006-01-02"),
			})
		}
	}

	return
}
