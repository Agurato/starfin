package infrastructure

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/agnivade/levenshtein"
	tmdb "github.com/cyruzin/golang-tmdb"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/Agurato/starfin/internal/model"
)

type Metadata interface {
	GetPosterLink(key string) string
	GetBackdropLink(key string) string
	GetPhotoLink(key string) string

	GetTMDBIDFromLink(inputUrl string) (tmdbID int, err error)
	GetPersonDetails(personID int64) *model.Person
	CreateFilm(file string, volumeID primitive.ObjectID, subFiles []string) *model.Film
	FetchFilmTMDBID(f *model.Film) error
	UpdateFilmDetails(film *model.Film)
}

type MetadataWrapper struct {
	client *tmdb.Client
}

// NewMetadataWrapper initializes a MetadataWrapper
func NewMetadataWrapper(tmdbAPIKey string) (*MetadataWrapper, error) {
	client, err := tmdb.Init(tmdbAPIKey)
	if err != nil {
		return nil, err
	}
	return &MetadataWrapper{
		client: client,
	}, nil
}

const (
	poster       = "poster"
	backdrop     = "backdrop"
	photo        = "photo"
	tmdbImageURL = "https://image.tmdb.org/t/p/"
)

// GetPosterLink caches a poster from TMDB using its id
func (mw MetadataWrapper) GetPosterLink(key string) string {
	return tmdbImageURL + tmdb.W342 + key
}

// GetBackdropLink caches a backdrop from TMDB using its id
func (mw MetadataWrapper) GetBackdropLink(key string) string {
	return tmdbImageURL + tmdb.W1280 + key
}

// GetPhotoLink caches a person's photo from TMDB using its id
func (mw MetadataWrapper) GetPhotoLink(key string) string {
	return tmdbImageURL + tmdb.W342 + key
}

func (mw MetadataWrapper) CreateFilm(file string, volumeID primitive.ObjectID, subFiles []string) *model.Film {
	filename := filepath.Base(file)
	mediaInfo, err := mw.getMediaInfo(os.Getenv("MEDIAINFO_PATH"), file)
	if err != nil {
		log.Error().Str("file", file).Msg("Could not get media info")
	}
	subtitles := model.GetExternalSubtitles(file, subFiles)
	film := model.Film{
		ID: primitive.NewObjectID(),
		VolumeFiles: []model.VolumeFile{{
			Path:         file,
			FromVolume:   volumeID,
			Info:         mediaInfo,
			ExtSubtitles: subtitles,
		}},
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
				film.ReleaseYear = year
				break
			}
		}
		if len(potentialYear) == 6 && potentialYear[0] == '(' && potentialYear[5] == ')' {
			year, err := strconv.Atoi(potentialYear[1:5])
			if err == nil {
				film.ReleaseYear = year
				break
			}
		}
	}
	// The film name should be right before the film year
	if film.ReleaseYear > 0 && i >= 0 {
		film.Name = strings.Join(parts[:i], " ")
	} else {
		film.Name = strings.Join(parts, " ")
	}

	// Get resolution from name
	resolutionPRegex, _ := regexp.Compile(`^\d\d\d\d?[pP]$`)
	resolutionKRegex, _ := regexp.Compile(`^\d[kK]$`)
	for i := len(parts) - 1; i >= 0; i-- {
		potentialRes := parts[i]
		if resolutionPRegex.MatchString(potentialRes) || resolutionKRegex.MatchString(potentialRes) {
			film.Resolution = potentialRes
			break
		}
	}
	// If resolution not found, get it from MediaInfo video
	if film.Resolution == "" {
		film.Resolution = mediaInfo.Resolution
	}

	return &film
}

// FetchFilmTMDBID fetches media ID from TMDB and stores it
func (mw MetadataWrapper) FetchFilmTMDBID(f *model.Film) error {
	urlOptions := make(map[string]string)
	if f.ReleaseYear != 0 {
		urlOptions["year"] = strconv.Itoa(f.ReleaseYear)
	}
	tmdbSearchRes, err := mw.client.GetSearchMovies(f.Name, urlOptions)
	if err != nil {
		return err
	}
	if len(tmdbSearchRes.Results) == 0 {
		return errors.New("film not found")
	}

	mostPopular := float32(0)
	for _, res := range tmdbSearchRes.Results {
		if res.Popularity > mostPopular {
			// Levenshtein distance so that the name corresponds at least a little bit
			if levenshtein.ComputeDistance(f.Name, res.Title) < len(f.Name)/3 || mostPopular == 0 {
				f.TMDBID = int(res.ID)
				mostPopular = res.Popularity
			}
		}
	}
	return nil
}

func (mw MetadataWrapper) UpdateFilmDetails(film *model.Film) {
	// Get details
	details, err := mw.client.GetMovieDetails(film.TMDBID, nil)
	if err != nil {
		log.Error().Err(err).Int("tmdbID", film.TMDBID).Msg("Unable to fetch film details from TMDB")
	}
	film.IMDbID = details.IMDbID
	film.Title = details.Title
	film.OriginalTitle = details.OriginalTitle
	film.Year = details.ReleaseDate[:4]
	film.Runtime = strconv.Itoa(details.Runtime)
	film.Tagline = details.Tagline
	film.Overview = details.Overview
	film.PosterPath = details.PosterPath
	film.BackdropPath = details.BackdropPath
	film.IMDbRating = mw.getIMDbRating(film.IMDbID)
	film.LetterboxdRating = mw.getLetterboxdRating(film.IMDbID)

	// Set genres
	film.Genres = nil
	for _, genre := range details.Genres {
		film.Genres = append(film.Genres, genre.Name)
	}

	// Set classification
	releaseDates, err := mw.client.GetMovieReleaseDates(film.TMDBID)
	if err != nil {
		log.Error().Err(err).Int("tmdbID", film.TMDBID).Msg("Unable to fetch film release dates from TMDB")
	} else {
		for _, releasesCountry := range releaseDates.Results {
			if releasesCountry.Iso3166_1 == "US" {
				film.Classification = releasesCountry.ReleaseDates[0].Certification
				break
			}
		}
	}

	// Set cast and crew
	credits, err := mw.client.GetMovieCredits(film.TMDBID, nil)
	if err != nil {
		log.Error().Err(err).Int("tmdbID", film.TMDBID).Msg("Unable to fetch film credits from TMDB")
	} else {
		film.Directors = nil
		film.Writers = nil
		film.Characters = nil
		for _, crew := range credits.Crew {
			if crew.Job == "Director" {
				film.Directors = append(film.Directors, crew.ID)
			}
			if crew.Department == "Writing" {
				if !slices.Contains(film.Writers, crew.ID) {
					film.Writers = append(film.Writers, crew.ID)
				}
			}
		}
		for _, cast := range credits.Cast {
			film.Characters = append(film.Characters, model.Character{CharacterName: cast.Character, ActorID: cast.ID})
		}
	}

	film.ProdCountries = nil
	// Set production countries
	for _, country := range details.ProductionCountries {
		film.ProdCountries = append(film.ProdCountries, country.Iso3166_1)
	}
}

func (mw MetadataWrapper) getMediaInfo(mediaInfoPath, filePath string) (model.MediaInfo, error) {
	var mediaInfo model.MediaInfo
	var mediaInfoJSONOutput model.MediaInfoJSONOutput

	out, err := exec.Command(mediaInfoPath, filePath).Output()
	if err != nil {
		return mediaInfo, err
	}
	fullOutput := strings.ReplaceAll(string(out), "\r\n", "\n")
	fullOutput = strings.Trim(fullOutput, "\n")
	var fullOutputLines []string
	for _, line := range strings.Split(fullOutput, "\n") {
		if strings.HasPrefix(line, "Complete name") {
			fullOutputLines = append(fullOutputLines, fmt.Sprintf("Name : %s", filepath.Base(strings.Split(line, " : ")[1])))
		} else {
			fullOutputLines = append(fullOutputLines, line)
		}
	}
	mediaInfo.FullOutput = template.HTML(strings.Join(fullOutputLines, "<br>"))

	out, err = exec.Command(mediaInfoPath, "--Output=JSON", filePath).Output()
	if err != nil {
		return mediaInfo, err
	}
	json.Unmarshal(out, &mediaInfoJSONOutput)

	// For every track
	for _, track := range mediaInfoJSONOutput.Media.Track {
		switch track["@type"] {
		// Fill General info
		case "General":
			mediaInfo.Format = track["Format"]
			totalSeconds, _ := strconv.ParseFloat(track["Duration"], 32)
			hours := int(totalSeconds / 3600)
			minutes := int(totalSeconds/60) % 60
			seconds := int(totalSeconds) % 60
			mediaInfo.Duration = fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
			totalSize, _ := strconv.Atoi(track["FileSize"])
			if totalSize > 1_000_000_000 {
				mediaInfo.FileSize = fmt.Sprintf("%.2f GB", float64(totalSize)/1_000_000_000)
			} else if totalSize > 1_000_000 {
				mediaInfo.FileSize = fmt.Sprintf("%.2f MB", float64(totalSize)/1_000_000)
			} else if totalSize > 1_000 {
				mediaInfo.FileSize = fmt.Sprintf("%.2f KB", float64(totalSize)/1_000)
			}
		// Fill Video info
		case "Video":
			mediaInfo.Video = append(mediaInfo.Video, model.VideoInfo{
				CodecID:    track["CodecID"],
				Profile:    track["Format_Profile"],
				Resolution: fmt.Sprintf("%sx%s", track["Width"], track["Height"]),
				FrameRate:  track["FrameRate"],
				BitDepth:   track["BitDepth"],
			})
			// Compute resolution on first video stream
			if mediaInfo.Resolution == "" {
				// Switch on the width because film can have black horizontal bars
				width, _ := strconv.Atoi(track["Width"])
				switch width {
				case 720:
					mediaInfo.Resolution = "480p"
				case 1280:
					mediaInfo.Resolution = "720p"
				case 1920:
					mediaInfo.Resolution = "1080p"
				case 2560:
					mediaInfo.Resolution = "1440p"
				case 3840:
					mediaInfo.Resolution = "4K"
				case 7680:
					mediaInfo.Resolution = "8K"
				}
			}
		// Fill Audio info
		case "Audio":
			mediaInfo.Audio = append(mediaInfo.Audio, model.AudioInfo{
				CodecID:      track["CodecID"],
				Channels:     track["Channels"],
				Language:     track["Language"],
				SamplingRate: track["SamplingRate"],
			})
		// Fill Text info
		case "Text":
			mediaInfo.Subs = append(mediaInfo.Subs, model.SubsInfo{
				CodecID:  track["CodecID"],
				Language: track["Language"],
				Forced:   track["Forced"],
			})
		}
	}

	return mediaInfo, nil
}

// getIMDbRating fetchs rating from IMDbID
func (mw MetadataWrapper) getIMDbRating(imdbId string) string {
	client := http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://www.imdb.com/title/%s/", imdbId), nil)
	req.Header.Add("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Str("imdb_id", imdbId).Msg("Cannot fetch rating from IMDb")
		return ""
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		log.Error().Str("imdb_id", imdbId).Msg("Cannot fetch rating from IMDb")
		return ""
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Error().Str("imdb_id", imdbId).Msg("Cannot fetch rating from IMDb")
		return ""
	}

	return doc.Find("#__next > main > div > section.ipc-page-background.ipc-page-background--base > section > div:nth-child(4) > section > section > div > div > div > div > div > div:nth-child(1) > a > span > div > div > div > span:nth-child(1)").First().Text()
}

// getLetterboxdRating fetchs rating from letterboxd using IMDbID
func (mw MetadataWrapper) getLetterboxdRating(imdbId string) string {
	resp, err := http.Get(fmt.Sprintf("https://letterboxd.com/search/films/%s/", imdbId))
	if err != nil {
		log.Error().Str("imdb_id", imdbId).Msg("Cannot fetch rating from Letterboxd")
		return ""
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		log.Error().Str("imdb_id", imdbId).Msg("Cannot fetch rating from Letterboxd")
		return ""
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Error().Str("imdb_id", imdbId).Msg("Cannot fetch rating from Letterboxd")
		return ""
	}

	filmUrl, exists := doc.Find("#content > div > div > section > ul > li:nth-child(1) > div").First().Attr("data-target-link")
	if !exists {
		log.Error().Str("imdb_id", imdbId).Msg("Cannot fetch rating from Letterboxd")
		return ""
	}

	resp, err = http.Get(fmt.Sprintf("https://letterboxd.com/csi%srating-histogram/", filmUrl))
	if err != nil {
		log.Error().Str("imdb_id", imdbId).Msg("Cannot fetch rating from Letterboxd")
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Error().Str("imdb_id", imdbId).Msg("Cannot fetch rating from Letterboxd")
		return ""
	}
	doc, err = goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Error().Str("imdb_id", imdbId).Msg("Cannot fetch rating from Letterboxd")
		return ""
	}

	return doc.Find("a.display-rating").First().Text()
}

// GetPersonDetails fetches details about a person from TMDB
func (mw MetadataWrapper) GetPersonDetails(personID int64) *model.Person {
	details, err := mw.client.GetPersonDetails(int(personID), nil)
	if err != nil {
		log.Error().Int64("personID", personID).Err(err).Send()
		return &model.Person{
			ID:     primitive.NewObjectID(),
			TMDBID: personID,
		}
	}

	return &model.Person{
		ID:       primitive.NewObjectID(),
		TMDBID:   personID,
		Name:     details.Name,
		Photo:    details.ProfilePath,
		Bio:      template.HTML(strings.ReplaceAll(details.Biography, "\n", "<br>")),
		Birthday: details.Birthday,
		Deathday: details.Deathday,
		IMDbID:   details.IMDbID,
	}
}

// GetTMDBIDFromLink returns the TMDB ID from a TMDB, IMDb, or Letterboxd URL
func (mw MetadataWrapper) GetTMDBIDFromLink(inputUrl string) (tmdbID int, err error) {
	urlParsed, err := url.Parse(inputUrl)
	if err != nil {
		return tmdbID, err
	}
	switch urlParsed.Host {
	case "www.themoviedb.org":
		tmdbID, err = mw.getTMDBIDFromTheMovieDB(inputUrl)
	case "www.imdb.com":
		tmdbID, err = mw.getTMDBIDFromIMDB(inputUrl)
	case "letterboxd.com":
		tmdbID, err = mw.getTMDBIDFromLetterboxd(inputUrl)
	default:
		err = errors.New("the host could not be found")
	}

	return tmdbID, err
}

// getTMDBIDFromTheMovieDB returns the TMDB ID from a TMDB URL
func (mw MetadataWrapper) getTMDBIDFromTheMovieDB(inputUrl string) (TMDBID int, err error) {
	err = nil
	// Parse URL
	urlParsed, err := url.Parse(inputUrl)
	if err != nil {
		return TMDBID, err
	}
	var tmdbIDStr string
	// TMDB movie url path should start with /movie/
	if strings.HasPrefix(urlParsed.Path, "/movie/") {
		if strings.HasSuffix(urlParsed.Path, "/") { // this type of link: https://www.themoviedb.org/movie/1817/
			tmdbIDStr = urlParsed.Path[7 : len(urlParsed.Path)-1]
		} else { // this type of link: https://www.themoviedb.org/movie/1817-phone-booth
			tmdbIDStr = strings.Split(urlParsed.Path[7:], "-")[0]
		}
		// TMDB ID shoudl be castable to an integer
		if TMDBID, err = strconv.Atoi(tmdbIDStr); err != nil {
			err = errors.New("could not parse TheMovieDB URL")
		}
	} else {
		err = errors.New("could not parse TheMovieDB URL")
	}
	return TMDBID, err
}

// getTMDBIDFromLetterboxd returns the TMDB ID from a Letterboxd URL
func (mw MetadataWrapper) getTMDBIDFromLetterboxd(inputUrl string) (TMDBID int, err error) {
	// Get the page's HTML
	res, err := http.Get(inputUrl)
	if err != nil {
		log.Error().Str("url", inputUrl).Msg("Cannot fetch TMDB ID from Letterboxd")
		return TMDBID, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return TMDBID, errors.New("cannot fetch TMDB ID from Letterboxd")
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return TMDBID, err
	}

	// Get TMDB anchor from the page
	tmdbUrl, exists := doc.Find("a[data-track-action=TMDb]").First().Attr("href")
	if !exists {
		return TMDBID, errors.New("cannot fetch TMDB ID from Letterboxd")
	}
	// Get TMDB ID from its URL
	return mw.getTMDBIDFromTheMovieDB(tmdbUrl)
}

// getTMDBIDFromIMDB returns the TMDB ID from an IMDb URL
func (mw MetadataWrapper) getTMDBIDFromIMDB(inputUrl string) (TMDBID int, err error) {
	// Parse URL
	urlParsed, err := url.Parse(inputUrl)
	if err != nil {
		return TMDBID, err
	}
	// IMDb movie URL path should start with /title/
	if strings.HasPrefix(urlParsed.Path, "/title/") {
		imdbID := urlParsed.Path[7 : len(urlParsed.Path)-1]
		// Get TMDB ID using the TMDB API
		tmdbIDInt64, err := mw.getTMDBIDFromIMDBID(imdbID)
		if err != nil {
			return TMDBID, err
		}
		TMDBID = int(tmdbIDInt64)
	} else {
		err = errors.New("cannot fetch TMDB ID from IMDB")
	}
	return
}

// getTMDBIDFromIMDBID retrieves the TMDB ID from an IMDb ID
func (mw MetadataWrapper) getTMDBIDFromIMDBID(imdbID string) (TMDBID int64, err error) {
	urlOptions := make(map[string]string)
	urlOptions["external_source"] = "imdb_id"
	res, err := mw.client.GetFindByID(imdbID, urlOptions)
	if err != nil {
		return TMDBID, err
	}
	TMDBID = res.MovieResults[0].ID
	return
}
