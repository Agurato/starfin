package database

import (
	"context"
	"errors"
	"fmt"
	"os"

	ctx "github.com/Agurato/starfin/internal/context"
	"github.com/Agurato/starfin/internal/media"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/exp/slices"
)

type MongoDB struct {
	ctx context.Context

	client *mongo.Client

	usersColl   *mongo.Collection
	volumesColl *mongo.Collection
	moviesColl  *mongo.Collection
	personsColl *mongo.Collection
}

// InitMongoDB init mongo db
func InitMongoDB() *MongoDB {
	mongoCtx := context.Background()
	// defer cancel()
	mongoClient, err := mongo.Connect(mongoCtx,
		options.Client().ApplyURI(fmt.Sprintf("mongodb://%s:%s@%s:%s",
			os.Getenv(ctx.EnvDBUser),
			os.Getenv(ctx.EnvDBPassword),
			os.Getenv(ctx.EnvDBURL),
			os.Getenv(ctx.EnvDBPort))))
	if err != nil {
		log.Fatal(err)
	}

	mongoDb := mongoClient.Database(os.Getenv(ctx.EnvDBName))
	return &MongoDB{
		ctx:         mongoCtx,
		client:      mongoClient,
		usersColl:   mongoDb.Collection("users"),
		volumesColl: mongoDb.Collection("volumes"),
		moviesColl:  mongoDb.Collection("movies"),
		personsColl: mongoDb.Collection("persons"),
	}
}

// Close closes the MongoDB connection
func (m MongoDB) Close() {
	m.client.Disconnect(m.ctx)
}

// IsOwnerPresent checks if theres is an owner in the server
func (m MongoDB) IsOwnerPresent() bool {
	countOwners, err := m.usersColl.CountDocuments(m.ctx, bson.M{"is_owner": true})
	if err != nil {
		log.Errorln("Could not retrieve if owner is present in the database")
	}
	return countOwners > 0
}

// AddUser adds a user to the database after checking parameter
func (m MongoDB) AddUser(user *User) error {
	_, err := m.usersColl.InsertOne(m.ctx, user)
	return err
}

// DeleteUser deletes the user from the DB
func (m MongoDB) DeleteUser(hexId string) error {
	userId, err := primitive.ObjectIDFromHex(hexId)
	if err != nil {
		return errors.New("invalid volume id")
	}
	res, err := m.usersColl.DeleteOne(m.ctx, bson.M{"_id": userId})
	if err != nil {
		return err
	}
	if res.DeletedCount != 1 {
		return errors.New("unable to delete user")
	}

	return nil
}

// IsUsernameAvailable returns true if the username (case insensitive) is not in use yet
func (m MongoDB) IsUsernameAvailable(username string) (bool, error) {
	count, err := m.usersColl.CountDocuments(m.ctx, bson.M{"name": primitive.Regex{Pattern: fmt.Sprintf("^%s$", username), Options: "i"}})
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// GetUserFromID gets user from its ID
func (m MongoDB) GetUserFromID(id primitive.ObjectID, user *User) error {
	return m.usersColl.FindOne(m.ctx, bson.M{"_id": id}).Decode(user)
}

// GetUserFromName gets user from it name
func (m MongoDB) GetUserFromName(username string, user *User) error {
	return m.usersColl.FindOne(m.ctx, bson.M{"name": username}).Decode(user)
}

// GetUserNb returns the number of users from the DB
func (m MongoDB) GetUserNb() (int64, error) {
	return m.usersColl.CountDocuments(m.ctx, bson.M{})
}

// GetUsers returns the list of users in the DB
func (m MongoDB) GetUsers() (users []User, err error) {
	usersCur, err := m.usersColl.Find(m.ctx, bson.M{})
	if err != nil {
		return
	}
	for usersCur.Next(m.ctx) {
		var user User
		err = usersCur.Decode(&user)
		if err != nil {
			return
		}
		users = append(users, user)
	}
	return users, nil
}

// SetUserPassword set a new password for a specific user
func (m MongoDB) SetUserPassword(userID primitive.ObjectID, newPassword string) error {
	_, err := m.usersColl.UpdateOne(m.ctx, bson.M{"_id": userID}, bson.M{"$set": bson.M{"password": newPassword}})
	return err
}

// Fetches volume from DB using specified ID and returns it via pointer
func (m MongoDB) GetVolumeFromID(id primitive.ObjectID, volume *media.Volume) error {
	return m.volumesColl.FindOne(m.ctx, bson.M{"_id": id}).Decode(&volume)
}

// GetVolumes returns the list of volumes in the DB
func (m MongoDB) GetVolumes() (volumes []media.Volume, err error) {
	volumeCur, err := m.volumesColl.Find(m.ctx, bson.M{})
	if err != nil {
		return
	}
	for volumeCur.Next(m.ctx) {
		var vol media.Volume
		err = volumeCur.Decode(&vol)
		if err != nil {
			return
		}
		volumes = append(volumes, vol)
	}
	return volumes, nil
}

// AddVolume adds a volume to the DB and start scanning the volume
func (m MongoDB) AddVolume(volume *media.Volume) error {
	_, err := m.volumesColl.InsertOne(m.ctx, *volume)
	return err
}

// DeleteVolume deletes the volume from the DB and all the movie which originated only from this volume
func (m MongoDB) DeleteVolume(hexId string) error {
	volumeId, err := primitive.ObjectIDFromHex(hexId)
	if err != nil {
		return errors.New("invalid volume id")
	}

	// Remove specified volume from all movie source
	update, err := m.moviesColl.UpdateMany(m.ctx,
		bson.M{},
		bson.D{
			{Key: "$pull", Value: bson.D{{Key: "volumefiles", Value: bson.D{{Key: "fromvolume", Value: volumeId}}}}},
		})
	if err != nil {
		return err
	}
	log.WithField("volumeId", hexId).Infof("%d movies are concerned with this volume deletion\n", update.ModifiedCount)
	del, err := m.moviesColl.DeleteMany(m.ctx, bson.M{"volumefiles": bson.D{{Key: "$size", Value: 0}}})
	if err != nil {
		return err
	}
	log.WithField("volumeId", hexId).Infof("%d movies were removed from database\n", del.DeletedCount)

	// Remove specified volume from "volumes" collection
	res, err := m.volumesColl.DeleteOne(m.ctx, bson.M{"_id": volumeId})
	if err != nil {
		return err
	}
	if res.DeletedCount != 1 {
		return errors.New("unable to delete volume")
	}
	log.WithField("volumeId", hexId).Infoln("Volume removed from database")

	return nil
}

// IsMovieInDB checks if a given movie is already present in DB
func (m MongoDB) IsMoviePresent(movie *media.Movie) bool {
	res := m.moviesColl.FindOne(m.ctx, bson.M{"tmdbid": movie.TMDBID})
	return res.Err() != mongo.ErrNoDocuments
}

// AddMovieToDB adds a given movie to the DB
func (m MongoDB) AddMovie(movie *media.Movie) error {
	_, err := m.moviesColl.InsertOne(m.ctx, movie)
	return err
}

// AddVolumeSourceToMovie adds the volume as a source to the given media
func (m MongoDB) AddVolumeSourceToMovie(movie *media.Movie) {
	res, err := m.moviesColl.UpdateOne(m.ctx, bson.M{"tmdbid": movie.TMDBID}, bson.M{"$addToSet": bson.M{"volumefiles": movie.VolumeFiles[0]}})
	if err != nil || res.ModifiedCount == 0 {
		log.WithField("path", movie.VolumeFiles[0].Path).Warningln("Unable to add volume as source of movie to database")
	} else {
		log.WithField("path", movie.VolumeFiles[0].Path).Debugln("Added volume as source of movie to database")
	}
}

// GetMovieFromPath retrieves a movie from a path
func (m MongoDB) GetMovieFromPath(mediaPath string) (movie *media.Movie, err error) {
	err = m.moviesColl.FindOne(m.ctx, bson.M{"volumefiles": bson.D{{Key: "$elemMatch", Value: bson.M{"path": mediaPath}}}}).Decode(movie)
	if err != nil {
		return nil, errors.New("could not get movie from path")
	}
	return movie, nil
}

// ReplaceMoviePath replaces a movie path if needed
func (m MongoDB) ReplaceMoviePath(oldMoviePath, newMoviePath string, newMovie *media.Movie) error {
	var (
		oldMovie media.Movie
	)
	// Get the current movie struct from mongo
	err := m.moviesColl.FindOne(m.ctx, bson.M{"volumefiles": bson.D{{Key: "$elemMatch", Value: bson.M{"path": oldMoviePath}}}}).Decode(&oldMovie)
	if err != nil {
		return errors.New("could not get movie from path")
	}
	oldPathIndex := slices.IndexFunc(oldMovie.VolumeFiles, func(vf media.VolumeFile) bool {
		return vf.Path == oldMoviePath
	})

	if oldMovie.TMDBID == newMovie.TMDBID { // If they have the same TMDB ID, replace the correct volumeFile
		m.moviesColl.UpdateOne(m.ctx, bson.M{"volumefiles": bson.D{{Key: "$elemMatch", Value: bson.M{"path": oldMoviePath}}}}, bson.M{"$set": bson.M{fmt.Sprintf("volumefiles.%d", oldPathIndex): newMovie.VolumeFiles[0]}})
	} else { // If they don't have the same TMDB ID, remove the path from the previous movie
		// If it only had 1 volumeFile, remove the movie entirely
		if len(oldMovie.VolumeFiles) == 1 {
			delete, err := m.moviesColl.DeleteOne(m.ctx, bson.M{"volumefiles": bson.D{{Key: "$elemMatch", Value: bson.M{"path": oldMoviePath}}}})
			if err != nil {
				return err
			}
			if delete.DeletedCount == 0 {
				return errors.New("could not delete movie when replacing with a new one")
			}
		} else {
			update, err := m.moviesColl.UpdateOne(m.ctx,
				bson.M{"volumefiles": bson.D{{Key: "$elemMatch", Value: bson.M{"path": oldMoviePath}}}},
				bson.D{{Key: "$pull", Value: bson.D{{Key: "volumefiles", Value: bson.D{{Key: "path", Value: oldMoviePath}}}}}})
			if err != nil {
				return err
			}
			if update.ModifiedCount == 0 {
				return errors.New("could not update movie when replacing with a new one")
			}
		}

		// Fetch movie details
		newMovie.FetchDetails()
		m.AddMovie(newMovie)
	}

	return nil
}

// RemoveMovieFile removes a movie from the database
func (m MongoDB) RemoveMovieFile(path string) error {
	deleteRes, err := m.moviesColl.DeleteOne(m.ctx, bson.M{"volumefiles": bson.D{{Key: "$elemMatch", Value: bson.M{"path": path}}}})
	if err != nil {
		return err
	}
	if deleteRes.DeletedCount == 0 {
		log.WithFields(log.Fields{"mediaPath": path}).Warningln("No movie was deleted")
	}
	return nil
}

// RemoveSubtitleFile removes a movie subtitle from the database
func (m MongoDB) RemoveSubtitleFile(mediaPath, subtitlePath string) error {
	var movie media.Movie
	err := m.moviesColl.FindOne(m.ctx, bson.M{"volumefiles": bson.D{{Key: "$elemMatch", Value: bson.M{"path": mediaPath}}}}).Decode(&movie)
	if err != nil {
		return err
	}
	volumeIndex := slices.IndexFunc(movie.VolumeFiles, func(vFile media.VolumeFile) bool {
		return vFile.Path == mediaPath
	})
	if volumeIndex == -1 {
		return errors.New("cannot remove subtitle from movie (no matching volume file")
	}

	subtitleIndex := slices.IndexFunc(movie.VolumeFiles[volumeIndex].ExtSubtitles, func(sub media.Subtitle) bool {
		return sub.Path == subtitlePath
	})
	if subtitleIndex == -1 {
		return errors.New("cannot remove subtitle from movie (no matching subtitle file")
	}
	movie.VolumeFiles[volumeIndex].ExtSubtitles = slices.Delete(movie.VolumeFiles[volumeIndex].ExtSubtitles, subtitleIndex, subtitleIndex+1)

	updateRes, err := m.moviesColl.UpdateOne(m.ctx, bson.M{"volumefiles": bson.D{{Key: "$elemMatch", Value: bson.M{"path": mediaPath}}}}, bson.M{"$set": bson.D{{Key: "volumefiles", Value: movie.VolumeFiles}}})
	if err != nil {
		return err
	}
	if updateRes.ModifiedCount == 0 {
		return errors.New("cannot remove subtitle from media")
	}
	return nil
}

// IsPersonPresent checks if a person is already registered in the DB
func (m MongoDB) IsPersonPresent(personID int64) bool {
	res := m.personsColl.FindOne(m.ctx, bson.M{"tmdbid": personID})
	return res.Err() != mongo.ErrNoDocuments
}

// AddPerson adds a person to the DB
// TODO: upsert
func (m MongoDB) AddPerson(person media.Person) {
	_, err := m.personsColl.InsertOne(m.ctx, person)
	if err != nil {
		log.WithField("personID", person.TMDBID).Errorln(err)
	}
}

// AddActors upserts the actors of a movie to the DB
func (m MongoDB) AddActors(actors []media.Person) {
	for _, actor := range actors {
		res, err := m.personsColl.UpdateOne(m.ctx, bson.M{"tmdbid": actor.TMDBID}, bson.M{"$set": actor}, options.Update().SetUpsert(true))
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
func (m MongoDB) GetPersonFromID(TMDBID int64) (person media.Person, err error) {
	err = m.personsColl.FindOne(m.ctx, bson.M{"tmdbid": TMDBID}).Decode(&person)
	return
}

// GetMovies returns a slice of Movie
func (m MongoDB) GetMovies() (movies []media.Movie) {
	moviesCur, err := m.moviesColl.Find(m.ctx, bson.M{})
	if err != nil {
		log.WithField("error", err).Errorln("Unable to retrieve movies from database")
	}
	for moviesCur.Next(m.ctx) {
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
func (m MongoDB) GetMovieFromID(id primitive.ObjectID) (movie media.Movie, err error) {
	err = m.moviesColl.FindOne(m.ctx, bson.M{"_id": id}).Decode(&movie)
	return movie, err
}

// GetMoviesWithActor returns a list of movies starring desired actor ID
func (m MongoDB) GetMoviesWithActor(actorID int64) (movies []media.Movie) {
	moviesCur, err := m.moviesColl.Find(m.ctx, bson.M{"cast": bson.D{{Key: "$elemMatch", Value: bson.M{"actorid": actorID}}}})
	if err != nil {
		log.WithFields(log.Fields{"error": err, "actorID": actorID}).Errorln("Unable to retrieve movies with actor from database")
		return
	}
	for moviesCur.Next(m.ctx) {
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
func (m MongoDB) GetMoviesWithDirector(directorID int64) (movies []media.Movie) {
	moviesCur, err := m.moviesColl.Find(m.ctx, bson.M{"directors": bson.D{{Key: "$elemMatch", Value: bson.M{"$eq": directorID}}}})
	if err != nil {
		log.WithFields(log.Fields{"error": err, "actorID": directorID}).Errorln("Unable to retrieve movies with actor from database")
		return
	}
	for moviesCur.Next(m.ctx) {
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
func (m MongoDB) GetMoviesWithWriter(writerID int64) (movies []media.Movie) {
	moviesCur, err := m.moviesColl.Find(m.ctx, bson.M{"writers": bson.D{{Key: "$elemMatch", Value: bson.M{"$eq": writerID}}}})
	if err != nil {
		log.WithFields(log.Fields{"error": err, "actorID": writerID}).Errorln("Unable to retrieve movies with actor from database")
		return
	}
	for moviesCur.Next(m.ctx) {
		var movie media.Movie
		err := moviesCur.Decode(&movie)
		if err != nil {
			log.WithField("error", err).Errorln("Unable to fetch movie from database")
		}
		movies = append(movies, movie)
	}
	return
}

// AddSubtitleToMoviePath adds the subtitle to a movie given the movie path
func (m MongoDB) AddSubtitleToMoviePath(movieFilePath string, sub media.Subtitle) error {
	var movie media.Movie
	err := m.moviesColl.FindOne(m.ctx, bson.M{"volumefiles": bson.D{{Key: "$elemMatch", Value: bson.M{"path": movieFilePath}}}}).Decode(&movie)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(movie.VolumeFiles, func(vFile media.VolumeFile) bool {
		return vFile.Path == movieFilePath
	})
	if i == -1 {
		return errors.New("cannot add subtitle to movie (no matching volume file")
	}
	if slices.Contains(movie.VolumeFiles[i].ExtSubtitles, sub) {
		return errors.New("subtitle is already added to media")
	}
	movie.VolumeFiles[i].ExtSubtitles = append(movie.VolumeFiles[i].ExtSubtitles, sub)
	updateRes, err := m.moviesColl.UpdateOne(m.ctx, bson.M{"volumefiles": bson.D{{Key: "$elemMatch", Value: bson.M{"path": movieFilePath}}}}, bson.M{"$set": bson.D{{Key: "volumefiles", Value: movie.VolumeFiles}}})
	if err != nil {
		return err
	}
	if updateRes.ModifiedCount == 0 {
		return errors.New("cannot add subtitle to media")
	}
	return nil
}

// GetMovieFromExternalSubtitle returns a movie from its external subtitle path
func (m MongoDB) GetMovieFromExternalSubtitle(subtitlePath string) (media.Movie, error) {
	var movie media.Movie
	err := m.moviesColl.FindOne(
		m.ctx,
		bson.M{
			"volumefiles": bson.D{{
				Key: "$elemMatch",
				Value: bson.M{"extsubtitles": bson.D{{
					Key:   "$elemMatch",
					Value: bson.M{"path": subtitlePath},
				}}},
			}},
		}).Decode(&movie)
	return movie, err
}
