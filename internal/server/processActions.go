package server

import (
	"errors"
	"math"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/Agurato/starfin/internal/database"
	"github.com/Agurato/starfin/internal/media"
	"github.com/PuerkitoBio/goquery"
	"github.com/matthewhartstonge/argon2"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AddUser checks that the user and password follow specific rules and adds it to the database
func AddUser(username, password1, password2 string, isAdmin bool) (*database.User, error) {
	argon := argon2.DefaultConfig()

	// Check username length
	if len(username) < 2 || len(username) > 25 {
		return nil, errors.New("username must be between 2 and 25 characters")
	}

	// Check if username is not already taken
	if available, err := db.IsUsernameAvailable(username); err != nil {
		log.Errorln(err)
		return nil, errors.New("an error occured …")
	} else if !available {
		return nil, errors.New("this username is already taken")
	}

	// Check if both passwords are equal
	if password1 != password2 {
		return nil, errors.New("passwords don't match")
	}

	// Check if password is at least 8 characters
	if len(password1) < 8 {
		return nil, errors.New("passwords must be at least 8 characters long")
	}

	// Hash & encode password
	encoded, err := argon.HashEncoded([]byte(password1))
	if err != nil {
		return nil, errors.New("an error occured while creating your account")
	}

	// Add user to DB
	user := &database.User{
		ID:       primitive.NewObjectID(),
		Name:     username,
		Password: string(encoded),
		IsOwner:  !db.IsOwnerPresent(),
		IsAdmin:  isAdmin,
	}

	err = db.AddUser(user)
	if err != nil {
		log.Errorln(err)
		return nil, errors.New("user could not be added")
	}

	return user, nil
}

// CheckLogin checks that the login is correct and returns the user it corresponds to
func CheckLogin(username, password string) (user *database.User, err error) {
	// Check username length
	if len(username) < 2 || len(username) > 25 {
		return nil, errors.New("username must be between 2 and 25 characters")
	}

	// Fetch encoded password from DB
	user = &database.User{}
	if err := db.GetUserFromName(username, user); err != nil {
		return nil, errors.New("authentication failed")
	}

	// Check if the username/password combination is valid
	if ok, err := argon2.VerifyEncoded([]byte(password), []byte(user.Password)); err != nil {
		return nil, errors.New("an error occured while logging you in")
	} else if !ok {
		return nil, errors.New("authentication failed")
	}

	return user, nil
}

// SetUserPassword checks that the password change follows specific rules and updates it in the database
func SetUserPassword(username, oldPassword, password1, password2 string) error {
	argon := argon2.DefaultConfig()

	// Check new passwords match
	if password1 != password2 {
		return errors.New("new passwords don't match")
	}

	// Check password length
	if len(password1) < 8 {
		return errors.New("passwords must be at least 8 characters long")
	}

	// Fetch encoded password from DB
	var userDB database.User
	if err := db.GetUserFromName(username, &userDB); err != nil {
		return errors.New("an error occured while checking for your password")
	}

	// Check if the username/password combination is valid
	ok, err := argon2.VerifyEncoded([]byte(oldPassword), []byte(userDB.Password))
	if err != nil {
		return errors.New("an error occured while checking for your password")
	}
	if !ok {
		return errors.New("authentication failed")
	}

	// Hash & encode password
	encoded, err := argon.HashEncoded([]byte(password1))
	if err != nil {
		return errors.New("an error occured while saving your password")
	}

	if err := db.SetUserPassword(userDB.ID, string(encoded)); err != nil {
		return errors.New("an error occured while saving your password")
	}

	return nil
}

// AddVolume checks that the volume follows specific rules and adds it to the database
func AddVolume(volume *media.Volume) error {
	// Check volume name length
	if len(volume.Name) < 3 {
		return errors.New("volume name must be between 3")
	}

	// Check path is a directory
	fileInfo, err := os.Stat(volume.Path)
	if err != nil {
		return errors.New("volume path does not exist")
	}
	if !fileInfo.IsDir() {
		return errors.New("volume path is not a directory")
	}

	// Add volume to the database
	err = db.AddVolume(volume)
	if err != nil {
		log.Errorln(err)
		return errors.New("volume could not be added")
	}

	// Search for media files in a separate goroutine to return the page asap
	go searchMediaFilesInVolume(volume)

	// Add file watch to the volume
	addFileWatch(volume)

	return nil
}

// SearchFilms returns a sublist of films containing the search terms
// Searches in the title and original title (case-insensitive)
// Searches films from specific year (indicated by "y:XXXX" as the last part of the search)
func SearchFilms(search string, films []media.Film) []media.Film {
	search = strings.Trim(search, " ")
	specialChars := regexp.MustCompile("[.,\\/#!$%\\^&\\*;:{}=\\-_`~()%\\s\\\\]")
	var (
		filteredFilms []media.Film
	)

	search = specialChars.ReplaceAllString(strings.ToLower(search), "")
	for _, m := range films {
		title := specialChars.ReplaceAllString(strings.ToLower(m.Title), "")
		originalTitle := specialChars.ReplaceAllString(strings.ToLower(m.OriginalTitle), "")
		if strings.Contains(title, search) || strings.Contains(originalTitle, search) {
			filteredFilms = append(filteredFilms, m)
		}
	}

	return filteredFilms
}

type Pagination struct {
	Number int64
	Active bool
	Dots   bool
}

// getPagination creates a Pagination slice
func getPagination[T any](currentPage int64, items []T) ([]T, []Pagination) {
	var pages []Pagination
	pageMax := int64(math.Ceil(float64(len(items)) / float64(nbFilmsPerPage)))

	pages = append(pages, Pagination{
		Number: 1,
		Active: currentPage == 1,
	})
	// Add dots to link between 1 and current-1
	if currentPage > 3 {
		pages = append(pages, Pagination{
			Dots: true,
		})
	}
	for i := currentPage - 1; i <= currentPage+1; i++ {
		if i <= 1 || i >= pageMax {
			continue
		}
		if i == currentPage {
			pages = append(pages, Pagination{
				Number: i,
				Active: true,
			})
		} else {
			pages = append(pages, Pagination{
				Number: i,
			})
		}
	}
	// Add dots to link between current+1 and max
	if currentPage < pageMax-2 {
		pages = append(pages, Pagination{
			Dots: true,
		})
	}
	if pageMax > 1 {
		pages = append(pages, Pagination{
			Number: pageMax,
			Active: currentPage == pageMax,
		})
	}

	// Return only part of the items (corresponding to the current page)
	itemsIndexStart := (currentPage - 1) * nbFilmsPerPage
	itemsIndexEnd := itemsIndexStart + nbFilmsPerPage

	var pagedItems []T
	for i := itemsIndexStart; i < itemsIndexEnd && i < int64(len(items)); i++ {
		pagedItems = append(pagedItems, items[i])
	}

	return pagedItems, pages
}

// GetTMDBIDFromLink returns the TMDB ID from a TMDB, IMDb, or Letterboxd URL
func GetTMDBIDFromLink(inputUrl string) (tmdbID int, err error) {
	urlParsed, err := url.Parse(inputUrl)
	if err != nil {
		return tmdbID, err
	}
	switch urlParsed.Host {
	case "www.themoviedb.org":
		tmdbID, err = getTMDBIDFromTheMovieDB(inputUrl)
	case "www.imdb.com":
		tmdbID, err = getTMDBIDFromIMDB(inputUrl)
	case "letterboxd.com":
		tmdbID, err = getTMDBIDFromLetterboxd(inputUrl)
	default:
		err = errors.New("the host could not be found")
	}

	return tmdbID, err
}

// getTMDBIDFromTheMovieDB returns the TMDB ID from a TMDB URL
func getTMDBIDFromTheMovieDB(inputUrl string) (TMDBID int, err error) {
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
func getTMDBIDFromLetterboxd(inputUrl string) (TMDBID int, err error) {
	// Get the page's HTML
	res, err := http.Get(inputUrl)
	if err != nil {
		log.WithField("url", inputUrl).Errorln("Cannot fetch TMDB ID from Letterboxd")
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
	return getTMDBIDFromTheMovieDB(tmdbUrl)
}

// getTMDBIDFromIMDB returns the TMDB ID from an IMDb URL
func getTMDBIDFromIMDB(inputUrl string) (TMDBID int, err error) {
	// Parse URL
	urlParsed, err := url.Parse(inputUrl)
	if err != nil {
		return TMDBID, err
	}
	// IMDb movie URL path should start with /title/
	if strings.HasPrefix(urlParsed.Path, "/title/") {
		imdbID := urlParsed.Path[7 : len(urlParsed.Path)-1]
		// Get TMDB ID using the TMDB API
		tmdbIDInt64, err := media.GetTMDBIDFromIMDBID(imdbID)
		if err != nil {
			return TMDBID, err
		}
		TMDBID = int(tmdbIDInt64)
	} else {
		err = errors.New("cannot fetch TMDB ID from IMDB")
	}
	return
}
