package server

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Agurato/starfin/internal/media"
	"github.com/radovskyb/watcher"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	fileWatcher *watcher.Watcher

	watchedVolumes []*media.Volume
)

func InitFileWatching() (err error) {
	fileWatcher = watcher.New()
	// fileWatcher.FilterOps(watcher.Create, watcher.Rename)

	go fileWatchEventHandler()

	volumes, err := db.GetVolumes()
	if err != nil {
		return err
	}
	for _, v := range volumes {
		AddFileWatch(&v)
	}

	if err := fileWatcher.Start(1 * time.Second); err != nil {
		log.Fatalln(err)
	}

	return nil
}

func CloseFileWatching() {
	fileWatcher.Close()
}

func AddFileWatch(v *media.Volume) {
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
				ext := filepath.Ext(path)

				// Retrieve volume
				volume := getVolumeFromFilePath(path)

				if media.IsVideoFileExtension(ext) { // If we're adding a video
					if err := AddMovieFromPath(path, volume.ID); err != nil {
						log.Errorln(err)
						continue
					}
				} else if media.IsSubtitleFileExtension(ext) { // If we're adding a subtitle
					// Get related media file and subtitle struct
					mediaPath, subtitle, ok := GetRelatedMediaFile(path)
					if ok {
						// Add it to the database
						err := db.AddSubtitleToMoviePath(mediaPath, *subtitle)
						if err != nil {
							log.WithFields(log.Fields{"subtitle": path, "media": mediaPath, "error": err}).Error("Cannot add subtitle to media")
						}
					}
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
				ext := filepath.Ext(event.Path)
				// Add it to watch list if video or subtitle
				if media.IsVideoFileExtension(ext) {
					volume := getVolumeFromFilePath(event.Path)

					// Get related subtitles
					subFiles, err := GetRelatedSubFiles(event.Path)
					if err != nil {
						log.WithField("path", event.Path).Errorln("Error with file rename: could not get related subtitles")
					}
					// Create media
					newMovie := media.NewMovie(event.Path, volume.ID, subFiles)
					err = newMovie.FetchTMDBID()
					if err != nil {
						log.WithFields(log.Fields{"path": event.Path, "error": err}).Errorln("Error with file rename: could not get TMDB ID")
						// TODO
					}
					err = db.ReplaceMoviePath(event.OldPath, event.Path, newMovie)
					if err != nil {
						log.WithFields(log.Fields{"path": event.Path, "error": err}).Errorln("Error with file rename: could not replace media path")
						// TODO
					}
				} else if media.IsSubtitleFileExtension(ext) {
					// TODO
					// Get media this subtitle was attached to
					// movie, err := db.GetMovieFromExternalSubtitle(event.OldPath)
					// if err != nil {

					// }
				}
			} else if event.Op == watcher.Remove {
				ext := filepath.Ext(event.Path)
				if media.IsVideoFileExtension(ext) { // If we're deleting a video
					if err := db.RemoveMovieFile(event.Path); err != nil {
						log.Errorln(err)
					}
				} else if media.IsSubtitleFileExtension(ext) { // If we're deleting a subtitle
					// Get related media file
					mediaPath, _, ok := GetRelatedMediaFile(event.Path)
					if ok {
						db.RemoveSubtitleFile(mediaPath, event.Path)
					}
				}
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

// AddMovieFromPath adds a movie from its path and the volume
func AddMovieFromPath(path string, volumeID primitive.ObjectID) error {
	// Get subtitle files in same directory
	subs, err := GetRelatedSubFiles(path)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "path": path}).Debugln("Cannot get related subtitle files")
	}
	movie := media.NewMovie(path, volumeID, subs)
	// Search ID on TMDB
	if err := movie.FetchTMDBID(); err != nil {
		log.WithFields(log.Fields{"file": path, "error": err}).Warningln("Unable to fetch movie ID from TMDB")
	} else {
		log.WithField("tmdbID", movie.TMDBID).Infoln("Found media with TMDB ID")
		// Fill info from TMDB
		movie.FetchDetails()
	}

	// Add media to DB
	TryAddMovieToDB(movie)

	return nil
}

// TryAddMovieToDB checks if the movie already exists in database before adding it
// Also adds persons to the database if they don't exist
func TryAddMovieToDB(movie *media.Movie) {
	if movie.TMDBID == 0 {
		if err := db.AddMovie(movie); err != nil {
			log.WithField("path", movie.VolumeFiles[0].Path).Warningln("Cannot add movie to database")
		}
	} else if !db.IsMoviePresent(movie) {
		db.AddMovie(movie)
	} else {
		db.AddVolumeSourceToMovie(movie)
	}

	for _, personID := range movie.GetCastAndCrewIDs() {
		if !db.IsPersonPresent(personID) {
			db.AddPerson(media.FetchPersonDetails(personID))
		}
	}
}

func SearchMediaFilesInVolume(volume *media.Volume) {
	// Channel to add media to DB as they are fetched from TMDB
	movieChan := make(chan *media.Movie)

	go volume.Scan(movieChan)

	for {
		movie, more := <-movieChan
		if more {
			TryAddMovieToDB(movie)
		} else {
			break
		}
	}
}

func GetRelatedSubFiles(movieFilePath string) (subs []string, err error) {
	dir := filepath.Dir(movieFilePath)
	movieFileBase := filepath.Base(movieFilePath)
	movieFileNoExt := movieFileBase[:len(movieFileBase)-len(filepath.Ext(movieFileBase))]
	matches, err := filepath.Glob(filepath.Join(dir, movieFileNoExt+"*"))
	if err != nil {
		return subs, err
	}
	for _, m := range matches {
		if media.IsSubtitleFileExtension(filepath.Ext(m)) {
			subs = append(subs, m)
		}
	}
	return subs, nil
}

func GetRelatedMediaFile(subFilePath string) (mediaPath string, sub *media.Subtitle, ok bool) {
	dir := filepath.Dir(subFilePath)
	subFileBase := filepath.Base(subFilePath)
	subFileBase = subFileBase[:strings.IndexRune(subFileBase, '.')]
	matches, err := filepath.Glob(filepath.Join(dir, subFileBase+"*"))
	if err != nil {
		return "", nil, false
	}
	for _, m := range matches {
		if media.IsVideoFileExtension(filepath.Ext(m)) {
			subtitles := media.GetExternalSubtitles(m, []string{subFilePath})
			if len(subtitles) > 0 {
				return m, &subtitles[0], true
			}
		}
	}

	return "", nil, false
}
