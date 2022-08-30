package media

import (
	"os"
	"path/filepath"

	"github.com/alitto/pond"
	log "github.com/sirupsen/logrus"
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

// Scan files from volume that have not been added to the db yet
func (v Volume) Scan(mediaChan chan *Movie) {
	videoFiles, subFiles, err := v.ListVideoFiles()
	if err != nil {
		log.WithField("volumePath", v.Path).Warningln("Unable to scan folder for video files")
	}

	log.WithField("volumePath", v.Path).Debugln("Scanning volume")

	// Create worker pool of size 10
	pool := pond.New(20, 0, pond.MinWorkers(20))

	// For each file
	for _, file := range videoFiles {
		file := file
		pool.Submit(func() {
			movie := NewMovie(file, v.ID, subFiles)

			// Search ID on TMDB
			err = movie.FetchTMDBID()
			if err != nil {
				log.WithFields(log.Fields{"file": file, "err": err}).Warningln("Unable to fetch movie ID from TMDB")
				return
			}
			log.WithField("tmdbID", movie.TMDBID).Infoln("Found media with TMDB ID")

			// Fill info from TMDB
			movie.FetchDetails()

			// Send media to the channel
			mediaChan <- movie
		})
	}

	pool.StopAndWait()
	close(mediaChan)
}
