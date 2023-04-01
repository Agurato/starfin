package model

import (
	"path/filepath"
	"strings"
)

// IsVideoFileExtension checks if extension is corresponding to a known video file extension
// See https://en.wikipedia.org/wiki/Video_file_format
func IsVideoFileExtension(ext string) bool {
	ext = strings.ToLower(ext)
	if ext == ".mkv" ||
		ext == ".mp4" || ext == ".m4p" || ext == ".m4v" ||
		ext == ".mpg" || ext == ".mp2" || ext == ".mpeg" || ext == ".mpe" || ext == ".mpv" || ext == ".m2v" ||
		ext == ".avi" ||
		ext == ".webm" ||
		ext == ".flv" || ext == ".f4v" || ext == ".f4p" || ext == ".f4a" || ext == ".f4b" ||
		ext == ".vob" ||
		ext == ".ogv" || ext == ".ogg" ||
		ext == ".mts" || ext == ".m2ts" || ext == ".ts" ||
		ext == ".mov" ||
		ext == ".wmv" ||
		ext == ".yuv" ||
		ext == ".asf" {
		return true
	}
	return false
}

// IsSubtitleFileExtension checks if extension is corresponding to a known subtitle file extension
// See https://en.wikipedia.org/wiki/Category:Subtitle_file_formats
func IsSubtitleFileExtension(ext string) bool {
	ext = strings.ToLower(ext)

	if ext == ".srt" ||
		ext == ".ssa" || ext == ".ass" ||
		ext == ".sub" || ext == ".idx" ||
		ext == ".smi" || ext == ".sami" ||
		ext == ".smil" ||
		ext == ".usf" ||
		ext == ".psb" ||
		ext == ".ssd" ||
		ext == ".vtt" {
		return true
	}

	return false
}

// Fetches external subtitle files next to the media file.
// Subtitles file names must start with the media file name without extension. Ex:
// media: BigBuckBunny.mkv
// sub:   BigBuckBunny.srt
// Language can be inferred from the subtitle file name, following ISO 639-1 codes. Ex:
// sub:   BigBuckBunny.en.srt
func GetExternalSubtitles(filmFilePath string, subFiles []string) (subs []Subtitle) {
	dir := filepath.Dir(filmFilePath)
	filmFileBase := filepath.Base(filmFilePath)
	filmFileNoExt := filmFileBase[:len(filmFileBase)-len(filepath.Ext(filmFileBase))]

	for _, subFile := range subFiles {
		// Checks same folder
		if !strings.HasPrefix(subFile, dir) {
			continue
		}
		subFileBase := filepath.Base(subFile)
		// Checks sub file start with the same name as film file
		if !strings.HasPrefix(subFileBase, filmFileNoExt) {
			continue
		}

		subFileEnd := subFileBase[len(filmFileNoExt):] // eg: .en.srt
		subFileExt := filepath.Ext(subFileBase)
		// If no language info
		if len(subFileEnd) == len(subFileExt) {
			subs = append(subs, Subtitle{
				Path: subFile,
			})
		} else {
			subs = append(subs, Subtitle{
				Path:     subFile,
				Language: subFileEnd[1 : len(subFileEnd)-len(subFileExt)],
			})
		}
	}

	return
}
