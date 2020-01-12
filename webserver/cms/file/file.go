package file

// I've separated file API out of the cms package because only this
// API uses the wander package which requires VIPS to be installed.

import (
	"errors"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/bahna/magazine/webserver/cms"
	"github.com/bahna/magazine/webserver/mongo"

	// NOTE: there is a strange behaviour while trying to "go generate" when this file is present
	// because of an import of "bitbucket.org/iharsuvorau/wander" which in imports
	// "github.com/davidbyttow/govips/pkg/vips".
	// To resolve the issue just remove the content of the file while executing "go generate".
	"bitbucket.org/iharsuvorau/wander"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	FileKind int = iota
	ImageKind
)

// File is used for file uploads.
type File struct {
	ID    bson.ObjectId `bson:"_id"`
	Title string
	// Credits field contains information about a file source or a name of an author.
	Credits string
	Kind    int
	URL     string
	Size    int64
	Created time.Time

	// Optimized can contain several URLs to optimized versions of a file from
	// the original File.URL field. Usually, it is used for images to store several
	// formats and sizes of them.
	Optimized []*OptimizedImage
}

type OptimizedImage struct {
	URL  string
	Size int64
}

// UploadFromForm dumps a file to a file system and upserts into a mongo database.
func UploadFromForm(col *mgo.Collection, fh *multipart.FileHeader, outputDir, caption, credits string, optimize bool) error {
	var kind int
	var canBeOptimized bool
	var optimizedURLs []string

	for _, v := range fh.Header["Content-Type"] {
		if strings.Contains(v, "image") {
			kind = ImageKind
			canBeOptimized = true
			break
		}
	}
	id := bson.NewObjectId()
	ext := path.Ext(fh.Filename)
	name := id.Hex() + ext
	url := fmt.Sprintf("/files/%s", name)
	filename := path.Join(outputDir, name)

	if f, err := fh.Open(); err == nil {
		b, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		if len(b) == 0 {
			return fmt.Errorf("empty file")
		}
		if err = writeFileFromBytes(b, filename); err != nil {
			return err
		}
		if err = f.Close(); err != nil {
			return err
		}
	} else {
		return err
	}

	if canBeOptimized && optimize {
		// NOTE: to use wander.optimizeVips make sure the libvips pkg is installed:
		// https://bitbucket.org/iharsuvorau/bahna/downloads/
		optimizedMap, err := wander.Optimize(wander.OptimizeVips, wander.Config{
			OutputDir:    path.Join(outputDir, "optimized"),
			OutputPrefix: "/optimized/",
			Sizes:        []string{"1x", "2x"},
			Formats:      []string{"jpg", "webp"},
		}, filename)
		if err != nil {
			return err
		}
		optimizedURLs = optimizedMap[path.Base(filename)]
	}

	var optimized = make([]*OptimizedImage, len(optimizedURLs))
	for i, v := range optimizedURLs {
		var size int64
		fn := path.Join(outputDir, v)
		if f, err := os.Open(fn); err != nil {
			return fmt.Errorf("failed to open an optimized image: %v", err)
		} else {
			stat, err := f.Stat()
			if err != nil {
				return fmt.Errorf("failed to get optimized image stats: %v", err)
			}
			size = stat.Size()
			if err = f.Close(); err != nil {
				return fmt.Errorf("failed to close optimized image: %v", err)
			}
		}
		optimized[i] = &OptimizedImage{
			URL:  path.Join("/", "files", v),
			Size: size,
		}
	}

	col.Database.Session.Refresh()
	_, err := col.UpsertId(id, &File{
		ID:        id,
		Title:     caption,
		Credits:   credits,
		Kind:      kind,
		URL:       url,
		Size:      fh.Size,
		Optimized: optimized,
		Created:   time.Now(),
	})

	return err
}

// AllFiles returns files from a database.
func AllFiles(col *mgo.Collection, query interface{}) ([]*File, error) {
	col.Database.Session.Refresh()
	items := []*File{}
	err := col.Find(query).All(&items)
	return items, err
}

// AllFilesByPage returns files by page.
func AllFilesByPage(col *mgo.Collection, query interface{}, perpage, page int) (items []*File, prev, next, total int, err error) {
	col.Database.Session.Refresh()

	q := col.Find(query).Sort("-created")
	viewed := (page - 1) * perpage

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

	return
}

// RemoveFile deletes a file from a file system and a database.
func RemoveFile(col *mgo.Collection, idStr string, filesDir string) error {
	var id bson.ObjectId
	if id = bson.ObjectIdHex(idStr); !id.Valid() {
		return errors.New("invalid ID")
	}

	file := new(File)
	col.Database.Session.Refresh()

	err := col.FindId(id).One(file)
	if err != nil {
		return err
	}

	filename := path.Join(filesDir, id.Hex()+path.Ext(file.URL))

	if err = col.RemoveId(id); err != nil {
		return err
	}

	paths, err := filepath.Glob(path.Join(path.Dir(filename), "optimized", id.Hex()+"*"))
	if err != nil {
		return err
	}
	paths = append(paths, filename)
	for _, v := range paths {
		if err = os.Remove(v); err != nil {
			return err
		}
	}

	return nil
}

// writeFileFromBytes writes bytes to a file, it creates directories if needed.
func writeFileFromBytes(b []byte, filepath string) error {
	err := ioutil.WriteFile(filepath, b, 0666)
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			if err = os.Mkdir(path.Dir(filepath), 0777); err != nil {
				return err
			}

			// one more try
			if err = ioutil.WriteFile(filepath, b, 0666); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

// GetImagesForContent fetches images for the provided content which are located
// in the Content.Images attribute only.
func GetImagesForContent(db *mgo.Database, c *cms.Content) (err error) {
	db.Session.Refresh()
	col := db.C("files")

	if len(c.Images) == 0 {
		return
	}

	for i, v := range c.Images {
		img := new(File)
		err = mongo.GetOne(col, bson.M{"url": v.URL}, img)
		if err != nil {
			if err == mgo.ErrNotFound {
				err = mongo.GetOne(col, bson.M{"optimized.url": v.URL}, img)
				if err != nil {
					err = fmt.Errorf("image %v not found: %v", v.URL, err)
					return
				}
				err = nil
			}
		}
		c.Images[i].Credits = img.Credits
	}

	return
}
