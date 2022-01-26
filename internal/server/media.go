package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tmdb "github.com/cyruzin/golang-tmdb"
	"go.mongodb.org/mongo-driver/bson"
)

// MediaInfo contains the information about one item from the search results
type MediaInfo struct {
	ID        int64
	MediaType string
	MediaURL  string
	Title     string
	Date      string
	Language  string
	Poster    string
	Backdrop  string

	Action string
}

// Media actions
const (
	MediaActionAdd    string = "add"
	MediaActionRemove string = "remove"

	MediaMovie string = "movie"
	MediaTV    string = "tv"
)

var (
	// TMDBClient holds the client for tmdb access
	TMDBClient *tmdb.Client
)

// InitTMDB initializes a tmdb client
func InitTMDB() {
	var err error
	TMDBClient, err = tmdb.Init(os.Getenv(EnvTMDBAPIKey))
	if err != nil {
		panic(err)
	}
}

// MediaSearchMulti returns an array of media informations corresponding to search query on TMDB
func MediaSearchMulti(searchQuery string, user User) ([]MediaInfo, error) {
	userColl := mongoDb.Collection("users")
	var searchResults []MediaInfo

	tmdbSearchRes, err := TMDBClient.GetSearchMulti(searchQuery, nil)
	if err != nil {
		return searchResults, err
	}

	for _, res := range tmdbSearchRes.Results {
		action := MediaActionAdd
		var userDB User
		if err := userColl.FindOne(MongoCtx, bson.M{"_id": user.ID}).Decode(&userDB); err != nil {
			// TODO
		}

		switch res.MediaType {
		case "movie":
			title := res.Title
			if res.OriginalLanguage == "fr" {
				title = res.OriginalTitle
			} else if res.OriginalLanguage != "en" && res.Title != res.OriginalTitle {
				title = fmt.Sprintf("%s (%s)", res.Title, res.OriginalTitle)
			}
			posterPath := ""
			if len(res.PosterPath) > 0 {
				posterPath = tmdb.GetImageURL(res.PosterPath, "original")
			}
			searchResults = append(searchResults, MediaInfo{
				ID:        res.ID,
				MediaType: "Movie",
				MediaURL:  fmt.Sprintf("/movie/%d", res.ID),
				Title:     title,
				Date:      strings.Split(res.ReleaseDate, "-")[0],
				Language:  res.OriginalLanguage,
				Poster:    posterPath,
				Action:    action,
			})
		case "tv":
			title := res.Name
			if res.OriginalLanguage == "fr" {
				title = res.OriginalName
			} else if res.OriginalLanguage != "en" && res.Name != res.OriginalName {
				title = fmt.Sprintf("%s (%s)", res.Name, res.OriginalName)
			}
			posterPath := ""
			if len(res.PosterPath) > 0 {
				posterPath = tmdb.GetImageURL(res.PosterPath, "original")
			}
			searchResults = append(searchResults, MediaInfo{
				ID:        res.ID,
				MediaType: "TV",
				MediaURL:  fmt.Sprintf("/tv/%d", res.ID),
				Title:     title,
				Date:      strings.Split(res.FirstAirDate, "-")[0],
				Language:  res.OriginalLanguage,
				Poster:    posterPath,
				Action:    action,
			})
		}
	}

	return searchResults, nil
}

// MediaMovieDetails fetches movie details from TMDB
func MediaMovieDetails(tmdbID int, inUserQueries bool) (MediaInfo, error) {
	info := MediaInfo{}

	details, err := TMDBClient.GetMovieDetails(tmdbID, nil)
	if err != nil {
		return info, err
	}
	title := details.Title
	if details.OriginalLanguage == "fr" {
		title = details.OriginalTitle
	} else if details.OriginalLanguage != "en" && details.Title != details.OriginalTitle {
		title = fmt.Sprintf("%s (%s)", details.Title, details.OriginalTitle)
	}
	posterPath := ""
	if len(details.PosterPath) > 0 {
		posterPath = tmdb.GetImageURL(details.PosterPath, "original")
	}
	backdropPath := ""
	if len(details.PosterPath) > 0 {
		backdropPath = tmdb.GetImageURL(details.BackdropPath, "original")
	}

	action := MediaActionAdd
	if inUserQueries {
		action = MediaActionRemove
	}

	return MediaInfo{
		ID:        details.ID,
		MediaType: "Movie",
		Title:     title,
		Date:      strings.Split(details.ReleaseDate, "-")[0],
		Language:  details.OriginalLanguage,
		Poster:    posterPath,
		Backdrop:  backdropPath,
		Action:    action,
	}, nil
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

// ListVideoFiles lists all the files that are considered as video files in a folder
// See func isVideoFileExtension(string)
func ListVideoFiles(root string, recursive bool) ([]string, error) {
	var (
		files      []string
		videoFiles []string
		err        error
	)
	if recursive {
		// files, err = filepath.Glob(path + "/**/*.mkv")

		err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
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
		// files, err = filepath.Glob(path + "/*.mkv")

		f, err := os.Open(root)
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
				files = append(files, filepath.Join(root, fileInfo.Name()))
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

// ScanVolume scans files from volume that have not been added to the db yet
func ScanVolume(volume Volume) {
	files, err := ListVideoFiles(volume.Path, volume.IsRecursive)
	if err != nil {
		// TODO: log
	}

	// releaseDateRegex, _ := regexp.Compile(`(\d\d\d\d)`)

	// For each file
	for _, file := range files {
		// Get movie name
		movieName, releaseDate := GetMovieInfoFromFileName(filepath.Base(file))
		fmt.Println(movieName)
		fmt.Println(releaseDate)

		// Search on TMDB

		// Add to DB

		// TODO: log
	}
}

// GetMovieInfoFromFileName returns movie name and release year inferred from the file name
// TODO: unit test
func GetMovieInfoFromFileName(filename string) (movieName string, releaseYear int) {

	// Split on '.' and ' '
	parts := strings.Split(strings.ReplaceAll(filename, ".", " "), " ")
	i := len(parts) - 1

	// Iterate in reverse and stop at first year info
	for ; i >= 0; i-- {
		potentialYear := parts[i]
		if len(potentialYear) == 4 {
			year, err := strconv.Atoi(potentialYear)
			if err == nil {
				releaseYear = year
				break
			}
		}
		if len(potentialYear) == 6 && potentialYear[0] == '(' && potentialYear[5] == ')' {
			year, err := strconv.Atoi(potentialYear[1:5])
			if err == nil {
				releaseYear = year
				break
			}
		}
	}

	// The movie name should be right before the movie year
	if releaseYear > 0 && i >= 0 {
		movieName = strings.Join(parts[:i], " ")
	} else {
		movieName = strings.Join(parts, " ")
	}
	return
}
