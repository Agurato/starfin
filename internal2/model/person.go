package model

import (
	"html/template"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Cast struct {
	Character string
	ActorID   int64
}

type Person struct {
	ID       primitive.ObjectID `bson:"_id"`
	TMDBID   int64
	Name     string
	Photo    string
	Bio      template.HTML
	Birthday string
	Deathday string
	IMDbID   string
}
