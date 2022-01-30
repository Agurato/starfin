package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Agurato/down-low-d/internal/media"
	"github.com/matthewhartstonge/argon2"
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
	IsAdmin  bool               `bson:"is_admin"`
}

var (
	MongoCtx     context.Context
	mongoDb      *mongo.Database
	mongoUsers   *mongo.Collection
	mongoVolumes *mongo.Collection
	mongoMovies  *mongo.Collection
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
	return
}

// AddUser adds a user to the database after checking parameter
func AddUser(username, password1, password2 string, isAdmin bool) error {
	argon := argon2.DefaultConfig()

	// Check username length
	if len(username) < 3 || len(username) > 25 {
		return errors.New("username must be between 3 and 25 characters")
	}

	// Check if username is not already taken
	count, err := mongoUsers.CountDocuments(MongoCtx, bson.M{"name": primitive.Regex{Pattern: fmt.Sprintf("^%s$", username), Options: "i"}})
	if err != nil {
		// TODO: log
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
		IsAdmin:  isAdmin,
	}
	_, err = mongoUsers.InsertOne(MongoCtx, user)
	return err
}

// GetUserFromID
func GetUserFromID(id primitive.ObjectID, user *User) error {
	return mongoUsers.FindOne(MongoCtx, bson.M{"_id": id}).Decode(&user)
}

func GetUserNb() int64 {
	count, err := mongoUsers.CountDocuments(MongoCtx, bson.M{})
	if err != nil {
		// TODO: log
	}
	return count
}

func GetVolumeFromID(id primitive.ObjectID, volume *media.Volume) error {
	return mongoVolumes.FindOne(MongoCtx, bson.M{"_id": id}).Decode(&volume)
}

func GetVolumes() (volumes []media.Volume) {
	volumeCur, err := mongoVolumes.Find(MongoCtx, bson.M{})
	if err != nil {
		// TODO: log
	}
	for volumeCur.Next(MongoCtx) {
		var vol media.Volume
		err := volumeCur.Decode(&vol)
		if err != nil {
			// TODO: log
		}
		volumes = append(volumes, vol)
	}
	return
}

func AddVolume(volume media.Volume) error {
	_, err := mongoVolumes.InsertOne(MongoCtx, volume)
	if err == nil {
		go func() {
			for _, mediaFile := range volume.Scan() {
				// If media is already in DB, add the current Volume to the media's origin
				if IsMediaInDB(&mediaFile) {
					fmt.Printf("%s is in DB\n", mediaFile.(*media.Movie).Path)
				} else {
					AddMediaToDB(&mediaFile)
				}
			}
		}()
	}
	return err
}

// IsMediaInDB checks if a given media is already present in DB
// TODO: case TVSeries
func IsMediaInDB(mediaFile *media.Media) bool {

	switch (*mediaFile).(type) {
	case *media.Movie:
		movie := (*mediaFile).(*media.Movie)
		res := mongoMovies.FindOne(MongoCtx, bson.M{"path": movie.Path})
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
			fmt.Println(err)
		}
	}
}
