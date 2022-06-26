package server

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Agurato/starfin/internal/media"
	"github.com/radovskyb/watcher"
	log "github.com/sirupsen/logrus"
)

var (
	fileWatcher *watcher.Watcher

	watchedVolumes []*media.Volume
)

func InitFileWatching() (err error) {
	fileWatcher = watcher.New()
	// fileWatcher.FilterOps(watcher.Create, watcher.Rename)

	go fileWatchEventHandler()

	for _, v := range GetVolumes() {
		AddFileWatch(v)
	}

	if err := fileWatcher.Start(1 * time.Second); err != nil {
		log.Fatalln(err)
	}

	return nil
}

func CloseFileWatching() {
	fileWatcher.Close()
}

func AddFileWatch(v media.Volume) {
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
	watchedVolumes = append(watchedVolumes, &v)
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
				if info.ModTime().Unix() == modTime {
					log.Debugln("File has stopped writing", path)
					delete(fileWrites, path)
					ext := filepath.Ext(path)

					// Retrieve volume
					var volume *media.Volume
					for _, v := range watchedVolumes {
						if strings.HasPrefix(path, v.Path) {
							volume = v
						}
					}

					log.Debugf("File %s from volume %s", path, volume.Path)

					if media.IsVideoFileExtension(ext) {
						// Get subtitle files in same directory
						subs, err := GetRelatedSubFiles(path)
						if err != nil {
							log.WithFields(log.Fields{"error": err, "path": path}).Debugln("Cannot get related subtitle files")
						}
						mediaFile := media.CreateMediaFromFilename(path, volume.ID, subs)
						// Search ID on TMDB
						if mediaFile.FetchMediaID() != nil {
							log.WithFields(log.Fields{"file": path, "err": err}).Warningln("Unable to fetch movie ID from TMDB")
							continue
						}
						log.WithField("tmdbID", mediaFile.GetTMDBID()).Infoln("Found media with TMDB ID")

						// Fill info from TMDB
						mediaFile.FetchMediaDetails()

						// Add media to DB
						AddMediaToDB(&mediaFile)
						for _, personID := range mediaFile.GetCastAndCrewIDs() {
							if !IsPersonInDB(personID) {
								AddPersonToDB(media.FetchPersonDetails(personID))
							}
						}
					} else if media.IsSubtitleFileExtension(ext) {
						// TODO: Add subtitle to database
						log.WithField("path", path).Debugln("Should add subtitle file to an existing movie here")
					}
					continue
				} else {
					fileWrites[path] = info.ModTime().Unix()
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
				// TODO: If rename, get movie and release year and see if it changed compared to the one in DB
			} else if event.Op == watcher.Remove {
				// TODO
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

func SearchMediaFilesInVolume(volume media.Volume) {
	// Channel to add media to DB as they are fetched from TMDB
	mediaChan := make(chan media.Media)

	go volume.Scan(mediaChan)

	for {
		mediaFile, more := <-mediaChan
		if more {
			if IsMediaInDB(&mediaFile) {
				AddVolumeSourceToMedia(&mediaFile, &volume)
			} else {
				AddMediaToDB(&mediaFile)
			}

			for _, personID := range mediaFile.GetCastAndCrewIDs() {
				if !IsPersonInDB(personID) {
					AddPersonToDB(media.FetchPersonDetails(personID))
				}
			}
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
