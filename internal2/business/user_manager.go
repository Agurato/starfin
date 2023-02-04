package business

import (
	"errors"
	"fmt"

	"github.com/Agurato/starfin/internal2/model"
	"github.com/matthewhartstonge/argon2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserStorer interface {
	IsOwnerPresent() bool
	IsUsernameAvailable(username string) (bool, error)
	GetUserFromName(username string, user *model.User) error
	GetUserNb() (int64, error)
	AddUser(user *model.User) error
	SetUserPassword(userID primitive.ObjectID, newPassword string) error
}

type UserManager struct {
	UserStorer
}

func (um UserManager) GetUserNb() (int64, error) {
	return um.UserStorer.GetUserNb()
}

// AddUser checks that the user and password follow specific rules and adds it to the database
func (um UserManager) AddUser(username, password1, password2 string, isAdmin bool) (*model.User, error) {
	argon := argon2.DefaultConfig()

	// Check username length
	if len(username) < 2 || len(username) > 25 {
		return nil, errors.New("username must be between 2 and 25 characters")
	}

	// Check if username is not already taken
	if available, err := um.UserStorer.IsUsernameAvailable(username); err != nil {
		return nil, err
	} else if !available {
		return nil, errors.New("this username is already taken")
	}

	// Check if both passwords are equal
	if password1 != password2 {
		return nil, errors.New("passwords don't match")
	}

	// Check if password is at least 8 characters
	if len(password1) < 8 {
		return nil, errors.New("passwords must be at least 8 characters long")
	}

	// Hash & encode password
	encoded, err := argon.HashEncoded([]byte(password1))
	if err != nil {
		return nil, errors.New("an error occured while creating your account")
	}

	// Add user to DB
	user := &model.User{
		ID:       primitive.NewObjectID(),
		Name:     username,
		Password: string(encoded),
		IsOwner:  !um.UserStorer.IsOwnerPresent(),
		IsAdmin:  isAdmin,
	}

	err = um.UserStorer.AddUser(user)
	if err != nil {
		return nil, fmt.Errorf("error adding user: %w", err)
	}

	return user, nil
}

// CheckLogin checks that the login is correct and returns the user it corresponds to
func (um UserManager) CheckLogin(username, password string) (user *model.User, err error) {
	// Check username length
	if len(username) < 2 || len(username) > 25 {
		return nil, errors.New("username must be between 2 and 25 characters")
	}

	// Fetch encoded password from DB
	user = &model.User{}
	if err := um.UserStorer.GetUserFromName(username, user); err != nil {
		return nil, errors.New("authentication failed")
	}

	// Check if the username/password combination is valid
	if ok, err := argon2.VerifyEncoded([]byte(password), []byte(user.Password)); err != nil {
		return nil, errors.New("an error occured while logging you in")
	} else if !ok {
		return nil, errors.New("authentication failed")
	}

	return user, nil
}

// SetUserPassword checks that the password change follows specific rules and updates it in the database
func (um UserManager) SetUserPassword(username, oldPassword, password1, password2 string) error {
	argon := argon2.DefaultConfig()

	// Check new passwords match
	if password1 != password2 {
		return errors.New("new passwords don't match")
	}

	// Check password length
	if len(password1) < 8 {
		return errors.New("passwords must be at least 8 characters long")
	}

	// Fetch encoded password from DB
	var userDB model.User
	if err := um.UserStorer.GetUserFromName(username, &userDB); err != nil {
		return errors.New("an error occured while checking for your password")
	}

	// Check if the username/password combination is valid
	ok, err := argon2.VerifyEncoded([]byte(oldPassword), []byte(userDB.Password))
	if err != nil {
		return errors.New("an error occured while checking for your password")
	}
	if !ok {
		return errors.New("authentication failed")
	}

	// Hash & encode password
	encoded, err := argon.HashEncoded([]byte(password1))
	if err != nil {
		return errors.New("an error occured while saving your password")
	}

	if err := um.UserStorer.SetUserPassword(userDB.ID, string(encoded)); err != nil {
		return errors.New("an error occured while saving your password")
	}

	return nil
}
