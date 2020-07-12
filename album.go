package main

import (
	"fmt"
	"log"
	"path"
)

type album struct {
	Name string
	Path string
	// TODO other metadata (location, time, ...)
	Photos []photo
}

func (album *album) createThumbs(basePath string, thumbDims dims) error {
	fmt.Printf("Creating thumbs for album %s in %s\n", album.Name, basePath)
	for _, photo := range album.Photos {
		photoPath := path.Join(basePath, photo.Path)
		thumbPath := path.Join(basePath, photo.Thumbnail)
		fmt.Printf("Photo: %s, Thumb: %s\n", photoPath, thumbPath)
		err := createThumbnail(photoPath, thumbPath, thumbDims)
		if err != nil {
			log.Printf("Failed to create thumb %s: %v", thumbPath, err)
		}
	}
	return nil
}
