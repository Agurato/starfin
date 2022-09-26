package database

import (
	"github.com/Agurato/starfin/internal/media"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DB interface {
	Close()

	IsOwnerPresent() bool

	AddUser(user *User) error
	DeleteUser(hexId string) error
	IsUsernameAvailable(username string) (bool, error)
	GetUserFromID(id primitive.ObjectID, user *User) error
	GetUserFromName(username string, user *User) error
	GetUserNb() (int64, error)
	GetUsers() (users []User, err error)
	SetUserPassword(userID primitive.ObjectID, newPassword string) error

	GetVolumeFromID(id primitive.ObjectID, volume *media.Volume) error
	GetVolumes() (volumes []media.Volume, err error)
	AddVolume(volume *media.Volume) error
	DeleteVolume(hexId string) error

	IsFilmPresent(FilmFile *media.Film) bool
	IsFilmPathPresent(filmPath string) bool
	IsSubtitlePathPresent(subPath string) bool
	AddFilm(FilmFile *media.Film) error
	AddVolumeSourceToFilm(FilmFile *media.Film) error
	GetFilmFromPath(filmPath string) (film *media.Film, err error)
	UpdateFilmVolumeFile(film *media.Film, oldPath string, newVolumeFile media.VolumeFile) error
	DeleteFilm(id primitive.ObjectID) error
	DeleteFilmVolumeFile(path string) error
	RemoveSubtitleFile(FilmPath, subtitlePath string) error

	IsPersonPresent(personID int64) bool
	AddPerson(person media.Person)
	AddActors(actors []media.Person)
	GetPersonFromID(TMDBID int64) (person media.Person, err error)
	GetPeople() (people []media.Person)

	GetFilmFromID(id primitive.ObjectID) (film media.Film, err error)
	GetFilmCount() int64
	GetFilms() (films []media.Film)
	GetFilmsRange(start, number int) (films []media.Film)
	GetFilmsFromVolume(id primitive.ObjectID) (films []media.Film)
	GetFilmsWithActor(actorID int64) (films []media.Film)
	GetFilmsWithDirector(directorID int64) (films []media.Film)
	GetFilmsWithWriter(writerID int64) (films []media.Film)
	AddSubtitleToFilmPath(filmFilePath string, sub media.Subtitle) error
	GetFilmFromExternalSubtitle(subtitlePath string) (media.Film, error)
}

// User is a user
type User struct {
	ID       primitive.ObjectID `bson:"_id"`
	Name     string             `bson:"name"`
	Password string             `bson:"password"`
	IsOwner  bool               `bson:"is_owner"`
	IsAdmin  bool               `bson:"is_admin"`
}
