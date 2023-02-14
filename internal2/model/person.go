package model

import (
	"html/template"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Cast struct {
	CharacterName string
	Actor         Person
}

type Person struct {
	ID       primitive.ObjectID `bson:"_id"`
	TMDBID   int64              `bson:"tmdb_id"`
	Name     string             `bson:"name"`
	Photo    string             `bson:"photo"`
	Bio      template.HTML      `bson:"bio"`
	Birthday string             `bson:"birthday"`
	Deathday string             `bson:"deathday"`
	IMDbID   string             `bson:"imdb_id"`
}
