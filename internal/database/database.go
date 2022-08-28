package database

import (
	"github.com/Agurato/starfin/internal/media"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DB interface {
	Close()

	IsOwnerPresent() bool

	AddUser(username, password1, password2 string, isAdmin bool) error
	DeleteUser(hexId string) error
	GetUserFromID(id primitive.ObjectID, user *User) error
	GetUserFromName(username string, user *User) error
	GetUserNb() int64
	GetUsers() (users []User)
	SetUserPassword(userID primitive.ObjectID, newPassword string) error

	GetVolumeFromID(id primitive.ObjectID, volume *media.Volume) error
	GetVolumes() (volumes []media.Volume)
	AddVolume(volume media.Volume) error
	DeleteVolume(hexId string) error

	IsMediaPresent(mediaFile *media.Media) bool
	AddMedia(mediaFile *media.Media)
	AddVolumeSourceToMedia(mediaFile *media.Media, volume *media.Volume)
	GetMediaFromPath(mediaPath string) (media.Media, error)
	ReplaceMediaPath(oldMediaPath, newMediaPath string, newMedia *media.Media) error
	AddSubtitleToMoviePath(movieFilePath string, sub media.Subtitle) error
	RemoveMediaFile(path string) error
	RemoveSubtitleFile(mediaPath, subtitlePath string) error

	IsPersonPresent(personID int64) bool
	AddPerson(person media.Person)
	AddActors(actors []media.Person)
	GetPersonFromID(TMDBID int64) (person media.Person, err error)

	GetMovies() (movies []media.Movie)
	GetMovieFromID(TMDBID int) (movie media.Movie, err error)
	GetMoviesWithActor(actorID int64) (movies []media.Movie)
	GetMoviesWithDirector(directorID int64) (movies []media.Movie)
	GetMoviesWithWriter(writerID int64) (movies []media.Movie)
}

// User is a user
type User struct {
	ID       primitive.ObjectID `bson:"_id"`
	Name     string             `bson:"name"`
	Password string             `bson:"password"`
	IsOwner  bool               `bson:"is_owner"`
	IsAdmin  bool               `bson:"is_admin"`
}
