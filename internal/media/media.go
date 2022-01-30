package media

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Media interface {
	FetchMediaID() error
	FetchMediaDetails()
}

// CreateMediaFromFilename instantiates a struct implementing the Media interface
// Currently only handles Movies
// TODO: TVSeries
func CreateMediaFromFilename(file string, volumeID primitive.ObjectID) Media {
	filename := filepath.Base(file)
	movie := Movie{
		ID:   primitive.NewObjectID(),
		Path: file,
	}
	// Split on '.' and ' '
	parts := strings.FieldsFunc(filename, func(r rune) bool {
		return r == '.' || r == ' '
	})
	i := len(parts) - 1

	// Iterate in reverse and stop at first year info
	for ; i >= 0; i-- {
		potentialYear := parts[i]
		if len(potentialYear) == 4 {
			year, err := strconv.Atoi(potentialYear)
			if err == nil {
				movie.ReleaseYear = year
				break
			}
		}
		if len(potentialYear) == 6 && potentialYear[0] == '(' && potentialYear[5] == ')' {
			year, err := strconv.Atoi(potentialYear[1:5])
			if err == nil {
				movie.ReleaseYear = year
				break
			}
		}
	}

	// The movie name should be right before the movie year
	if movie.ReleaseYear > 0 && i >= 0 {
		movie.Name = strings.Join(parts[:i], " ")
	} else {
		movie.Name = strings.Join(parts, " ")
	}
	fmt.Println("Found movie", movie.Name, movie.ReleaseYear)

	movie.FromVolumes = append(movie.FromVolumes, volumeID)
	return &movie
}

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
