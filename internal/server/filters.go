package server

import (
	"regexp"
	"strconv"

	"github.com/Agurato/starfin/internal/media"
	"golang.org/x/exp/slices"
)

var (
	minReleaseYear, maxReleaseYear int
	filters                        Filters

	paramsRegex *regexp.Regexp = regexp.MustCompile(`\/(year\/(?P<year>\d{4}s?)\/)?(genre\/(?P<genre>[a-zA-Z\s]+)\/)?(country\/(?P<country>[a-zA-Z\s]+)\/)?(page\/(?P<page>\d{1,})\/)?`)
)

// Filters holds the different filters that can be applied
type Filters struct {
	Decades   []Decade
	Genres    []string
	Countries []string
}

type Decade struct {
	DecadeYear int
	Years      []int
}

func (f *Filters) ComputeDecades() {
	var decade Decade
	for i := maxReleaseYear; i >= minReleaseYear; i-- {
		decade.DecadeYear = (i / 10) * 10
		decade.Years = append(decade.Years, i)
		if i%10 == 0 || i == minReleaseYear {
			f.Decades = append(f.Decades, decade)
			decade = Decade{}
		}
	}
}

// initFilters initializes the list of decades and genres used to filter
func initFilters() {
	films := db.GetFilms()
	for _, film := range films {
		if film.ReleaseYear != 0 {
			if minReleaseYear == 0 || minReleaseYear > film.ReleaseYear {
				minReleaseYear = film.ReleaseYear
			}
			if maxReleaseYear == 0 || maxReleaseYear < film.ReleaseYear {
				maxReleaseYear = film.ReleaseYear
			}
		}
		for _, genre := range film.Genres {
			if !slices.Contains(filters.Genres, genre) {
				filters.Genres = append(filters.Genres, genre)
			}
		}
		for _, country := range film.ProdCountries {
			if !slices.Contains(filters.Countries, country) {
				filters.Countries = append(filters.Countries, country)
			}
		}
	}
	slices.Sort(filters.Genres)
	slices.SortFunc(filters.Countries, func(a, b string) bool {
		return getCountryName(a) < getCountryName(b)
	})
	filters.ComputeDecades()
}

// addToFilters adds release year if new min or max, and missing genres to the filter
func addToFilters(film *media.Film) {
	computeDecades := false
	if minReleaseYear == 0 || minReleaseYear > film.ReleaseYear {
		minReleaseYear = film.ReleaseYear
		computeDecades = true
	}
	if maxReleaseYear == 0 || maxReleaseYear < film.ReleaseYear {
		maxReleaseYear = film.ReleaseYear
		computeDecades = true
	}
	if computeDecades {
		filters.ComputeDecades()
	}
	for _, genre := range film.Genres {
		if !slices.Contains(filters.Genres, genre) {
			filters.Genres = append(filters.Genres, genre)
		}
	}
	for _, country := range film.ProdCountries {
		if !slices.Contains(filters.Countries, country) {
			filters.Countries = append(filters.Countries, country)
		}
	}
	slices.Sort(filters.Genres)
	slices.SortFunc(filters.Countries, func(a, b string) bool {
		return getCountryName(a) < getCountryName(b)
	})
}

func ParseParamsFilters(params string) (yearFilter string, years []int, genre, country string, page int, err error) {
	submatches := paramsRegex.FindStringSubmatch(params)
	for i, captureName := range paramsRegex.SubexpNames() {
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
