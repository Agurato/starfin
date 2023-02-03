package model

import "go.mongodb.org/mongo-driver/bson/primitive"

// User is a user
type User struct {
	ID       primitive.ObjectID `bson:"_id"`
	Name     string             `bson:"name"`
	Password string             `bson:"password"`
	IsOwner  bool               `bson:"is_owner"`
	IsAdmin  bool               `bson:"is_admin"`
}
