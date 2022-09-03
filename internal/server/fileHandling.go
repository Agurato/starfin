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
		if err := AddMovieFromPath(path, volume.ID); err != nil {
			return err
		}
	} else if media.IsSubtitleFileExtension(ext) { // Adding a subtitle

		// Get related media file and subtitle struct
		mediaPath, subtitle := getRelatedMediaFile(path)
		if mediaPath != "" {
			// Add it to the database
			err := db.AddSubtitleToMoviePath(mediaPath, *subtitle)
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
		// Create movie
		newMovie := media.NewMovie(newPath, volume.ID, subFiles)
		err = newMovie.FetchTMDBID()
		if err != nil {
			log.WithFields(log.Fields{"path": newPath, "error": err}).Errorln("Error with file rename: could not get TMDB ID")
			// TODO
		}

		// Get the current movie struct from mongo
		oldMovie, err := db.GetMovieFromPath(oldPath)
		if err != nil {
			return errors.New("could not get movie from path")
		}

		if oldMovie.TMDBID == newMovie.TMDBID {
			// If they have the same TMDB ID, replace the correct volumeFile
			if err = db.UpdateMovieVolumeFile(oldMovie, oldPath, newMovie.VolumeFiles[0]); err != nil {
				log.WithFields(log.Fields{"oldPath": oldPath}).Errorln(err)
			}
		} else {
			// If they don't have the same TMDB ID, remove the path from the previous movie
			db.DeleteMovieVolumeFile(oldPath)

			// Fetch movie details and add it to the database
			if err := AddMovieFromPath(newPath, volume.ID); err != nil {
				return err
			}
		}
	} else if media.IsSubtitleFileExtension(ext) {
		// TODO
		// Get media this subtitle was attached to
		// movie, err := db.GetMovieFromExternalSubtitle(oldPath)
		// if err != nil {

		// }
	}
	return nil
}

// handleFileRemoved handles the media and subtitle file removing
func handleFileRemoved(path string) {
	ext := filepath.Ext(path)
	if media.IsVideoFileExtension(ext) { // If we're deleting a video
		if err := db.DeleteMovieVolumeFile(path); err != nil {
			log.Errorln(err)
		}
	} else if media.IsSubtitleFileExtension(ext) { // If we're deleting a subtitle
		// Get related media file
		mediaPath, _ := getRelatedMediaFile(path)
		if mediaPath != "" {
			db.RemoveSubtitleFile(mediaPath, path)
		}
	}
}

// getRelatedSubFiles returns a list of subtitle file paths for a given movie file path
func getRelatedSubFiles(movieFilePath string) (subs []string, err error) {
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

// getRelatedMediaFile returns a related media file, and the subtitle struct for a given subtitle file path
func getRelatedMediaFile(subFilePath string) (mediaPath string, sub *media.Subtitle) {
	dir := filepath.Dir(subFilePath)
	subFileBase := filepath.Base(subFilePath)
	subFileBase = subFileBase[:strings.IndexRune(subFileBase, '.')]
	// Get all files which have the same filename beginning
	matches, err := filepath.Glob(filepath.Join(dir, subFileBase+"*"))
	if err != nil {
		return "", nil
	}
	subFilesFilter := []string{subFilePath}
	for _, m := range matches {
		// If it's a video file, get its external subtitles with the current subtitle file path as a filter
		if media.IsVideoFileExtension(filepath.Ext(m)) {
			subtitles := media.GetExternalSubtitles(m, subFilesFilter)
			// If the media has a subtitle, then it's related
			if len(subtitles) > 0 {
				return m, &subtitles[0]
			}
		}
	}

	return "", nil
}
