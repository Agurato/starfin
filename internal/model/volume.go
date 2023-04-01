package model

import (
	"os"
	"path/filepath"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Volume holds the volume paths to fetch media from
type Volume struct {
	ID          primitive.ObjectID `bson:"_id"`
	Name        string             `bson:"name"`
	Path        string             `bson:"path"`
	IsRecursive bool               `bson:"is_recursive"`
	MediaType   string             `bson:"media_type"`
}

type VolumeFile struct {
	Path         string             `bson:"path"`
	FromVolume   primitive.ObjectID `bson:"from_volume"`
	Info         MediaInfo          `bson:"info"`
	ExtSubtitles []Subtitle         `bson:"ext_subtitles"`
}

type Subtitle struct {
	Language string `bson:"language"`
	Path     string `bson:"path"`
}

// ListVideoFiles lists all the files that are considered as video in the volume
func (v Volume) ListVideoFiles() ([]string, []string, error) {
	var (
		files      []string
		videoFiles []string
		subFiles   []string
		err        error
	)
	if v.IsRecursive {
		err = filepath.Walk(v.Path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, nil, err
		}
	} else {
		f, err := os.Open(v.Path)
		if err != nil {
			return nil, nil, err
		}
		defer f.Close()
		fileInfos, err := f.Readdir(-1)
		if err != nil {
			return nil, nil, err
		}
		for _, fileInfo := range fileInfos {
			if !fileInfo.IsDir() {
				files = append(files, filepath.Join(v.Path, fileInfo.Name()))
			}
		}
	}

	for _, file := range files {
		ext := filepath.Ext(file)
		if IsVideoFileExtension(ext) {
			videoFiles = append(videoFiles, file)
		} else if IsSubtitleFileExtension(ext) {
			subFiles = append(subFiles, file)
		}
	}

	return videoFiles, subFiles, nil
}
