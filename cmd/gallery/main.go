package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kpango/glg"
	"github.com/rking788/destiny-gear-vendor/db"
)

type itemMetadata struct {
	ModelName    string
	ItemName     string
	ThumbnailURL string
	Tier         int
}
type output struct {
	Items []*itemMetadata
}

func main() {

	inPath := flag.String("path", "", "The directory containing all of the USDZ files to be served and where the new HTML and thumbnails should be written.")

	flag.Parse()

	if *inPath == "" {
		glg.Errorf("Forgot to specify a path to the USDZ")
		return
	}

	db, err := db.GetAssetDBConnection()
	if err != nil {
		glg.Errorf("Error trying to get database connection: %s", err.Error())
		return
	}

	itemLookup, err := db.GetIconLookup()
	if err != nil {
		glg.Errorf("Error trying to select item metadata: %s", err.Error())
		return
	}
	glg.Info("Finished DB lookup")

	existing, err := findExistingItems(*inPath, itemLookup)
	if err != nil {
		glg.Errorf("Error getting matching files in directory: %s", err.Error())
		return
	}

	glg.Infof("About to write %d items to the HTML gallery", len(existing))

	indexPath := path.Join(*inPath, "index.html")
	os.Remove(indexPath)
	outF, err := os.OpenFile(indexPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		glg.Errorf("Error opening index.html: %s", err.Error())
		return
	}

	templateData := output{}
	templateData.Items = make([]*itemMetadata, 0, 200)

	for _, item := range existing {
		i := &itemMetadata{
			ModelName:    item.Hash + ".usdz",
			ItemName:     item.Name,
			ThumbnailURL: fmt.Sprintf("https://www.bungie.net%s", item.Icon),
			Tier:         item.Tier,
		}

		templateData.Items = append(templateData.Items, i)
	}

	sort.Sort(sortTier(templateData.Items))

	tpl := template.New("gallery.tpl.html")
	tpl.ParseFiles("gallery.tpl.html")
	tpl.Execute(outF, templateData)

	// Copy the screen.css file to the same output directory as the index.html
	cssPath := path.Join(*inPath, "screen.css")
	inF, err := os.Open("screen.css")
	if err != nil {
		glg.Errorf("Error copying screen.css to destination: ", err.Error())
		return
	}
	defer inF.Close()

	outCSS, err := os.Create(cssPath)
	if err != nil {
		glg.Errorf("Error opening destination file for screen.css: ", err.Error())
		return
	}
	defer outF.Close()

	_, err = io.Copy(inF, outCSS)
	if err != nil {
		glg.Errorf("Failed to copy input to output for screen.css: ", err.Error())
		return
	}
}

type sortTier []*itemMetadata

func (t sortTier) Len() int           { return len(t) }
func (t sortTier) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t sortTier) Less(i, j int) bool { return t[i].Tier > t[j].Tier }

func findExistingItems(base string, all map[string]*db.Item) (map[string]*db.Item, error) {

	result := make(map[string]*db.Item)

	usdzGlob := path.Join(base, "*.usdz")
	matches, err := filepath.Glob(usdzGlob)
	if err != nil {
		return nil, err
	}

	for _, match := range matches {
		glg.Infof("Found file: %s", match)

		comps := strings.Split(match, "/")
		hash := strings.Split(comps[len(comps)-1], ".")[0]
		item := all[hash]

		glg.Infof("%s: %s", item.Hash, item.Name)

		result[item.Hash] = item
	}

	return result, nil
}
