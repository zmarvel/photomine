package main

import (
	"fmt"
	"log"
	"path"

	"github.com/h2non/bimg"
)

type album struct {
	Name string
	Path string
	// TODO other metadata (location, time, ...)
	Photos []photo
}

func createThumbnail(fromPath string, thumbPath string, thumbDims dims) error {
	buf, err := bimg.Read(fromPath)
	if err != nil {
		return err
	}

	fullSize := bimg.NewImage(buf)
	origSize, err := fullSize.Size()
	if err != nil {
		return err
	}

	// Assume thumbDims.Width > thumbDims.Height
	var thumbImg []byte
	if origSize.Width > origSize.Height {
		// Horizontal orientation
		thumbImg, err = fullSize.Resize(thumbDims.Width, thumbDims.Height)
		if err != nil {
			return err
		}
	} else {
		// Vertical orientation
		thumbImg, err = fullSize.Resize(thumbDims.Height, thumbDims.Width)
		if err != nil {
			return err
		}
	}

	err = bimg.Write(thumbPath, thumbImg)
	return err
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
