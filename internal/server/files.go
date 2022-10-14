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

	initFilters()

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

// addFilmFromPath adds a film from its path and the volume
func addFilmFromPath(path string, volumeID primitive.ObjectID) error {
	// Get subtitle files in same directory
	subs, err := getRelatedSubFiles(path)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "path": path}).Debugln("Cannot get related subtitle files")
	}
	film := media.NewFilm(path, volumeID, subs)
	// Search ID on TMDB
	if err := film.FetchTMDBID(); err != nil {
		log.WithFields(log.Fields{"file": path, "error": err}).Warningln("Unable to fetch film ID from TMDB")
		film.Title = film.Name
	} else {
		log.WithField("tmdbID", film.TMDBID).Infoln("Found media with TMDB ID")
		// Fill info from TMDB
		film.FetchDetails()
	}

	// Add media to DB
	if err = tryAddFilmToDB(film); err != nil {
		log.WithField("path", film.VolumeFiles[0].Path).Errorln(err)
	}

	return nil
}

// tryAddFilmToDB checks if the film already exists in database before adding it
// Also adds persons to the database if they don't exist
func tryAddFilmToDB(film *media.Film) error {
	if film.TMDBID == 0 || !db.IsFilmPresent(film) {
		if err := db.AddFilm(film); err != nil {
			return errors.New("cannot add film to database")
		}
		addToFilters(film)
		// Cache poster, backdrop
		go cachePosterAndBackdrop(film)
	} else {
		if err := db.AddVolumeSourceToFilm(film); err != nil {
			return errors.New("cannot add volume source to film in database")
		}
	}

	for _, personID := range film.GetCastAndCrewIDs() {
		if !db.IsPersonPresent(personID) {
			person := media.FetchPersonDetails(personID)
			db.AddPerson(person)
			// Cache photos
			go cachePersonPhoto(&person)
		}
	}

	return nil
}

// cachePosterAndBackdrop caches the poster and the backdrop image of a film
func cachePosterAndBackdrop(film *media.Film) {
	hasToWait, err := media.CachePoster(film.PosterPath)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "filmID": film.ID}).Errorln("Could not cache poster")
	}
	if hasToWait {
		log.WithFields(log.Fields{"warning": err, "filmID": film.ID}).Errorln("Will try to cache poster later")
	}
	hasToWait, err = media.CacheBackdrop(film.BackdropPath)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "filmID": film.ID}).Errorln("Could not cache backdrop")
	}
	if hasToWait {
		log.WithFields(log.Fields{"warning": err, "filmID": film.ID}).Errorln("Will try to cache backdrop later")
	}
}

// cacheCast caches the person's image
func cachePersonPhoto(person *media.Person) {
	hasToWait, err := media.CachePhoto(person.Photo)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "personTMDBID": person.TMDBID}).Errorln("Could not cache photo")
	}
	if hasToWait {
		log.WithFields(log.Fields{"warning": err, "personTMDBID": person.TMDBID}).Errorln("Will try to cache photo later")
	}
}

// searchMediaFilesInVolume scans a volume and add all films to the database
func searchMediaFilesInVolume(volume *media.Volume) {
	// Channel to add media to DB as they are fetched from TMDB
	filmChan := make(chan *media.Film)

	go volume.Scan(filmChan)

	for {
		film, more := <-filmChan
		if more {
			tryAddFilmToDB(film)
		} else {
			break
		}
	}
}

// SynchronizeFilesAndDB synchronizes the database to the current files in the volume
// It adds the missing films and subtitles from the database, and removes the films and subtitles
// that are not currently in the volume
func SynchronizeFilesAndDB(volume *media.Volume) {
	videoFiles, subFiles, err := volume.ListVideoFiles()
	if err != nil {
		log.WithField("volume", volume.Path).Errorln("Could not synchronize volume with database")
	}

	// Add to database all new video files
	for _, videoFile := range videoFiles {
		// If film is not in database
		if !db.IsFilmPathPresent(videoFile) {
			handleFileCreate(videoFile)
		}
	}

	// Add to database all new subtitle files
	for _, subFile := range subFiles {
		// If film is not in database
		if !db.IsSubtitlePathPresent(subFile) {
			handleFileCreate(subFile)
		}
	}

	// Get all films from volume
	films := db.GetFilmsFromVolume(volume.ID)
	for _, film := range films {
		for _, volumeFile := range film.VolumeFiles {
			// If the film is not in the volume files, remove this film
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
