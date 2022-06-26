package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Agurato/starfin/internal/media"
	"github.com/matthewhartstonge/argon2"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// User is a user
type User struct {
	ID       primitive.ObjectID `bson:"_id"`
	Name     string             `bson:"name"`
	Password string             `bson:"password"`
	IsOwner  bool               `bson:"is_owner"`
	IsAdmin  bool               `bson:"is_admin"`
}

var (
	MongoCtx     context.Context
	mongoDb      *mongo.Database
	mongoUsers   *mongo.Collection
	mongoVolumes *mongo.Collection
	mongoMovies  *mongo.Collection
	mongoPersons *mongo.Collection
)

// InitMongo init mongo db
func InitMongo() (mongoClient *mongo.Client) {
	MongoCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	mongoClient, err := mongo.Connect(MongoCtx,
		options.Client().ApplyURI(fmt.Sprintf("mongodb://%s:%s@%s:%s",
			os.Getenv(EnvDBUser),
			os.Getenv(EnvDBPassword),
			os.Getenv(EnvDBURL),
			os.Getenv(EnvDBPort))))
	if err != nil {
		log.Fatal(err)
	}

	mongoDb = mongoClient.Database(os.Getenv(EnvDBName))
	mongoUsers = mongoDb.Collection("users")
	mongoVolumes = mongoDb.Collection("volumes")
	mongoMovies = mongoDb.Collection("movies")
	mongoPersons = mongoDb.Collection("persons")
	return
}

func IsOwnerInDatabase() bool {
	countOwners, err := mongoUsers.CountDocuments(MongoCtx, bson.M{"is_owner": true})
	if err != nil {
		log.Errorln("Could not retrieve if owner is present in the database")
	}
	return countOwners > 0
}

// AddUser adds a user to the database after checking parameter
func AddUser(username, password1, password2 string, isAdmin bool) error {
	argon := argon2.DefaultConfig()

	// Check username length
	if len(username) < 2 || len(username) > 25 {
		return errors.New("username must be between 2 and 25 characters")
	}

	// Check if username is not already taken
	count, err := mongoUsers.CountDocuments(MongoCtx, bson.M{"name": primitive.Regex{Pattern: fmt.Sprintf("^%s$", username), Options: "i"}})
	if err != nil {
		log.WithFields(log.Fields{"name": username, "error": err}).Errorln("Unable to check if user exists")
	}
	if count > 0 {
		return errors.New("this username is already taken")
	}

	// Check if both passwords are equal
	if password1 != password2 {
		return errors.New("passwords don't match")
	}

	// Check if password is at least 8 characters
	if len(password1) < 8 {
		return errors.New("passwords must be at least 8 characters long")
	}

	// Hash & encode password
	encoded, err := argon.HashEncoded([]byte(password1))
	if err != nil {
		return errors.New("an error occured while creating your account")
	}

	// Add user to DB
	user := &User{
		ID:       primitive.NewObjectID(),
		Name:     username,
		Password: string(encoded),
		IsOwner:  !IsOwnerInDatabase(),
		IsAdmin:  isAdmin,
	}
	_, err = mongoUsers.InsertOne(MongoCtx, user)
	return err
}

// DeleteUser deletes the user from the DB
func DeleteUser(hexId string) error {
	userId, err := primitive.ObjectIDFromHex(hexId)
	if err != nil {
		return errors.New("invalid volume id")
	}
	res, err := mongoUsers.DeleteOne(MongoCtx, bson.M{"_id": userId})
	if err != nil {
		return err
	}
	if res.DeletedCount != 1 {
		return errors.New("unable to delete user")
	}

	return nil
}

// GetUserFromID gets user from its ID
func GetUserFromID(id primitive.ObjectID, user *User) error {
	return mongoUsers.FindOne(MongoCtx, bson.M{"_id": id}).Decode(&user)
}

// GetUserNb returns the number of users from the DB
func GetUserNb() int64 {
	count, err := mongoUsers.CountDocuments(MongoCtx, bson.M{})
	if err != nil {
		log.WithField("error", err).Errorln("Unable to retrieve number of users")
	}
	return count
}

// GetUsers returns the list of users in the DB
func GetUsers() (users []User) {
	usersCur, err := mongoUsers.Find(MongoCtx, bson.M{})
	if err != nil {
		log.WithField("error", err).Errorln("Unable to retrieve users from database")
	}
	for usersCur.Next(MongoCtx) {
		var user User
		err := usersCur.Decode(&user)
		if err != nil {
			log.WithField("error", err).Errorln("Unable to fetch user from database")
		}
		users = append(users, user)
	}
	return
}

// Fetches volume from DB using specified ID and returns it via pointer
func GetVolumeFromID(id primitive.ObjectID, volume *media.Volume) error {
	return mongoVolumes.FindOne(MongoCtx, bson.M{"_id": id}).Decode(&volume)
}

// GetVolumes returns the list of volumes in the DB
func GetVolumes() (volumes []media.Volume) {
	volumeCur, err := mongoVolumes.Find(MongoCtx, bson.M{})
	if err != nil {
		log.WithField("error", err).Errorln("Unable to retrieve volumes from database")
	}
	for volumeCur.Next(MongoCtx) {
		var vol media.Volume
		err := volumeCur.Decode(&vol)
		if err != nil {
			log.WithField("error", err).Errorln("Unable to fetch volume from database")
		}
		volumes = append(volumes, vol)
	}
	return
}

// AddVolume adds a volume to the DB and start scanning the volume
func AddVolume(volume media.Volume) error {
	// Check volume name length
	if len(volume.Name) < 3 {
		return errors.New("volume name must be between 3")
	}

	// Check path is a directory
	fileInfo, err := os.Stat(volume.Path)
	if err != nil {
		return errors.New("volume path does not exist")
	}
	if !fileInfo.IsDir() {
		return errors.New("volume path is not a directory")
	}

	_, err = mongoVolumes.InsertOne(MongoCtx, volume)
	if err == nil {
		// Search for media files in a separate goroutine to return the page asap
		go SearchMediaFilesInVolume(volume)

		// Add file watch to the volume
		AddFileWatch(volume)
	}
	return err
}

// DeleteVolume deletes the volume from the DB and all the media which originated only from this volume
func DeleteVolume(hexId string) error {
	volumeId, err := primitive.ObjectIDFromHex(hexId)
	if err != nil {
		return errors.New("invalid volume id")
	}

	// Remove specified volume from all media source
	// TODO: TV Series
	update, err := mongoMovies.UpdateMany(MongoCtx,
		bson.M{},
		bson.D{
			{Key: "$pull", Value: bson.D{{Key: "paths", Value: bson.D{{Key: "fromvolume", Value: volumeId}}}}},
		})
	if err != nil {
		return err
	}
	log.WithField("volumeId", hexId).Infof("%d movies are concerned with this volume deletion\n", update.ModifiedCount)
	del, err := mongoMovies.DeleteMany(MongoCtx, bson.M{"paths": bson.D{{Key: "$size", Value: 0}}})
	if err != nil {
		return err
	}
	log.WithField("volumeId", hexId).Infof("%d movies were removed from database\n", del.DeletedCount)

	// Remove specified volume from "volumes" collection
	res, err := mongoVolumes.DeleteOne(MongoCtx, bson.M{"_id": volumeId})
	if err != nil {
		return err
	}
	if res.DeletedCount != 1 {
		return errors.New("unable to delete volume")
	}
	log.WithField("volumeId", hexId).Infoln("Volume removed from database")

	return nil
}

// IsMediaInDB checks if a given media is already present in DB
// TODO: case TVSeries
func IsMediaInDB(mediaFile *media.Media) bool {
	switch (*mediaFile).(type) {
	case *media.Movie:
		movie := (*mediaFile).(*media.Movie)
		res := mongoMovies.FindOne(MongoCtx, bson.M{"tmdbid": movie.TMDBID})
		return res.Err() != mongo.ErrNoDocuments
	}

	return false
}

// AddMediaToDB adds a given media to the DB
// TODO: case TVSeries
func AddMediaToDB(mediaFile *media.Media) {
	switch (*mediaFile).(type) {
	case *media.Movie:
		movie := (*mediaFile).(*media.Movie)
		_, err := mongoMovies.InsertOne(MongoCtx, movie)
		if err != nil {
			log.WithField("path", movie.Paths[0].Path).Errorln("Unable to add movie to database")
		}
	}
}

// AddVolumeSourceToMedia adds the volume as a source to the given media
// TODO: case TVSeries
func AddVolumeSourceToMedia(mediaFile *media.Media, volume *media.Volume) {
	switch (*mediaFile).(type) {
	case *media.Movie:
		movie := (*mediaFile).(*media.Movie)
		res, err := mongoMovies.UpdateOne(MongoCtx, bson.M{"tmdbid": movie.TMDBID}, bson.M{"$addToSet": bson.M{"paths": movie.Paths[0]}})
		if err != nil || res.ModifiedCount == 0 {
			log.WithField("path", movie.Paths[0].Path).Warningln("Unable to volume as source of movie to database")
		} else {
			log.WithField("path", movie.Paths[0].Path).Debugln("Added volume as source of movie to database")
		}
	}
}

func AddSubtitleToMoviePath(movieFilePath string, sub media.Subtitle) error {
	var movie media.Movie
	err := mongoMovies.FindOne(MongoCtx, bson.M{"paths": bson.D{{Key: "$elemMatch", Value: bson.M{"path": movieFilePath}}}}).Decode(&movie)
	if err != nil {
		return err
	}
	for i, volumeFile := range movie.Paths {
		if volumeFile.Path == movieFilePath {
			movie.Paths[i].ExtSubtitles = append(movie.Paths[i].ExtSubtitles, sub)
		}
	}
	updateRes, err := mongoMovies.UpdateOne(MongoCtx, bson.M{"paths": bson.D{{Key: "$elemMatch", Value: bson.M{"path": movieFilePath}}}}, bson.M{"$set": bson.D{{Key: "paths", Value: movie.Paths}}})
	if err != nil {
		return err
	}
	if updateRes.ModifiedCount == 0 {
		return errors.New("cannot add subtitle to media")
	}
	return nil
}

// IsPersonInDB checks if a person is already registered in the DB
func IsPersonInDB(personID int64) bool {
	res := mongoPersons.FindOne(MongoCtx, bson.M{"tmdbid": personID})
	return res.Err() != mongo.ErrNoDocuments
}

// AddPersonToDB adds a person to the DB
func AddPersonToDB(person media.Person) {
	_, err := mongoPersons.InsertOne(MongoCtx, person)
	if err != nil {
		log.WithField("personID", person.TMDBID).Errorln(err)
	}
}

// AddActorsToDB upserts the actors of a media to the DB
func AddActorsToDB(actors []media.Person) {
	for _, actor := range actors {
		res, err := mongoPersons.UpdateOne(MongoCtx, bson.M{"tmdbid": actor.TMDBID}, bson.M{"$set": actor}, options.Update().SetUpsert(true))
		if err != nil {
			log.WithField("actorName", actor.Name).Warningln("Unable to add actor to database:", err)
		}
		if res.MatchedCount > 0 {
			if res.ModifiedCount > 0 {
				log.WithField("actorName", actor.Name).Debugln("Actor updated in database")
			} else if res.UpsertedCount > 0 {
				log.WithField("actorName", actor.Name).Debugln("Actor added to database")
			} else {
				log.WithField("actorName", actor.Name).Debugln("Actor already in database")
			}
		} else {
			log.WithField("actorName", actor.Name).Debugln("Actor added to database")
		}
	}
}

// GetPersonFromID returns the Person struct
func GetPersonFromID(TMDBID int64) (person media.Person, err error) {
	err = mongoPersons.FindOne(MongoCtx, bson.M{"tmdbid": TMDBID}).Decode(&person)
	return
}

// GetMovies returns a slice of Movie
func GetMovies() (movies []media.Movie) {
	moviesCur, err := mongoMovies.Find(MongoCtx, bson.M{})
	if err != nil {
		log.WithField("error", err).Errorln("Unable to retrieve movies from database")
	}
	for moviesCur.Next(MongoCtx) {
		var movie media.Movie
		err := moviesCur.Decode(&movie)
		if err != nil {
			log.WithField("error", err).Errorln("Unable to fetch movie from database")
		}
		movies = append(movies, movie)
	}
	return
}

// GetMovieFromID returns a movie from its TMDB ID
func GetMovieFromID(TMDBID int) (movie media.Movie, err error) {
	err = mongoMovies.FindOne(MongoCtx, bson.M{"tmdbid": TMDBID}).Decode(&movie)
	return movie, err
}

// GetMoviesWithActor returns a list of movies starring desired actor ID
func GetMoviesWithActor(actorID int64) (movies []media.Movie) {
	moviesCur, err := mongoMovies.Find(MongoCtx, bson.M{"cast": bson.D{{Key: "$elemMatch", Value: bson.M{"actorid": actorID}}}})
	if err != nil {
		log.WithFields(log.Fields{"error": err, "actorID": actorID}).Errorln("Unable to retrieve movies with actor from database")
		return
	}
	for moviesCur.Next(MongoCtx) {
		var movie media.Movie
		err := moviesCur.Decode(&movie)
		if err != nil {
			log.WithField("error", err).Errorln("Unable to fetch movie from database")
		}
		movies = append(movies, movie)
	}
	return
}

// GetMoviesWithDirector returns a list of movies directed by desired director ID
func GetMoviesWithDirector(directorID int64) (movies []media.Movie) {
	moviesCur, err := mongoMovies.Find(MongoCtx, bson.M{"directors": bson.D{{Key: "$elemMatch", Value: bson.M{"$eq": directorID}}}})
	if err != nil {
		log.WithFields(log.Fields{"error": err, "actorID": directorID}).Errorln("Unable to retrieve movies with actor from database")
		return
	}
	for moviesCur.Next(MongoCtx) {
		var movie media.Movie
		err := moviesCur.Decode(&movie)
		if err != nil {
			log.WithField("error", err).Errorln("Unable to fetch movie from database")
		}
		movies = append(movies, movie)
	}
	return
}

// GetMoviesWithWriter returns a list of movies written by desired writer ID
func GetMoviesWithWriter(writerID int64) (movies []media.Movie) {
	moviesCur, err := mongoMovies.Find(MongoCtx, bson.M{"writers": bson.D{{Key: "$elemMatch", Value: bson.M{"$eq": writerID}}}})
	if err != nil {
		log.WithFields(log.Fields{"error": err, "actorID": writerID}).Errorln("Unable to retrieve movies with actor from database")
		return
	}
	for moviesCur.Next(MongoCtx) {
		var movie media.Movie
		err := moviesCur.Decode(&movie)
		if err != nil {
			log.WithField("error", err).Errorln("Unable to fetch movie from database")
		}
		movies = append(movies, movie)
	}
	return
}
