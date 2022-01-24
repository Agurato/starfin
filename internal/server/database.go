package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

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
	mongoDb *mongo.Database
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
	return
}

// AddUser adds a user to the database after checking parameter
func AddUser(username, password1, password2 string, isAdmin bool) error {
	argon := argon2.DefaultConfig()
	userColl := mongoDb.Collection("users")

	// Check username length
	if len(username) < 3 || len(username) > 25 {
		return errors.New("username must be between 3 and 25 characters")
	}

	// Check if username is not already taken
	count, _ := userColl.CountDocuments(MongoCtx, bson.M{"name": primitive.Regex{Pattern: fmt.Sprintf("^%s$", username), Options: "i"}})
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
	_, err = userColl.InsertOne(MongoCtx, user)
	return err
}
