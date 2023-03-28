package business

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Agurato/starfin/internal2/model"
	"github.com/radovskyb/watcher"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/exp/slices"
)

type FileStorer interface {
	GetVolumes() ([]model.Volume, error)

	AddSubtitleToFilmPath(filmFilePath string, sub model.Subtitle) error
	RemoveSubtitleFile(mediaPath, subtitlePath string) error

	GetFilmFromPath(filmPath string) (film *model.Film, err error)

	UpdateFilmVolumeFile(film *model.Film, oldPath string, newVolumeFile model.VolumeFile) error
	DeleteFilmVolumeFile(path string) error

	IsFilmPathPresent(filmPath string) bool
	IsSubtitlePathPresent(subPath string) bool
	GetFilmsFromVolume(id primitive.ObjectID) (films []model.Film)
}

type WatcherMetadataGetter interface {
	CreateFilm(file string, volumeID primitive.ObjectID, subFiles []string) *model.Film
	FetchFilmTMDBID(f *model.Film) error
	UpdateFilmDetails(film *model.Film)
}

type FileWatcher struct {
	FileStorer
	FilmManager
	WatcherMetadataGetter

	watcher        *watcher.Watcher
	watchedVolumes []*model.Volume
}

func NewFileWatcher(fs FileStorer, fm FilmManager, wmg WatcherMetadataGetter) *FileWatcher {
	fileWatcher := &FileWatcher{
		FileStorer:            fs,
		FilmManager:           fm,
		WatcherMetadataGetter: wmg,
		watcher:               watcher.New(),
	}

	go fileWatcher.eventListener()

	volumes, _ := fileWatcher.FileStorer.GetVolumes()
	for _, v := range volumes {
		fileWatcher.AddVolume(&v)
		fileWatcher.synchronizeFilesAndDB(&v)
	}

	return fileWatcher
}

func (fw *FileWatcher) Run() {
	if err := fw.watcher.Start(1 * time.Second); err != nil {
		log.Fatalln(err)
	}
}

func (fw *FileWatcher) Stop() {
	fw.watcher.Close()
}

func (fw *FileWatcher) AddVolume(v *model.Volume) {
	if v.IsRecursive {
		if err := fw.watcher.AddRecursive(v.Path); err != nil {
			log.WithFields(log.Fields{"path": v.Path, "error": err}).Errorln("Could not watch volume")
			return
		}
	} else {
		if err := fw.watcher.Add(v.Path); err != nil {
			log.WithFields(log.Fields{"path": v.Path, "error": err}).Errorln("Could not watch volume")
			return
		}
	}
	fw.watchedVolumes = append(fw.watchedVolumes, v)
}

// eventListener listens for file creation, renaming and deletion
func (fw *FileWatcher) eventListener() {
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

				if err := fw.handleFileCreate(path); err != nil {
					log.Errorln(err)
					continue
				}
			}
		// There is a new file event
		case event := <-fw.watcher.Event:
			log.WithField("event", event).Debugln("New file event")
			if event.Op == watcher.Create || event.Op == watcher.Write {
				// Add file if not a video or sub and if not already in map
				if _, ok := fileWrites[event.Path]; !ok && !event.IsDir() {
					ext := filepath.Ext(event.Path)
					// Add it to watch list if video or subtitle
					if model.IsVideoFileExtension(ext) || model.IsSubtitleFileExtension(ext) {
						fileWrites[event.Path] = 0
					}
				}
			} else if event.Op == watcher.Rename {
				fw.handleFileRenamed(event.OldPath, event.Path)
			} else if event.Op == watcher.Remove {
				fw.handleFileRemoved(event.Path)
			}
		// Error in file watching
		case err := <-fw.watcher.Error:
			log.WithField("error", err).Errorln("Error event")
			return
		// Stop watching files
		case <-fw.watcher.Closed:
			return
		}
	}
}

func (fw *FileWatcher) handleFileCreate(path string) error {
	ext := filepath.Ext(path)

	// Retrieve volume
	volume := fw.getVolumeFromFilePath(path)

	if model.IsVideoFileExtension(ext) { // Adding a video
		if err := fw.addFilmFromPath(path, volume.ID); err != nil {
			return err
		}
	} else if model.IsSubtitleFileExtension(ext) { // Adding a subtitle
		// Get related media file and subtitle struct
		mediaPaths, subtitle := fw.getRelatedMediaFiles(path)
		for _, mediaPath := range mediaPaths {
			// Add it to the database
			err := fw.FileStorer.AddSubtitleToFilmPath(mediaPath, *subtitle)
			if err != nil {
				log.WithFields(log.Fields{"subtitle": path, "media": mediaPath, "error": err}).Error("Cannot add subtitle to media")
			}
		}
	}

	return nil
}

func (fw *FileWatcher) handleFileRenamed(oldPath, newPath string) error {
	ext := filepath.Ext(newPath)
	// Add it to watch list if video or subtitle
	if model.IsVideoFileExtension(ext) {
		volume := fw.getVolumeFromFilePath(newPath)

		// Get related subtitles
		subFiles, err := fw.getRelatedSubFiles(newPath)
		if err != nil {
			log.WithField("path", newPath).Errorln("Error with file rename: could not get related subtitles")
		}
		// Create film
		newFilm := fw.WatcherMetadataGetter.CreateFilm(newPath, volume.ID, subFiles)
		err = fw.WatcherMetadataGetter.FetchFilmTMDBID(newFilm)
		if err != nil {
			log.WithFields(log.Fields{"path": newPath, "error": err}).Errorln("Error with file rename: could not get TMDB ID")
			// TODO
		}

		// Get the current film struct from mongo
		oldFilm, err := fw.FileStorer.GetFilmFromPath(oldPath)
		if err != nil {
			return errors.New("could not get film from path")
		}

		if oldFilm.TMDBID == newFilm.TMDBID {
			// If they have the same TMDB ID, replace the correct volumeFile
			if err = fw.FileStorer.UpdateFilmVolumeFile(oldFilm, oldPath, newFilm.VolumeFiles[0]); err != nil {
				log.WithFields(log.Fields{"oldPath": oldPath}).Errorln(err)
			}
		} else {
			// If they don't have the same TMDB ID, remove the path from the previous film
			if err := fw.FileStorer.DeleteFilmVolumeFile(oldPath); err != nil {
				return err
			}

			// Fetch film details and add it to the database
			if err := fw.addFilmFromPath(newPath, volume.ID); err != nil {
				return err
			}
		}
	} else if model.IsSubtitleFileExtension(ext) {
		// Remove old subtitle
		mediaPaths, _ := fw.getRelatedMediaFiles(oldPath)
		for _, mediaPath := range mediaPaths {
			fw.FileStorer.RemoveSubtitleFile(mediaPath, oldPath)
		}

		// Add new subtitle
		mediaPaths, subtitle := fw.getRelatedMediaFiles(newPath)
		for _, mediaPath := range mediaPaths {
			// Add it to the database
			err := fw.FileStorer.AddSubtitleToFilmPath(mediaPath, *subtitle)
			if err != nil {
				log.WithFields(log.Fields{"subtitle": newPath, "media": mediaPath, "error": err}).Error("Cannot add subtitle to media")
			}
		}
	}
	return nil
}

// handleFileRemoved handles the media and subtitle file removing
func (fw *FileWatcher) handleFileRemoved(path string) {
	ext := filepath.Ext(path)
	if model.IsVideoFileExtension(ext) { // If we're deleting a video
		if err := fw.FileStorer.DeleteFilmVolumeFile(path); err != nil {
			log.Errorln(err)
		}
	} else if model.IsSubtitleFileExtension(ext) { // If we're deleting a subtitle
		// Get related media file
		mediaPaths, _ := fw.getRelatedMediaFiles(path)
		for _, mediaPath := range mediaPaths {
			fw.FileStorer.RemoveSubtitleFile(mediaPath, path)
		}
	}
}

// SynchronizeFilesAndDB synchronizes the database to the current files in the volume
// It adds the missing films and subtitles from the database, and removes the films and subtitles
// that are not currently in the volume
func (fw *FileWatcher) synchronizeFilesAndDB(volume *model.Volume) {
	videoFiles, subFiles, err := volume.ListVideoFiles()
	if err != nil {
		log.WithField("volume", volume.Path).Errorln("Could not synchronize volume with database")
	}

	// Add to database all new video files
	for _, videoFile := range videoFiles {
		// If film is not in database
		if !fw.FileStorer.IsFilmPathPresent(videoFile) {
			fw.handleFileCreate(videoFile)
		}
	}

	// Add to database all new subtitle files
	for _, subFile := range subFiles {
		// If film is not in database
		if !fw.FileStorer.IsSubtitlePathPresent(subFile) {
			fw.handleFileCreate(subFile)
		}
	}

	// Get all films from volume
	films := fw.FileStorer.GetFilmsFromVolume(volume.ID)
	for _, film := range films {
		for _, volumeFile := range film.VolumeFiles {
			// If the film is not in the volume files, remove this film
			if !slices.Contains(videoFiles, volumeFile.Path) {
				fw.handleFileRemoved(volumeFile.Path)
			}
			// If the subtitle is not in the volume files, remove this subtitle
			for _, sub := range volumeFile.ExtSubtitles {
				if !slices.Contains(subFiles, sub.Path) {
					fw.handleFileRemoved(sub.Path)
				}
			}
		}
	}
}

// getRelatedMediaFiles returns a related media file, and the subtitle struct for a given subtitle file path
func (fw *FileWatcher) getRelatedMediaFiles(subFilePath string) (mediaPath []string, sub *model.Subtitle) {
	dir := filepath.Dir(subFilePath)
	subFileBase := filepath.Base(subFilePath)
	subFileBase = subFileBase[:strings.IndexRune(subFileBase, '.')]
	// Get all files which have the same filename beginning
	matches, err := filepath.Glob(filepath.Join(dir, subFileBase+"*"))
	if err != nil {
		return nil, nil
	}
	subFilesFilter := []string{subFilePath}
	for _, m := range matches {
		// If it's a video file, get its external subtitles with the current subtitle file path as a filter
		if model.IsVideoFileExtension(filepath.Ext(m)) {
			subtitles := model.GetExternalSubtitles(m, subFilesFilter)
			// If the media has a subtitle, then it's related
			if len(subtitles) > 0 {
				mediaPath = append(mediaPath, m)
				sub = &subtitles[0]
			}
		}
	}

	return
}

// getRelatedSubFiles returns a list of subtitle file paths for a given film file path
func (fw *FileWatcher) getRelatedSubFiles(filmFilePath string) (subs []string, err error) {
	dir := filepath.Dir(filmFilePath)
	filmFileBase := filepath.Base(filmFilePath)
	filmFileNoExt := filmFileBase[:len(filmFileBase)-len(filepath.Ext(filmFileBase))]
	matches, err := filepath.Glob(filepath.Join(dir, filmFileNoExt+"*"))
	if err != nil {
		return subs, err
	}
	for _, m := range matches {
		if model.IsSubtitleFileExtension(filepath.Ext(m)) {
			subs = append(subs, m)
		}
	}
	return subs, nil
}

func (fw *FileWatcher) getVolumeFromFilePath(path string) *model.Volume {
	for _, v := range fw.watchedVolumes {
		if strings.HasPrefix(path, v.Path) {
			return v
		}
	}
	return nil
}

// addFilmFromPath adds a film from its path and the volume
func (fw *FileWatcher) addFilmFromPath(path string, volumeID primitive.ObjectID) error {
	// Get subtitle files in same directory
	subs, err := fw.getRelatedSubFiles(path)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "path": path}).Debugln("Cannot get related subtitle files")
	}
	film := fw.WatcherMetadataGetter.CreateFilm(path, volumeID, subs)
	// Search ID on TMDB
	if err := fw.WatcherMetadataGetter.FetchFilmTMDBID(film); err != nil {
		log.WithFields(log.Fields{"file": path, "error": err}).Warningln("Unable to fetch film ID from TMDB")
		film.Title = film.Name
	} else {
		log.WithField("tmdbID", film.TMDBID).Infoln("Found media with TMDB ID")
		// Fill info from TMDB
		fw.WatcherMetadataGetter.UpdateFilmDetails(film)
	}

	// Add media to DB
	if err = fw.FilmManager.AddFilm(film, false); err != nil {
		log.WithField("path", film.VolumeFiles[0].Path).Errorln(err)
	}

	return nil
}