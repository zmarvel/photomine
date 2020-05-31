package main

import (
	"html/template"
	"log"
	"os"
	"path"
)

type photo struct {
	Path        string
	Description string
	// TODO attributes (ISO, shutter speed, ...)
}

type album struct {
	Name string
	Path string
	// TODO other metadata (location, time, ...)
	Photos []photo
}

type albumIndex struct {
	Title  string
	Albums []album
}

func main() {
	// Assume the site root is the current working directory.
	// TODO accept an argument for this

	siteRoot, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}

	// Load templates
	templatePath := path.Join(siteRoot, "_templates")
	indexTemplate, err := template.ParseFiles(path.Join(templatePath, "index.gohtml"))
	if err != nil {
		log.Fatalf("Failed to parse index template: %v", err)
	}

	albumTemplate, err := template.ParseFiles(path.Join(templatePath, "album.gohtml"))
	if err != nil {
		log.Fatalf("Failed to parse album template: %v", err)
	}
	_ = albumTemplate

	// Create output directory
	outputPath := path.Join(siteRoot, "_build")
	err = os.Mkdir(outputPath, 0755)
	if err != nil {
		log.Fatalf("Failed to create _build dir: %v", err)
	}

	// Walk albums dir and construct a list of albums
	albumsPath := path.Join(siteRoot, "_albums")
	albumsDir, err := os.Open(albumsPath)
	if err != nil {
		log.Fatalf("Failed to open _albums dir: %v", err)
	}

	albumPaths, err := albumsDir.Readdirnames(0)
	if err != nil {
		log.Fatalf("Failed to read _albums dir: %v", err)
	}

	var index albumIndex
	index.Title = "photomine"

	for _, subdirPath := range albumPaths {
		var album album
		subdirAbsPath := path.Join(albumsPath, subdirPath)
		album.Name = subdirPath
		album.Path = subdirAbsPath
		index.Albums = append(index.Albums, album)
		// Don't populate Photos yet--not needed to build the index
	}

	// TODO expand index template
	indexHTML, err := os.Create(path.Join(outputPath, "index.html"))
	if err != nil {
		log.Fatalf("Failed to create index output file: %v", err)
	}
	indexTemplate.Execute(indexHTML, index)

	for _, album := range index.Albums {
		albumDir, err := os.Open(album.Path)
		if err != nil {
			log.Fatalf("Failed to open album dir %s: %v", album.Path, err)
		}

		photoPaths, err := albumDir.Readdirnames(0)
		if err != nil {
			log.Fatalf("Failed to read album dir %s: %v", album.Path, err)
		}

		for _, photoPath := range photoPaths {
			photo := photo{photoPath, photoPath}
			album.Photos = append(album.Photos, photo)
		}

		// TODO expand album template
	}
}
