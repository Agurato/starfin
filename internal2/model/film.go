package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Film struct {
	ID          primitive.ObjectID `bson:"_id"`
	VolumeFiles []VolumeFile
	Name        string // Name fetched from filename
	Resolution  string // Resolution fetched from filename
	ReleaseYear int    // Release year fetched from filename
	TMDBID      int
	IMDbID      string

	// Fetched from online sources. Only these variables will be used by the template
	Title            string
	OriginalTitle    string
	Year             string
	Runtime          string
	Tagline          string
	Overview         string
	PosterPath       string
	BackdropPath     string
	Classification   string
	IMDbRating       string
	LetterboxdRating string
	Genres           []string
	Directors        []int64
	Writers          []int64
	Cast             []Cast
	ProdCountries    []string
}
