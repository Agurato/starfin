package server

import (
	"regexp"
	"strconv"

	"github.com/Agurato/starfin/internal/media"
	"github.com/Agurato/starfin/internal2/model"
	"github.com/pariz/gountries"
	"golang.org/x/exp/slices"
)

// Filters holds the different filters that can be applied
type Filters struct {
	minReleaseYear int
	maxReleaseYear int
	Decades        []Decade

	Genres    []string
	Countries []string

	paramsRegex *regexp.Regexp
}

type Decade struct {
	DecadeYear int
	Years      []int
}

func NewFilters(films []model.Film) *Filters {
	var (
		paramsYearRegex    string = `(\/year\/(?P<year>\d{4}s?))?`
		paramsGenreRegex   string = `(\/genre\/(?P<genre>[a-zA-Z\s]+))?`
		paramsCountryRegex string = `(\/country\/(?P<country>[a-zA-Z\s]+))?`
		paramsPageRegex    string = `(\/page\/(?P<page>\d{1,}))?`
	)

	filters := &Filters{
		paramsRegex: regexp.MustCompile(paramsYearRegex + paramsGenreRegex + paramsCountryRegex + paramsPageRegex),
	}

	for _, film := range films {
		filters.addToYears(film.ReleaseYear)
		filters.addToGenres(film.Genres)
		filters.addToCountries(film.ProdCountries)
	}
	filters.computeDecades()
	slices.Sort(filters.Genres)
	slices.SortFunc(filters.Countries, func(a, b string) bool {
		return filters.GetCountryName(a) < filters.GetCountryName(b)
	})

	return filters
}

// AddToFilters adds release year if new min or max, and missing genres to the filter
func (f *Filters) AddToFilters(film *media.Film) {
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
func (f *Filters) ParseParamsFilters(params string) (yearFilter string, years []int, genre, country string, page int, err error) {
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

func (f *Filters) GetCountryName(code string) string {
	country, _ := gountries.New().FindCountryByAlpha(code)
	return country.Name.Common
}

func (f *Filters) computeDecades() {
	var decade Decade
	for i := f.maxReleaseYear; i >= f.minReleaseYear; i-- {
		decade.DecadeYear = (i / 10) * 10
		decade.Years = append(decade.Years, i)
		if i%10 == 0 || i == f.minReleaseYear {
			f.Decades = append(f.Decades, decade)
			decade = Decade{}
		}
	}
}

func (f *Filters) addToYears(filmReleaseYear int) bool {
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

func (f *Filters) addToGenres(filmGenres []string) {
	for _, genre := range filmGenres {
		if !slices.Contains(f.Genres, genre) {
			f.Genres = append(f.Genres, genre)
		}
	}
}

func (f *Filters) addToCountries(filmProdCountries []string) {
	for _, country := range filmProdCountries {
		if !slices.Contains(f.Countries, country) {
			f.Countries = append(f.Countries, country)
		}
	}
}
