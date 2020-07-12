package main

import (
	"io/ioutil"
	"path"
	"strings"

	"github.com/burntsushi/toml"
)

type imageConfig struct {
	Extensions []string
}

type config struct {
	Title       string
	BuildDir    string
	AlbumDir    string
	TemplateDir string
	Image       imageConfig
}

func (config *config) hasValidExt(filePath string) bool {
	ext := strings.TrimPrefix(path.Ext(filePath), ".")
	for _, validExt := range config.Image.Extensions {
		if ext == validExt {
			return true
		}
	}
	return false
}

func loadConfig(path string) (config, error) {
	var conf config
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return conf, err
	}

	if _, err := toml.Decode(string(data), &conf); err != nil {
		return conf, err
	}

	return conf, nil
}

func defaultConfig() config {
	return config{
		"photomine",
		"_build",
		"_albums",
		"_templates",
		imageConfig{
			[]string{},
		},
	}
}
