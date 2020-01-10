package sitemap

import (
	"encoding/xml"
	"os"
)

// TimeLayout is a layout used for Item.Lastmod formatting.
const TimeLayout = "2006-01-02"

// Item is an item of a sitemap. Should be provided by web apps.
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
