package model

import "time"

type RarbgTorrent struct {
	Hash     string    `bson:"hash"`
	Title    string    `bson:"title"`
	DT       time.Time `bson:"dt"`
	Category string    `bson:"cat"`
	Size     *int64    `bson:"size"`
	IMDbID   *string   `bson:"imdb"`
}
