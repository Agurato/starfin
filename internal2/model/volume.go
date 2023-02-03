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
	Path         string
	FromVolume   primitive.ObjectID
	Info         MediaInfo
	ExtSubtitles []Subtitle
}

type Subtitle struct {
	Language string
	Path     string
}
