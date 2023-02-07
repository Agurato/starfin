package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Volume holds the volume paths to fetch media from
type Volume struct {
	ID          primitive.ObjectID `bson:"_id"`
	Name        string             `bson:"name"`
	Path        string             `bson:"path"`
	IsRecursive bool               `bson:"is_recursive"`
	MediaType   string             `bson:"media_type"`
}

type VolumeFile struct {
	Path         string             `bson:"path"`
	FromVolume   primitive.ObjectID `bson:"from_volume"`
	Info         MediaInfo          `bson:"info"`
	ExtSubtitles []Subtitle         `bson:"ext_subtitles"`
}

type Subtitle struct {
	Language string `bson:"language"`
	Path     string `bson:"path"`
}
