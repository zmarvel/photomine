package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"path"
	"sync"

	"github.com/h2non/bimg"
)

type photo struct {
	Path        string
	Description string
	Thumbnail   string
	// TODO attributes (ISO, shutter speed, ...)
}

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

	// Use fixed thumbnail scaling factor.
	// TODO this should be made configurable.
	thumbDims := dims{1920 / 8, 1080 / 8}

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
	defer albumsDir.Close()

	albumPaths, err := albumsDir.Readdirnames(0)
	if err != nil {
		log.Fatalf("Failed to read _albums dir: %v", err)
	}

	var index albumIndex
	index.Title = "photomine"

	for _, subdirPath := range albumPaths {
		var album album
		subdirAbsPath := path.Join(albumsPath, subdirPath)
		info, err := os.Stat(subdirAbsPath)
		if err != nil {
			log.Printf("Failed to stat %s: %v", subdirAbsPath, err)
			continue
		}

		if !info.IsDir() {
			log.Printf("Skipping non-directory %s", subdirAbsPath)
			continue
		}

		album.Name = subdirPath
		// TODO relative path?
		album.Path = subdirPath
		index.Albums = append(index.Albums, album)
		// Don't populate Photos yet--not needed to build the index
	}

	indexHTML, err := os.Create(path.Join(outputPath, "index.html"))
	if err != nil {
		log.Fatalf("Failed to create index output file: %v", err)
	}
	defer indexHTML.Close()
	indexTemplate.Execute(indexHTML, index)

	var thumbWaitGroup sync.WaitGroup
	for _, album := range index.Albums {
		// log.Printf("Album path %s", album.Path)
		albumOutputPath := path.Join(outputPath, album.Path)

		// Copy the images into the output directory
		// TODO maybe allow filtering by filename
		inputDir := path.Join(albumsPath, album.Path)
		if err = copyDir(inputDir, albumOutputPath); err != nil {
			log.Fatalf("Failed to copy images: %v", err)
		}

		albumDir, err := os.Open(albumOutputPath)
		if err != nil {
			log.Fatalf("Failed to open album dir %s: %v", albumOutputPath, err)
		}
		defer albumDir.Close()

		photoPaths, err := albumDir.Readdirnames(0)
		if err != nil {
			log.Fatalf("Failed to read album dir %s: %v", albumOutputPath, err)
		}

		// Create thumbnails directory
		thumbDir := "thumb"
		err = os.Mkdir(path.Join(albumOutputPath, thumbDir), 0755)
		for _, photoPath := range photoPaths {
			thumbPath := path.Join(thumbDir, photoPath)
			photo := photo{photoPath, photoPath, thumbPath}
			album.Photos = append(album.Photos, photo)
		}
		thumbWaitGroup.Add(1)
		go func() {
			defer thumbWaitGroup.Done()
			album.createThumbs(albumOutputPath, thumbDims)
		}()

		albumIndexPath := path.Join(albumOutputPath, "index.html")
		albumIndex, err := os.Create(albumIndexPath)
		if err != nil {
			log.Fatalf("Failed to create album index %s: %v", albumIndexPath, err)
		}
		defer albumIndex.Close()

		albumTemplate.Execute(albumIndex, album)
	}

	thumbWaitGroup.Wait()
}

// Copy the file at fromPath to toPath.
func copyFile(fromPath string, toPath string) error {
	from, err := os.Open(fromPath)
	if err != nil {
		return err
	}
	defer from.Close()

	to, err := os.Create(toPath)
	if err != nil {
		return err
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	return err
}

// Copy all files in the directory at fromPath into the directory at toPath. If
// toPath does not exist, it will be created.
func copyDir(fromPath string, toPath string) error {
	toInfo, err := os.Stat(toPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	} else if os.IsNotExist(err) {
		err = os.Mkdir(toPath, 0755)
		if err != nil {
			return err
		}
	} else if !toInfo.IsDir() {
		return fmt.Errorf("%s exists and is not a directory", toPath)
	}

	from, err := os.Open(fromPath)
	if err != nil {
		return err
	}

	fromFiles, err := from.Readdir(0)
	if err != nil {
		return err
	}

	for _, fromFileInfo := range fromFiles {
		basename := fromFileInfo.Name()
		srcPath := path.Join(fromPath, basename)
		dstPath := path.Join(toPath, basename)
		if fromFileInfo.IsDir() {
			if err = copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err = copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

type dims struct {
	Width  int
	Height int
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
