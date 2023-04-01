package business

import (
	"regexp"
	"strconv"

	"github.com/Agurato/starfin/internal/model"
	"github.com/pariz/gountries"
	"golang.org/x/exp/slices"
)

// Filterer holds the different filters that can be applied
type Filterer interface {
	AddFilms(films []model.Film)
	AddFilm(films *model.Film)
	ParseParamsFilters(params string) (yearFilter string, years []int, genre, country string, page int, err error)
	GetCountryName(code string) string

	GetCountries() []string
	GetDecades() []model.Decade
	GetGenres() []string
}

type FiltererWrapper struct {
	minReleaseYear int
	maxReleaseYear int
	Decades        []model.Decade

	Genres    []string
	Countries []string

	paramsRegex *regexp.Regexp
}

func NewFiltererWrapper() *FiltererWrapper {
	var (
		paramsYearRegex    string = `(\/year\/(?P<year>\d{4}s?))?`
		paramsGenreRegex   string = `(\/genre\/(?P<genre>[a-zA-Z\s]+))?`
		paramsCountryRegex string = `(\/country\/(?P<country>[a-zA-Z\s]+))?`
		paramsPageRegex    string = `(\/page\/(?P<page>\d{1,}))?`
	)

	fw := &FiltererWrapper{
		paramsRegex: regexp.MustCompile(paramsYearRegex + paramsGenreRegex + paramsCountryRegex + paramsPageRegex),
	}

	return fw
}

// AddFilm adds release year if new min or max, and missing genres to the filter
func (f *FiltererWrapper) AddFilms(films []model.Film) {
	for _, film := range films {
		f.addToYears(film.ReleaseYear)
		f.addToGenres(film.Genres)
		f.addToCountries(film.ProdCountries)
	}
	f.computeDecades()
	slices.Sort(f.Genres)
	slices.SortFunc(f.Countries, func(a, b string) bool {
		return f.GetCountryName(a) < f.GetCountryName(b)
	})
}

// AddFilm adds release year if new min or max, and missing genres to the filter
func (f *FiltererWrapper) AddFilm(film *model.Film) {
	if f.addToYears(film.ReleaseYear) {
		f.computeDecades()
	}
	f.addToGenres(film.Genres)
	slices.Sort(f.Genres)
	f.addToCountries(film.ProdCountries)
	slices.SortFunc(f.Countries, func(a, b string) bool {
		return f.GetCountryName(a) < f.GetCountryName(b)
	})
}

// ParseParamsFilters parses a params string and returns the filtered years, genre, country and page number
func (f *FiltererWrapper) ParseParamsFilters(params string) (yearFilter string, years []int, genre, country string, page int, err error) {
	submatches := f.paramsRegex.FindStringSubmatch(params)
	for i, captureName := range f.paramsRegex.SubexpNames() {
		if captureName == "year" {
			yearFilter = submatches[i]
		} else if captureName == "genre" {
			genre = submatches[i]
		} else if captureName == "country" {
			country = submatches[i]
		} else if captureName == "page" {
			pageMatch := submatches[i]
			if pageMatch == "" {
				page = 1
			} else {
				page, err = strconv.Atoi(submatches[i])
				if err != nil {
					return
				}
			}
		}
	}
	if len(yearFilter) == 5 {
		decade, _ := strconv.Atoi(yearFilter[:4])
		for i := decade; i < decade+10; i++ {
			years = append(years, i)
		}
	} else if len(yearFilter) == 4 {
		yearInt, _ := strconv.Atoi(yearFilter)
		years = append(years, yearInt)
	}
	return yearFilter, years, genre, country, page, nil
}

func (f *FiltererWrapper) GetCountryName(code string) string {
	country, _ := gountries.New().FindCountryByAlpha(code)
	return country.Name.Common
}

func (f *FiltererWrapper) GetCountries() []string {
	return f.Countries
}

func (f *FiltererWrapper) GetDecades() []model.Decade {
	return f.Decades
}

func (f *FiltererWrapper) GetGenres() []string {
	return f.Genres
}

func (f *FiltererWrapper) computeDecades() {
	var decade model.Decade
	for i := f.maxReleaseYear; i >= f.minReleaseYear; i-- {
		decade.DecadeYear = (i / 10) * 10
		decade.Years = append(decade.Years, i)
		if i%10 == 0 || i == f.minReleaseYear {
			f.Decades = append(f.Decades, decade)
			decade = model.Decade{}
		}
	}
}

func (f *FiltererWrapper) addToYears(filmReleaseYear int) bool {
	computeDecades := false
	if f.minReleaseYear == 0 || f.minReleaseYear > filmReleaseYear {
		f.minReleaseYear = filmReleaseYear
		computeDecades = true
	}
	if f.maxReleaseYear == 0 || f.maxReleaseYear < filmReleaseYear {
		f.maxReleaseYear = filmReleaseYear
		computeDecades = true
	}
	return computeDecades
}

func (f *FiltererWrapper) addToGenres(filmGenres []string) {
	for _, genre := range filmGenres {
		if !slices.Contains(f.Genres, genre) {
			f.Genres = append(f.Genres, genre)
		}
	}
}

func (f *FiltererWrapper) addToCountries(filmProdCountries []string) {
	for _, country := range filmProdCountries {
		if !slices.Contains(f.Countries, country) {
			f.Countries = append(f.Countries, country)
		}
	}
}
