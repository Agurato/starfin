package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/Agurato/starfin/internal/model"
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
	filmsColl   *mongo.Collection
	peopleColl  *mongo.Collection
}

// InitMongoDB init mongo db
func NewMongoDB(dbUser, dbPassword, dbURL, dbPort, dbName string) *MongoDB {
	mongoCtx := context.Background()
	// defer cancel()
	mongoClient, err := mongo.Connect(mongoCtx, options.Client().ApplyURI(fmt.Sprintf("mongodb://%s:%s@%s:%s", dbUser, dbPassword, dbURL, dbPort)))
	if err != nil {
		log.Fatal(err)
	}

	mongoDb := mongoClient.Database(dbName)
	return &MongoDB{
		ctx:         mongoCtx,
		client:      mongoClient,
		usersColl:   mongoDb.Collection("users"),
		volumesColl: mongoDb.Collection("volumes"),
		filmsColl:   mongoDb.Collection("films"),
		peopleColl:  mongoDb.Collection("people"),
	}
}

func getFilmPathFilter(path string) primitive.M {
	return bson.M{"volume_files": bson.D{{Key: "$elemMatch", Value: bson.M{"path": path}}}}
}

// Close closes the MongoDB connection
func (m MongoDB) Close() {
	m.client.Disconnect(m.ctx)
}

// IsOwnerPresent checks if theres is an owner in the server
func (m MongoDB) IsOwnerPresent() (bool, error) {
	countOwners, err := m.usersColl.CountDocuments(m.ctx, bson.M{"is_owner": true})
	if err != nil {
		return false, err
	}
	return countOwners > 0, nil
}

// CreateUser adds a user to the database after checking parameter
func (m MongoDB) CreateUser(user *model.User) error {
	_, err := m.usersColl.InsertOne(m.ctx, user)
	return err
}

// DeleteUser deletes the user from the DB
func (m MongoDB) DeleteUser(userId primitive.ObjectID) error {
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
func (m MongoDB) GetUserFromID(id primitive.ObjectID) (*model.User, error) {
	var user model.User
	err := m.usersColl.FindOne(m.ctx, bson.M{"_id": id}).Decode(&user)
	return &user, err
}

// GetUserFromName gets user from it name
func (m MongoDB) GetUserFromName(username string, user *model.User) error {
	return m.usersColl.FindOne(m.ctx, bson.M{"name": username}).Decode(user)
}

// GetUserNb returns the number of users from the DB
func (m MongoDB) GetUserNb() (int64, error) {
	return m.usersColl.CountDocuments(m.ctx, bson.M{})
}

// GetUsers returns the list of users in the DB
func (m MongoDB) GetUsers() (users []model.User, err error) {
	usersCur, err := m.usersColl.Find(m.ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("Error while retrieving users from DB: %w", err)
	}
	for usersCur.Next(m.ctx) {
		var user model.User
		err = usersCur.Decode(&user)
		if err != nil {
			return nil, fmt.Errorf("Error while decoding user from DB: %w", err)
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
func (m MongoDB) GetVolumeFromID(id primitive.ObjectID) (*model.Volume, error) {
	var volume model.Volume
	err := m.volumesColl.FindOne(m.ctx, bson.M{"_id": id}).Decode(&volume)
	return &volume, err
}

// GetVolumes returns the list of volumes in the DB
func (m MongoDB) GetVolumes() (volumes []model.Volume, err error) {
	volumeCur, err := m.volumesColl.Find(m.ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("Error while retrieving volumes from DB: %w", err)
	}
	for volumeCur.Next(m.ctx) {
		var vol model.Volume
		err = volumeCur.Decode(&vol)
		if err != nil {
			return nil, fmt.Errorf("Error while decoding volume from DB: %w", err)
		}
		volumes = append(volumes, vol)
	}
	return volumes, nil
}

// AddVolume adds a volume to the DB and start scanning the volume
func (m MongoDB) AddVolume(volume *model.Volume) error {
	_, err := m.volumesColl.InsertOne(m.ctx, *volume)
	return err
}

// DeleteVolume deletes the volume from the DB and all the film which originated only from this volume
func (m MongoDB) DeleteVolume(volumeId primitive.ObjectID) error {
	// Remove specified volume from all film source
	update, err := m.filmsColl.UpdateMany(m.ctx,
		bson.M{},
		bson.D{
			{Key: "$pull", Value: bson.D{{Key: "volume_files", Value: bson.D{{Key: "fromvolume", Value: volumeId}}}}},
		})
	if err != nil {
		return err
	}
	log.WithField("volumeId", volumeId).Infof("%d films are concerned with this volume deletion\n", update.ModifiedCount)
	del, err := m.filmsColl.DeleteMany(m.ctx, bson.M{"volume_files": bson.D{{Key: "$size", Value: 0}}})
	if err != nil {
		return err
	}
	log.WithField("volumeId", volumeId).Infof("%d films were removed from database\n", del.DeletedCount)

	// Remove specified volume from "volumes" collection
	res, err := m.volumesColl.DeleteOne(m.ctx, bson.M{"_id": volumeId})
	if err != nil {
		return err
	}
	if res.DeletedCount != 1 {
		return errors.New("unable to delete volume")
	}
	log.WithField("volumeId", volumeId).Infoln("Volume removed from database")

	return nil
}

// IsFilmPathPresent checks if a film path is present in the database
func (m MongoDB) IsFilmPathPresent(filmPath string) bool {
	film := model.Film{}
	return m.filmsColl.FindOne(m.ctx, getFilmPathFilter(filmPath)).Decode(&film) == nil
}

// IsSubtitlePathPresent checks if a subtitle path is present in the database
func (m MongoDB) IsSubtitlePathPresent(subPath string) bool {
	_, err := m.GetFilmFromExternalSubtitle(subPath)
	return err == nil
}

// IsFilmInDB checks if a given film is already present in DB
func (m MongoDB) IsFilmPresent(film *model.Film) bool {
	res := m.filmsColl.FindOne(m.ctx, bson.M{"tmdb_id": film.TMDBID})
	return res.Err() != mongo.ErrNoDocuments
}

// AddFilmToDB adds a given film to the DB
// If the film is already in the database, updates it
func (m MongoDB) AddFilm(film *model.Film) error {
	_, err := m.filmsColl.UpdateOne(m.ctx, bson.M{"_id": film.ID}, bson.M{"$set": film}, options.Update().SetUpsert(true))
	return err
}

// AddVolumeSourceToFilm adds the volume as a source to the given media
func (m MongoDB) AddVolumeSourceToFilm(film *model.Film) error {
	res, err := m.filmsColl.UpdateOne(m.ctx, bson.M{"tmdb_id": film.TMDBID}, bson.M{"$addToSet": bson.M{"volume_files": film.VolumeFiles[0]}})
	if err != nil {
		return err
	} else if res.ModifiedCount == 0 {
		return errors.New("unable to add volume as source of film to database")
	}
	log.WithField("path", film.VolumeFiles[0].Path).Debugln("Added volume as source of film to database")
	return nil
}

// GetFilmFromPath retrieves a film from a path
func (m MongoDB) GetFilmFromPath(filmPath string) (film *model.Film, err error) {
	film = &model.Film{}
	err = m.filmsColl.FindOne(m.ctx, getFilmPathFilter(filmPath)).Decode(film)
	if err != nil {
		return nil, errors.New("could not get film from path")
	}
	return film, nil
}

// UpdateFilmVolumeFile updates the path to a film
// film: Film struct that has its path changed
// oldPath: file path of the volumefile that will be changed
// newVolumeFile: VolumeFile struct that replaces the previous one
func (m MongoDB) UpdateFilmVolumeFile(film *model.Film, oldPath string, newVolumeFile model.VolumeFile) error {
	oldPathIndex := slices.IndexFunc(film.VolumeFiles, func(vf model.VolumeFile) bool {
		return vf.Path == oldPath
	})
	update, err := m.filmsColl.UpdateOne(m.ctx, getFilmPathFilter(oldPath), bson.M{"$set": bson.M{fmt.Sprintf("volume_files.%d", oldPathIndex): newVolumeFile}})
	if err != nil {
		return err
	}
	if update.ModifiedCount == 0 {
		return errors.New("could not update the volume file")
	}
	return nil
}

// DeleteFilm deletes a film
func (m MongoDB) DeleteFilm(ID primitive.ObjectID) error {
	del, err := m.filmsColl.DeleteOne(m.ctx, bson.M{"_id": ID})
	if err != nil {
		return err
	}
	if del.DeletedCount == 0 {
		return errors.New("could not delete film")
	}
	return nil
}

// DeleteFilmFromPath removes a film from the database
// If the film has only 1 volume file, then the film is entirely deleted
func (m MongoDB) DeleteFilmVolumeFile(path string) error {
	film, err := m.GetFilmFromPath(path)
	if err != nil {
		return err
	}
	// If it only had 1 volumeFile, remove the film entirely
	if len(film.VolumeFiles) == 1 {
		m.DeleteFilm(film.ID)
	} else {
		update, err := m.filmsColl.UpdateOne(m.ctx,
			getFilmPathFilter(path),
			bson.D{{Key: "$pull", Value: bson.D{{Key: "volume_files", Value: bson.D{{Key: "path", Value: path}}}}}})
		if err != nil {
			return err
		}
		if update.ModifiedCount == 0 {
			return errors.New("could not update film when replacing with a new one")
		}
	}
	return nil
}

// RemoveSubtitleFile removes a film subtitle from the database
func (m MongoDB) RemoveSubtitleFile(mediaPath, subtitlePath string) error {
	var film model.Film
	err := m.filmsColl.FindOne(m.ctx, getFilmPathFilter(mediaPath)).Decode(&film)
	if err != nil {
		return err
	}
	volumeIndex := slices.IndexFunc(film.VolumeFiles, func(vFile model.VolumeFile) bool {
		return vFile.Path == mediaPath
	})
	if volumeIndex == -1 {
		return errors.New("cannot remove subtitle from film (no matching volume file")
	}

	subtitleIndex := slices.IndexFunc(film.VolumeFiles[volumeIndex].ExtSubtitles, func(sub model.Subtitle) bool {
		return sub.Path == subtitlePath
	})
	if subtitleIndex == -1 {
		return errors.New("cannot remove subtitle from film (no matching subtitle file")
	}
	film.VolumeFiles[volumeIndex].ExtSubtitles = slices.Delete(film.VolumeFiles[volumeIndex].ExtSubtitles, subtitleIndex, subtitleIndex+1)

	updateRes, err := m.filmsColl.UpdateOne(m.ctx, getFilmPathFilter(mediaPath), bson.M{"$set": bson.D{{Key: "volume_files", Value: film.VolumeFiles}}})
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
	res := m.peopleColl.FindOne(m.ctx, bson.M{"tmdb_id": personID})
	return res.Err() != mongo.ErrNoDocuments
}

// AddPerson adds a person to the DB
// TODO: upsert
func (m MongoDB) AddPerson(person *model.Person) {
	_, err := m.peopleColl.InsertOne(m.ctx, *person)
	if err != nil {
		log.WithField("personID", person.TMDBID).Errorln(err)
	}
}

// AddActors upserts the actors of a film to the DB
func (m MongoDB) AddActors(actors []model.Person) {
	for _, actor := range actors {
		res, err := m.peopleColl.UpdateOne(m.ctx, bson.M{"tmdb_id": actor.TMDBID}, bson.M{"$set": actor}, options.Update().SetUpsert(true))
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
func (m MongoDB) GetPersonFromID(ID primitive.ObjectID) (*model.Person, error) {
	var person model.Person
	err := m.peopleColl.FindOne(m.ctx, bson.M{"_id": ID}).Decode(&person)
	return &person, err
}

// GetPersonFromTMDBID returns the Person struct
func (m MongoDB) GetPersonFromTMDBID(TMDBID int64) (*model.Person, error) {
	var person model.Person
	err := m.peopleColl.FindOne(m.ctx, bson.M{"tmdb_id": TMDBID}).Decode(&person)
	return &person, err
}

func (m MongoDB) GetPeople() (people []model.Person, err error) {
	options := options.Find()
	options.SetSort(bson.M{"title": 1})
	peopleCur, err := m.peopleColl.Find(m.ctx, bson.M{}, options)
	if err != nil {
		return nil, fmt.Errorf("Error while retrieving people from DB: %w", err)
	}
	for peopleCur.Next(m.ctx) {
		var person model.Person
		err := peopleCur.Decode(&person)
		if err != nil {
			return nil, fmt.Errorf("Error while decoding person from DB: %w", err)
		}
		people = append(people, person)
	}
	return
}

// GetFilmFromID returns a film from its TMDB ID
func (m MongoDB) GetFilmFromID(id primitive.ObjectID) (*model.Film, error) {
	var film model.Film
	err := m.filmsColl.FindOne(m.ctx, bson.M{"_id": id}).Decode(&film)
	return &film, err
}

func (m MongoDB) GetFilmCount() int64 {
	count, err := m.filmsColl.CountDocuments(m.ctx, bson.M{})
	if err != nil {
		return 0
	}
	return count
}

// GetFilms returns a slice of Film
func (m MongoDB) GetFilms() (films []model.Film, err error) {
	options := options.Find()
	options.SetSort(bson.M{"title": 1})
	filmsCur, err := m.filmsColl.Find(m.ctx, bson.M{}, options)
	if err != nil {
		return nil, fmt.Errorf("Error while retrieving films from DB: %w", err)
	}
	for filmsCur.Next(m.ctx) {
		var film model.Film
		err := filmsCur.Decode(&film)
		if err != nil {
			return nil, fmt.Errorf("Error while decoding film from DB: %w", err)
		}
		films = append(films, film)
	}
	return
}

func (m MongoDB) GetFilmsFiltered(years []int, genre, country string) (films []model.Film) {
	options := options.Find()
	options.SetSort(bson.M{"title": 1})
	filter := bson.M{}
	if len(years) > 0 {
		var orYears []bson.M
		for _, year := range years {
			orYears = append(orYears, bson.M{"releaseyear": year})
		}
		filter["$or"] = orYears
	}
	if genre != "" {
		filter["genres"] = primitive.Regex{Pattern: fmt.Sprintf("^%s$", genre), Options: "i"}
	}
	if country != "" {
		filter["prodcountries"] = primitive.Regex{Pattern: fmt.Sprintf("^%s$", country), Options: "i"}
	}

	filmsCur, err := m.filmsColl.Find(m.ctx, filter, options)
	if err != nil {
		log.WithField("error", err).Errorln("Unable to retrieve films from database")
		return
	}
	for filmsCur.Next(m.ctx) {
		var film model.Film
		err := filmsCur.Decode(&film)
		if err != nil {
			log.WithField("error", err).Errorln("Unable to fetch film from database")
		}
		films = append(films, film)
	}
	return
}

// GetFilmsRange returns a slice of Film from start to number
func (m MongoDB) GetFilmsRange(start, number int) (films []model.Film) {
	options := options.Find()
	options.SetSort(bson.M{"title": 1})
	options.SetSkip(int64(start))
	options.SetLimit(int64(number))
	filmsCur, err := m.filmsColl.Find(m.ctx, bson.M{}, options)
	if err != nil {
		log.WithField("error", err).Errorln("Unable to retrieve films from database")
		return
	}
	for filmsCur.Next(m.ctx) {
		var film model.Film
		err := filmsCur.Decode(&film)
		if err != nil {
			log.WithField("error", err).Errorln("Unable to fetch film from database")
		}
		films = append(films, film)
	}
	return
}

// GetFilmsFromVolume retrieves all films from a specific volume ID
func (m MongoDB) GetFilmsFromVolume(id primitive.ObjectID) (films []model.Film) {
	filmsCur, err := m.filmsColl.Find(m.ctx, bson.M{"volume_files": bson.D{{Key: "$elemMatch", Value: bson.M{"fromvolume": id}}}})
	if err != nil {
		log.WithField("error", err).Errorln("Unable to retrieve films from database")
	}
	for filmsCur.Next(m.ctx) {
		var film model.Film
		err := filmsCur.Decode(&film)
		if err != nil {
			log.WithField("error", err).Errorln("Unable to fetch film from database")
		}
		films = append(films, film)
	}
	return
}

// GetFilmsWithActor returns a list of films starring desired actor ID
func (m MongoDB) GetFilmsWithActor(actorID int64) (films []model.Film) {
	filmsCur, err := m.filmsColl.Find(m.ctx, bson.M{"characters": bson.D{{Key: "$elemMatch", Value: bson.M{"actor_id": actorID}}}})
	if err != nil {
		log.WithFields(log.Fields{"error": err, "actorID": actorID}).Errorln("Unable to retrieve films with actor from database")
		return
	}
	for filmsCur.Next(m.ctx) {
		var film model.Film
		err := filmsCur.Decode(&film)
		if err != nil {
			log.WithField("error", err).Errorln("Unable to fetch film from database")
		}
		films = append(films, film)
	}
	return
}

// GetFilmsWithDirector returns a list of films directed by desired director ID
func (m MongoDB) GetFilmsWithDirector(directorID int64) (films []model.Film) {
	filmsCur, err := m.filmsColl.Find(m.ctx, bson.M{"directors": bson.D{{Key: "$elemMatch", Value: bson.M{"$eq": directorID}}}})
	if err != nil {
		log.WithFields(log.Fields{"error": err, "actorID": directorID}).Errorln("Unable to retrieve films with actor from database")
		return
	}
	for filmsCur.Next(m.ctx) {
		var film model.Film
		err := filmsCur.Decode(&film)
		if err != nil {
			log.WithField("error", err).Errorln("Unable to fetch film from database")
		}
		films = append(films, film)
	}
	return
}

// GetFilmsWithWriter returns a list of films written by desired writer ID
func (m MongoDB) GetFilmsWithWriter(writerID int64) (films []model.Film) {
	filmsCur, err := m.filmsColl.Find(m.ctx, bson.M{"writers": bson.D{{Key: "$elemMatch", Value: bson.M{"$eq": writerID}}}})
	if err != nil {
		log.WithFields(log.Fields{"error": err, "actorID": writerID}).Errorln("Unable to retrieve films with actor from database")
		return
	}
	for filmsCur.Next(m.ctx) {
		var film model.Film
		err := filmsCur.Decode(&film)
		if err != nil {
			log.WithField("error", err).Errorln("Unable to fetch film from database")
		}
		films = append(films, film)
	}
	return
}

// AddSubtitleToFilmPath adds the subtitle to a film given the film path
func (m MongoDB) AddSubtitleToFilmPath(filmFilePath string, sub model.Subtitle) error {
	var film model.Film
	err := m.filmsColl.FindOne(m.ctx, getFilmPathFilter(filmFilePath)).Decode(&film)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(film.VolumeFiles, func(vFile model.VolumeFile) bool {
		return vFile.Path == filmFilePath
	})
	if i == -1 {
		return errors.New("cannot add subtitle to film (no matching volume file")
	}
	if slices.Contains(film.VolumeFiles[i].ExtSubtitles, sub) {
		return errors.New("subtitle is already added to media")
	}
	film.VolumeFiles[i].ExtSubtitles = append(film.VolumeFiles[i].ExtSubtitles, sub)
	updateRes, err := m.filmsColl.UpdateOne(m.ctx, getFilmPathFilter(filmFilePath), bson.M{"$set": bson.D{{Key: "volume_files", Value: film.VolumeFiles}}})
	if err != nil {
		return err
	}
	if updateRes.ModifiedCount == 0 {
		return errors.New("cannot add subtitle to media")
	}
	return nil
}

// GetFilmFromExternalSubtitle returns a film from its external subtitle path
func (m MongoDB) GetFilmFromExternalSubtitle(subtitlePath string) (model.Film, error) {
	var film model.Film
	err := m.filmsColl.FindOne(
		m.ctx,
		bson.M{
			"volume_files": bson.D{{
				Key: "$elemMatch",
				Value: bson.M{"extsubtitles": bson.D{{
					Key:   "$elemMatch",
					Value: bson.M{"path": subtitlePath},
				}}},
			}},
		}).Decode(&film)
	return film, err
}
