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
	IsMoviePathPresent(moviePath string) bool
	IsSubtitlePathPresent(subPath string) bool
	AddMovie(MovieFile *media.Movie) error
	AddVolumeSourceToMovie(MovieFile *media.Movie) error
	GetMovieFromPath(moviePath string) (movie *media.Movie, err error)
	UpdateMovieVolumeFile(movie *media.Movie, oldPath string, newVolumeFile media.VolumeFile) error
	DeleteMovie(id primitive.ObjectID) error
	DeleteMovieVolumeFile(path string) error
	RemoveSubtitleFile(MoviePath, subtitlePath string) error

	IsPersonPresent(personID int64) bool
	AddPerson(person media.Person)
	AddActors(actors []media.Person)
	GetPersonFromID(TMDBID int64) (person media.Person, err error)

	GetMovieFromID(id primitive.ObjectID) (movie media.Movie, err error)
	GetMovies() (movies []media.Movie)
	GetMoviesFromVolume(id primitive.ObjectID) (movies []media.Movie)
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
