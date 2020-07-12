package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/h2non/bimg"
)

type albumIndex struct {
	Title  string
	Albums []album
}

type imageConfig struct {
	Extensions []string
}

func main() {
	// Assume the site root is the current working directory.
	// TODO accept an argument for this
	siteRoot, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}

	config, err := loadConfig(path.Join(siteRoot, "config.toml"))
	if err != nil {
		if os.IsNotExist(err) {
			config = defaultConfig()
		} else {
			log.Fatalf("Failed to load config: %v", err)
		}
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

	photoTemplate, err := template.ParseFiles(path.Join(templatePath, "photo.gohtml"))
	if err != nil {
		log.Fatalf("Failed to parse photo template: %v", err)
	}

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
	index.Title = config.Title

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
		album.Path = subdirPath
		index.Albums = append(index.Albums, album)
		// Don't populate Photos yet--not needed to build the index
	}

	// Sort albums by name
	sort.Slice(index.Albums, func(i, j int) bool {
		return index.Albums[i].Name < index.Albums[j].Name
	})

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
			if config.hasValidExt(photoPath) {
				thumbPath := path.Join(thumbDir, photoPath)
				var photo photo
				photo.Description = photoPath
				photo.Path = photoPath
				photo.Thumbnail = thumbPath
				parts := strings.Split(photo.Path, ".")
				photo.Page = strings.Join(parts[0:len(parts)-1], ".") + ".html"
				album.Photos = append(album.Photos, photo)
			}
		}
		sort.Slice(album.Photos, func(i, j int) bool {
			return album.Photos[i].Path < album.Photos[j].Path
		})
		thumbWaitGroup.Add(1)
		go func() {
			defer thumbWaitGroup.Done()
			album.createThumbs(albumOutputPath, thumbDims)
		}()

		// Build links between photos for next/previous links, and render photo page
		for i, photo := range album.Photos {
			if i > 0 {
				photo.Prev = album.Photos[i-1].Page
			}
			if i < len(album.Photos)-1 {
				photo.Next = album.Photos[i+1].Page
			}

			photoHTML, err := os.Create(path.Join(albumOutputPath, photo.Page))
			if err != nil {
				log.Fatalf("Failed to create photo page %s: %v", photo.Page, err)
			}

			err = photoTemplate.Execute(photoHTML, photo)
			if err != nil {
				log.Fatalf("Failed to execute photo template: %v", err)
			}
		}

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
