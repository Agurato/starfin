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
func (v Volume) ListVideoFiles() ([]string, error) {
	var (
		files      []string
		videoFiles []string
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
			return nil, err
		}
	} else {
		f, err := os.Open(v.Path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		fileInfos, err := f.Readdir(-1)
		if err != nil {
			return nil, err
		}
		for _, fileInfo := range fileInfos {
			if !fileInfo.IsDir() {
				files = append(files, filepath.Join(v.Path, fileInfo.Name()))
			}
		}
	}

	for _, file := range files {
		if IsVideoFileExtension(filepath.Ext(file)) {
			videoFiles = append(videoFiles, file)
		}
	}

	return videoFiles, nil
}

// Scan files from volume that have not been added to the db yet
func (v Volume) Scan(mediaChan chan Media, actorChan chan []Actor) {
	files, err := v.ListVideoFiles()
	if err != nil {
		log.WithField("volumePath", v.Path).Warningln("Unable to scan folder for video files")
	}

	log.WithField("volumePath", v.Path).Debugln("Scanning volume")

	// Create worker pool of size 10
	pool := pond.New(10, 0, pond.MinWorkers(10))

	// For each file
	for _, file := range files {
		file := file
		pool.Submit(func() {
			media := CreateMediaFromFilename(file, v.ID)

			// Search ID on TMDB
			err = media.FetchMediaID()
			if err != nil {
				log.WithFields(log.Fields{"file": file, "err": err}).Warningln("Unable to fetch movie ID from TMDB")
				return
			}
			log.WithField("tmdbID", media.GetTMDBID()).Infoln("Found media with TMDB ID")

			// Fill info from TMDB
			actors := media.FetchMediaDetails()

			// Send media to the channel
			mediaChan <- media

			// Send actors to the channel
			actorChan <- actors
		})
	}

	pool.StopAndWait()
	close(mediaChan)
	close(actorChan)
}
