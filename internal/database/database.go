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

	IsMoviePresent(MovieFile *media.Movie) bool
	AddMovie(MovieFile *media.Movie)
	AddVolumeSourceToMovie(MovieFile *media.Movie, volume *media.Volume)
	GetMovieFromPath(moviePath string) (movie *media.Movie, err error)
	ReplaceMoviePath(oldMoviePath, newMoviePath string, newMovie *media.Movie) error
	RemoveMovieFile(path string) error
	RemoveSubtitleFile(MoviePath, subtitlePath string) error

	IsPersonPresent(personID int64) bool
	AddPerson(person media.Person)
	AddActors(actors []media.Person)
	GetPersonFromID(TMDBID int64) (person media.Person, err error)

	GetMovies() (movies []media.Movie)
	GetMovieFromID(TMDBID int) (movie media.Movie, err error)
	GetMoviesWithActor(actorID int64) (movies []media.Movie)
	GetMoviesWithDirector(directorID int64) (movies []media.Movie)
	GetMoviesWithWriter(writerID int64) (movies []media.Movie)
	AddSubtitleToMoviePath(movieFilePath string, sub media.Subtitle) error
	GetMovieFromExternalSubtitle(subtitlePath string) (media.Movie, error)
}

// User is a user
type User struct {
	ID       primitive.ObjectID `bson:"_id"`
	Name     string             `bson:"name"`
	Password string             `bson:"password"`
	IsOwner  bool               `bson:"is_owner"`
	IsAdmin  bool               `bson:"is_admin"`
}
