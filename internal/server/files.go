package server

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Agurato/starfin/internal/media"
	"github.com/radovskyb/watcher"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/exp/slices"
)

var (
	fileWatcher *watcher.Watcher

	watchedVolumes []*media.Volume
)

// InitFileWatching starts watching for changes in files
func InitFileWatching() (err error) {
	fileWatcher = watcher.New()
	// fileWatcher.FilterOps(watcher.Create, watcher.Rename)

	go fileWatchEventHandler()

	volumes, err := db.GetVolumes()
	if err != nil {
		return err
	}
	for _, v := range volumes {
		addFileWatch(&v)
		SynchronizeFilesAndDB(&v)
	}

	if err := fileWatcher.Start(1 * time.Second); err != nil {
		log.Fatalln(err)
	}

	return nil
}

// CloseFileWatching stops watching for changes in files
func CloseFileWatching() {
	fileWatcher.Close()
}

// addFileWatch adds a file watching on a specified volume
func addFileWatch(v *media.Volume) {
	if v.IsRecursive {
		if err := fileWatcher.AddRecursive(v.Path); err != nil {
			log.WithFields(log.Fields{"path": v.Path, "error": err}).Errorln("Could not watch volume")
			return
		}
	} else {
		if err := fileWatcher.Add(v.Path); err != nil {
			log.WithFields(log.Fields{"path": v.Path, "error": err}).Errorln("Could not watch volume")
			return
		}
	}
	watchedVolumes = append(watchedVolumes, v)
}

// fileWatchEventHandler handles file creation, renaming and deletion
func fileWatchEventHandler() {
	fileWrites := make(map[string]int64)

	createdFilesTicker := time.NewTicker(5 * time.Second)
	for {
		select {
		// Every X seconds, check if files are still being written
		case <-createdFilesTicker.C:
			for path, modTime := range fileWrites {
				info, err := os.Stat(path)
				if err != nil {
					delete(fileWrites, path)
					continue
				}
				// File is still being written
				if info.ModTime().Unix() != modTime {
					fileWrites[path] = info.ModTime().Unix()
					continue
				}

				// File is not being written anymore
				log.Debugln("File has stopped writing", path)
				delete(fileWrites, path)

				if err := handleFileCreate(path); err != nil {
					log.Errorln(err)
					continue
				}
			}
		// There is a new file event
		case event := <-fileWatcher.Event:
			log.WithField("event", event).Debugln("New file event")
			if event.Op == watcher.Create || event.Op == watcher.Write {
				// Add file if not a video or sub and if not already in map
				if _, ok := fileWrites[event.Path]; !ok && !event.IsDir() {
					ext := filepath.Ext(event.Path)
					// Add it to watch list if video or subtitle
					if media.IsVideoFileExtension(ext) || media.IsSubtitleFileExtension(ext) {
						fileWrites[event.Path] = 0
					}
				}
			} else if event.Op == watcher.Rename {
				handleFileRenamed(event.OldPath, event.Path)
			} else if event.Op == watcher.Remove {
				handleFileRemoved(event.Path)
			}
		// Error in file watching
		case err := <-fileWatcher.Error:
			log.WithField("error", err).Errorln("Error event")
			return
		// Stop watching files
		case <-fileWatcher.Closed:
			return
		}
	}
}

func getVolumeFromFilePath(path string) *media.Volume {
	for _, v := range watchedVolumes {
		if strings.HasPrefix(path, v.Path) {
			return v
		}
	}
	return nil
}

// addMovieFromPath adds a movie from its path and the volume
func addMovieFromPath(path string, volumeID primitive.ObjectID) error {
	// Get subtitle files in same directory
	subs, err := getRelatedSubFiles(path)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "path": path}).Debugln("Cannot get related subtitle files")
	}
	movie := media.NewMovie(path, volumeID, subs)
	// Search ID on TMDB
	if err := movie.FetchTMDBID(); err != nil {
		log.WithFields(log.Fields{"file": path, "error": err}).Warningln("Unable to fetch movie ID from TMDB")
		movie.Title = movie.Name
	} else {
		log.WithField("tmdbID", movie.TMDBID).Infoln("Found media with TMDB ID")
		// Fill info from TMDB
		movie.FetchDetails()
	}

	// Add media to DB
	if err = tryAddMovieToDB(movie); err != nil {
		log.WithField("path", movie.VolumeFiles[0].Path).Errorln(err)
	}

	return nil
}

// tryAddMovieToDB checks if the movie already exists in database before adding it
// Also adds persons to the database if they don't exist
func tryAddMovieToDB(movie *media.Movie) error {
	if movie.TMDBID == 0 || !db.IsMoviePresent(movie) {
		if err := db.AddMovie(movie); err != nil {
			return errors.New("cannot add movie to database")
		}
		// Cache poster, backdrop
		go func() {
			hasToWait, err := media.CachePoster(movie.PosterPath)
			if err != nil {
				log.WithFields(log.Fields{"error": err, "movieID": movie.ID}).Errorln("Could not cache poster")
			}
			if hasToWait {
				log.WithFields(log.Fields{"warning": err, "movieID": movie.ID}).Errorln("Will try to cache poster later")
			}
			hasToWait, err = media.CacheBackdrop(movie.BackdropPath)
			if err != nil {
				log.WithFields(log.Fields{"error": err, "movieID": movie.ID}).Errorln("Could not cache backdrop")
			}
			if hasToWait {
				log.WithFields(log.Fields{"warning": err, "movieID": movie.ID}).Errorln("Will try to cache backdrop later")
			}
		}()
	} else {
		if err := db.AddVolumeSourceToMovie(movie); err != nil {
			return errors.New("cannot add volume source to movie in database")
		}
	}

	for _, personID := range movie.GetCastAndCrewIDs() {
		if !db.IsPersonPresent(personID) {
			person := media.FetchPersonDetails(personID)
			db.AddPerson(person)
			// Cache photos
			go func() {
				hasToWait, err := media.CachePhoto(person.Photo)
				if err != nil {
					log.WithFields(log.Fields{"error": err, "personTMDBID": person.TMDBID}).Errorln("Could not cache photo")
				}
				if hasToWait {
					log.WithFields(log.Fields{"warning": err, "movieID": movie.ID}).Errorln("Will try to cache photo later")
				}
			}()
		}
	}

	return nil
}

// searchMediaFilesInVolume scans a volume and add all movies to the database
func searchMediaFilesInVolume(volume *media.Volume) {
	// Channel to add media to DB as they are fetched from TMDB
	movieChan := make(chan *media.Movie)

	go volume.Scan(movieChan)

	for {
		movie, more := <-movieChan
		if more {
			tryAddMovieToDB(movie)
		} else {
			break
		}
	}
}

// SynchronizeFilesAndDB synchronizes the database to the current files in the volume
// It adds the missing movies and subtitles from the database, and removes the movies and subtitles
// that are not currently in the volume
func SynchronizeFilesAndDB(volume *media.Volume) {
	videoFiles, subFiles, err := volume.ListVideoFiles()
	if err != nil {
		log.WithField("volume", volume.Path).Errorln("Could not synchronize volume with database")
	}

	// Add to database all new video files
	for _, videoFile := range videoFiles {
		// If movie is not in database
		if !db.IsMoviePathPresent(videoFile) {
			handleFileCreate(videoFile)
		}
	}

	// Add to database all new subtitle files
	for _, subFile := range subFiles {
		// If movie is not in database
		if !db.IsSubtitlePathPresent(subFile) {
			handleFileCreate(subFile)
		}
	}

	// Get all movies from volume
	movies := db.GetMoviesFromVolume(volume.ID)
	for _, movie := range movies {
		for _, volumeFile := range movie.VolumeFiles {
			// If the movie is not in the volume files, remove this movie
			if !slices.Contains(videoFiles, volumeFile.Path) {
				handleFileRemoved(volumeFile.Path)
			}
			// If the subtitle is not in the volume files, remove this subtitle
			for _, sub := range volumeFile.ExtSubtitles {
				if !slices.Contains(subFiles, sub.Path) {
					handleFileRemoved(sub.Path)
				}
			}
		}
	}
}
