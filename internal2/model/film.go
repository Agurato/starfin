package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Film struct {
	ID          primitive.ObjectID `bson:"_id"`
	VolumeFiles []VolumeFile       `bson:"volume_file"`
	Name        string             `bson:"name"`         // Name fetched from filename
	Resolution  string             `bson:"resolution"`   // Resolution fetched from filename
	ReleaseYear int                `bson:"release_year"` // Release year fetched from filename
	TMDBID      int                `bson:"tmdb_id"`
	IMDbID      string             `bson:"imdb_id"`

	// Fetched from online sources. Only these variables will be used by the template
	Title            string   `bson:"title"`
	OriginalTitle    string   `bson:"original_title"`
	Year             string   `bson:"year"`
	Runtime          string   `bson:"runtime"`
	Tagline          string   `bson:"tagline"`
	Overview         string   `bson:"overview"`
	PosterPath       string   `bson:"poster_path"`
	BackdropPath     string   `bson:"backdrop_path"`
	Classification   string   `bson:"classification"`
	IMDbRating       string   `bson:"imdb_rating"`
	LetterboxdRating string   `bson:"letterboxd_rating"`
	Genres           []string `bson:"genres"`
	Directors        []int64  `bson:"directors"`
	Writers          []int64  `bson:"writers"`
	Cast             []Cast   `bson:"cast"`
	ProdCountries    []string `bson:"prod_countries"`
}
