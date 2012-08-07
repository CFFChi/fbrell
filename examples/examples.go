// Package examples implements the DB and S3 backed examples backend for Rell.
package examples

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"github.com/daaku/go.errcode"
	"github.com/daaku/go.flag.pkgpath"
	"github.com/daaku/rell/redis"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"
)

// Some categories are hidden from the listing.
var hidden = map[string]bool{
	"bugs":   true,
	"fbml":   true,
	"hidden": true,
	"secret": true,
	"tests":  true,
}

type Example struct {
	Name    string `json:"-"`
	Content []byte `json:"-"`
	AutoRun bool   `json:"autoRun"`
	Title   string `json:"-"`
	URL     string `json:"-"`
}

type Category struct {
	Name    string
	Example []*Example
	Hidden  bool
}

type DB struct {
	Category []*Category
	Reverse  map[string]*Example
}

var (
	// Directory for disk backed DBs.
	oldExamplesDir = pkgpath.Dir(
		"rell.examples.old",
		"github.com/daaku/rell/examples/db/old",
		"The directory containing examples for the old SDK.")
	newExamplesDir = pkgpath.Dir(
		"rell.examples.new",
		"github.com/daaku/rell/examples/db/mu",
		"The directory containing examples for the new SDK.")

	// We have two disk backed DBs.
	old *DB
	mu  *DB

	// Stock response for the index page.
	emptyExample = &Example{Title: "Welcome", URL: "/", AutoRun: true}
)

// Loads a specific examples directory.
func loadDir(name string) (*DB, error) {
	categories, err := ioutil.ReadDir(name)
	if err != nil {
		return nil, fmt.Errorf("Failed to read directory %s: %s", name, err)
	}
	db := &DB{
		Category: make([]*Category, 0, len(categories)),
		Reverse:  make(map[string]*Example),
	}
	db.Reverse[ContentID(emptyExample.Content)] = emptyExample
	for _, categoryFileInfo := range categories {
		categoryName := categoryFileInfo.Name()
		if !categoryFileInfo.IsDir() {
			log.Printf(
				"Got unexpected file instead of directory for category: %s",
				categoryName)
			continue
		}
		category := &Category{
			Name:   categoryName,
			Hidden: hidden[categoryName],
		}
		categoryDir := filepath.Join(name, categoryName)
		examples, err := ioutil.ReadDir(categoryDir)
		if err != nil {
			return nil, fmt.Errorf("Failed to read category %s: %s", categoryDir, err)
		}
		category.Example = make([]*Example, 0, len(examples))
		for _, exampleFileInfo := range examples {
			exampleName := exampleFileInfo.Name()
			exampleFile := filepath.Join(categoryDir, exampleName)
			content, err := ioutil.ReadFile(exampleFile)
			if err != nil {
				return nil, fmt.Errorf(
					"Failed to read example %s: %s", exampleFile, err)
			}
			cleanName := exampleName[:len(exampleName)-5]
			example := &Example{
				Name:    cleanName,
				Content: content,
				AutoRun: true,
				Title:   categoryName + " · " + cleanName,
				URL:     path.Join("/", categoryName, cleanName),
			}
			category.Example = append(category.Example, example)
			db.Reverse[ContentID(bytes.TrimSpace(content))] = example
		}
		db.Category = append(db.Category, category)
	}
	return db, nil
}

// Load an Example for a given version and path.
func Load(version, path string) (*Example, error) {
	parts := strings.Split(path, "/")
	if len(parts) == 2 && parts[1] == "" {
		return emptyExample, nil
	} else if len(parts) == 4 {
		if parts[1] != "raw" && parts[1] != "simple" {
			return nil, errcode.New(http.StatusNotFound, "Invalid URL: %s", path)
		}
		parts = []string{"", parts[2], parts[3]}
	} else if len(parts) != 3 {
		return nil, errcode.New(http.StatusNotFound, "Invalid URL: %s", path)
	}

	if parts[1] == "saved" {
		item, err := redis.Client().Call("GET", makeKey(parts[2]))
		if err != nil {
			return nil, err
		}
		if item.Nil() {
			return nil, errcode.New(
				http.StatusNotFound, "Example not found: %s", path)
		}
		return &Example{
			Content: item.Elem.Bytes(),
			Title:   "Stored Example",
			URL:     path,
		}, nil
	}
	category := GetDB(version).FindCategory(parts[1])
	if category == nil {
		return nil, errcode.New(http.StatusNotFound, "Could not find category: %s", parts[1])
	}
	example := category.FindExample(parts[2])
	if example == nil {
		return nil, errcode.New(http.StatusNotFound, "Could not find example: %s", parts[2])
	}
	return example, nil
}

// Get the DB for a given SDK Version.
func GetDB(version string) *DB {
	var err error
	if version == "mu" {
		if mu == nil {
			mu, err = loadDir(*newExamplesDir)
			if err != nil {
				log.Fatal(err)
			}
		}
		return mu
	}
	if old == nil {
		old, err = loadDir(*oldExamplesDir)
		if err != nil {
			log.Fatal(err)
		}
	}
	return old
}

// Find a category by it's name.
func (d *DB) FindCategory(name string) *Category {
	for _, category := range d.Category {
		if category.Name == name {
			return category
		}
	}
	return nil
}

// Find an example by it's name.
func (c *Category) FindExample(name string) *Example {
	for _, example := range c.Example {
		if example.Name == name {
			return example
		}
	}
	return nil
}

// Save an Example.
func Save(id string, content []byte) error {
	if len(content) > 10240 {
		return errcode.New(
			http.StatusRequestEntityTooLarge,
			"Maximum allowed size is 10 kilobytes.")
	}
	_, err := redis.Client().Call("SET", makeKey(id), content)
	if err != nil {
		log.Printf("Error in cache.Set: %s", err)
	}
	return err
}

func makeKey(id string) string {
	return "fbrell_examples:" + id
}

func ContentID(content []byte) string {
	h := md5.New()
	_, err := h.Write(content)
	if err != nil {
		log.Fatalf("Error comupting md5 sum: %s", err)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
