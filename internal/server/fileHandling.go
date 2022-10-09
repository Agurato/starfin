package server

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/Agurato/starfin/internal/media"
	log "github.com/sirupsen/logrus"
)

func handleFileCreate(path string) error {
	ext := filepath.Ext(path)

	// Retrieve volume
	volume := getVolumeFromFilePath(path)

	if media.IsVideoFileExtension(ext) { // Adding a video
		if err := addFilmFromPath(path, volume.ID); err != nil {
			return err
		}
	} else if media.IsSubtitleFileExtension(ext) { // Adding a subtitle
		// Get related media file and subtitle struct
		mediaPaths, subtitle := getRelatedMediaFiles(path)
		for _, mediaPath := range mediaPaths {
			// Add it to the database
			err := db.AddSubtitleToFilmPath(mediaPath, *subtitle)
			if err != nil {
				log.WithFields(log.Fields{"subtitle": path, "media": mediaPath, "error": err}).Error("Cannot add subtitle to media")
			}
		}
	}

	return nil
}

func handleFileRenamed(oldPath, newPath string) error {
	ext := filepath.Ext(newPath)
	// Add it to watch list if video or subtitle
	if media.IsVideoFileExtension(ext) {
		volume := getVolumeFromFilePath(newPath)

		// Get related subtitles
		subFiles, err := getRelatedSubFiles(newPath)
		if err != nil {
			log.WithField("path", newPath).Errorln("Error with file rename: could not get related subtitles")
		}
		// Create film
		newFilm := media.NewFilm(newPath, volume.ID, subFiles)
		err = newFilm.FetchTMDBID()
		if err != nil {
			log.WithFields(log.Fields{"path": newPath, "error": err}).Errorln("Error with file rename: could not get TMDB ID")
			// TODO
		}

		// Get the current film struct from mongo
		oldFilm, err := db.GetFilmFromPath(oldPath)
		if err != nil {
			return errors.New("could not get film from path")
		}

		if oldFilm.TMDBID == newFilm.TMDBID {
			// If they have the same TMDB ID, replace the correct volumeFile
			if err = db.UpdateFilmVolumeFile(oldFilm, oldPath, newFilm.VolumeFiles[0]); err != nil {
				log.WithFields(log.Fields{"oldPath": oldPath}).Errorln(err)
			}
		} else {
			// If they don't have the same TMDB ID, remove the path from the previous film
			if err := db.DeleteFilmVolumeFile(oldPath); err != nil {
				return err
			}

			// Fetch film details and add it to the database
			if err := addFilmFromPath(newPath, volume.ID); err != nil {
				return err
			}
		}
	} else if media.IsSubtitleFileExtension(ext) {
		// Remove old subtitle
		mediaPaths, _ := getRelatedMediaFiles(oldPath)
		for _, mediaPath := range mediaPaths {
			db.RemoveSubtitleFile(mediaPath, oldPath)
		}

		// Add new subtitle
		mediaPaths, subtitle := getRelatedMediaFiles(newPath)
		for _, mediaPath := range mediaPaths {
			// Add it to the database
			err := db.AddSubtitleToFilmPath(mediaPath, *subtitle)
			if err != nil {
				log.WithFields(log.Fields{"subtitle": newPath, "media": mediaPath, "error": err}).Error("Cannot add subtitle to media")
			}
		}
	}
	return nil
}

// handleFileRemoved handles the media and subtitle file removing
func handleFileRemoved(path string) {
	ext := filepath.Ext(path)
	if media.IsVideoFileExtension(ext) { // If we're deleting a video
		if err := db.DeleteFilmVolumeFile(path); err != nil {
			log.Errorln(err)
		}
	} else if media.IsSubtitleFileExtension(ext) { // If we're deleting a subtitle
		// Get related media file
		mediaPaths, _ := getRelatedMediaFiles(path)
		for _, mediaPath := range mediaPaths {
			db.RemoveSubtitleFile(mediaPath, path)
		}
	}
}

// getRelatedSubFiles returns a list of subtitle file paths for a given film file path
func getRelatedSubFiles(filmFilePath string) (subs []string, err error) {
	dir := filepath.Dir(filmFilePath)
	filmFileBase := filepath.Base(filmFilePath)
	filmFileNoExt := filmFileBase[:len(filmFileBase)-len(filepath.Ext(filmFileBase))]
	matches, err := filepath.Glob(filepath.Join(dir, filmFileNoExt+"*"))
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

// getRelatedMediaFiles returns a related media file, and the subtitle struct for a given subtitle file path
func getRelatedMediaFiles(subFilePath string) (mediaPath []string, sub *media.Subtitle) {
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
		if media.IsVideoFileExtension(filepath.Ext(m)) {
			subtitles := media.GetExternalSubtitles(m, subFilesFilter)
			// If the media has a subtitle, then it's related
			if len(subtitles) > 0 {
				mediaPath = append(mediaPath, m)
				sub = &subtitles[0]
			}
		}
	}

	return
}
