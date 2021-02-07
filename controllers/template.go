package controllers

import (
	"html/template"
	"log"
	"os"
	"path/filepath"
)

// All Templates
var (
	AllHTMLTemplates *template.Template
)

// RetrieveAllTemplate = Get all .html templates
func RetrieveAllTemplate() {
	var htmlListing []string
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	err = filepath.Walk(workingDir+"/templates/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		switch checkDir := info.Mode(); {
		case checkDir.IsRegular():
			htmlListing = append(htmlListing, path)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	AllHTMLTemplates = template.Must(template.ParseFiles(htmlListing...))
}
