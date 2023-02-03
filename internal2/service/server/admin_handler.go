package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Agurato/starfin/internal2/model"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AdminStorer interface {
	GetVolumes() (volumes []model.Volume, err error)
	GetVolumeFromID(id primitive.ObjectID, volume *model.Volume) error
	DeleteVolume(hexId string) error

	GetUsers() (users []model.User, err error)
	GetUserFromID(id primitive.ObjectID, user *model.User) error
	DeleteUser(hexId string) error

	GetFilms() (films []model.Film)
	GetFilmFromID(id primitive.ObjectID) (film model.Film, err error)
	GetPersonFromTMDBID(ID int64) (person model.Person, err error)
}

type AdminHandler struct {
	AdminStorer
}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
}

// GetAdmin displays the admin page
func (ah AdminHandler) GetAdmin(c *gin.Context) {
	var (
		volumesWithStringID []gin.H
		usersWithStringID   []gin.H
	)

	volumes, err := ah.AdminStorer.GetVolumes()
	if err != nil {
		log.Errorln(err)
		RenderHTML(c, http.StatusOK, "pages/admin.go.html", gin.H{
			"title":   "Admin",
			"volumes": volumesWithStringID,
			"users":   usersWithStringID,
			"error":   "An error occured …",
		})
	}
	for _, vol := range volumes {
		volumesWithStringID = append(volumesWithStringID, gin.H{
			"id":  vol.ID.Hex(),
			"obj": vol,
		})
	}

	users, err := ah.AdminStorer.GetUsers()
	if err != nil {
		log.Errorln(err)
		RenderHTML(c, http.StatusOK, "pages/admin.go.html", gin.H{
			"title":   "Admin",
			"volumes": volumesWithStringID,
			"users":   usersWithStringID,
			"error":   "An error occured …",
		})
	}
	for _, user := range users {
		usersWithStringID = append(usersWithStringID, gin.H{
			"id":  user.ID.Hex(),
			"obj": user,
		})
	}
	RenderHTML(c, http.StatusOK, "pages/admin.go.html", gin.H{
		"title":   "Admin",
		"volumes": volumesWithStringID,
		"users":   usersWithStringID,
	})
}

// GetAdminVolume displays the volume edit page
func (ah AdminHandler) GetAdminVolume(c *gin.Context) {
	volumeIdStr := c.Param("volumeId")

	// If we're adding a new volume
	if volumeIdStr == "new" {
		RenderHTML(c, http.StatusOK, "pages/admin_volume.go.html", gin.H{
			"title":  "Add new volume",
			"volume": model.Volume{},
			"new":    true,
		})
		return
	}

	volumeId, err := primitive.ObjectIDFromHex(volumeIdStr)
	if err != nil {
		RenderHTML(c, http.StatusOK, "pages/admin_volume.go.html", gin.H{
			"title": "Edit volume",
			"error": "Incorrect volume ID!",
		})
	}
	var volume model.Volume
	if err := ah.AdminStorer.GetVolumeFromID(volumeId, &volume); err != nil {
		RenderHTML(c, http.StatusOK, "pages/admin_volume.go.html", gin.H{
			"title": "Edit volume",
			"error": "Volume does not exist!",
		})
		return
	}
	RenderHTML(c, http.StatusOK, "pages/admin_volume.go.html", gin.H{
		"title":  "Edit volume",
		"volume": volume,
		"id":     volume.ID.Hex(),
		"new":    false,
	})
}

// PostEditVolume handles editing (and adding) a volume from POST request
func (ah AdminHandler) PostEditVolume(c *gin.Context) {
	volumeIdStr := c.PostForm("id")
	volumeName := strings.Trim(c.PostForm("name"), " ")
	volumePath := strings.Trim(c.PostForm("path"), " ")
	volumeIsRecursive := c.PostForm("recursive") == "recursive"
	volumeMediaType := c.PostForm("mediatype") // "Film" or "TV"

	if volumeIdStr == "" {
		volume := &model.Volume{
			ID:          primitive.NewObjectID(),
			Name:        volumeName,
			Path:        volumePath,
			IsRecursive: volumeIsRecursive,
			MediaType:   volumeMediaType,
		}
		// Adding a volume
		if err := AddVolume(volume); err != nil {
			RenderHTML(c, http.StatusUnauthorized, "pages/admin_volume.go.html", gin.H{
				"title":  "Add new volume",
				"volume": model.Volume{},
				"new":    true,
				"error":  err.Error(),
			})
			return
		}
	} else {
		// TODO: editing a volume
		var volume model.Volume
		volumeID, _ := primitive.ObjectIDFromHex(volumeIdStr)
		ah.AdminStorer.GetVolumeFromID(volumeID, &volume)
		RenderHTML(c, http.StatusUnauthorized, "pages/admin_volume.go.html", gin.H{
			"title":  "Edit volume",
			"volume": volume,
			"id":     volume.ID.Hex(),
			"new":    false,
			"error":  "This functionality is not available yet!",
		})
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin")
}

// PostDeleteVolume deletes a volume from a POST request
func (ah AdminHandler) PostDeleteVolume(c *gin.Context) {
	volumeID := c.PostForm("volumeId")

	err := ah.AdminStorer.DeleteVolume(volumeID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err})
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Volume #%s deleted", volumeID)})
}

// GetAdminUser displays the user edit page
func (ah AdminHandler) GetAdminUser(c *gin.Context) {
	userIdStr := c.Param("userId")

	// If we're adding a new user
	if userIdStr == "new" {
		RenderHTML(c, http.StatusOK, "pages/admin_user.go.html", gin.H{
			"title": "Add new user",
			"user":  model.User{},
			"new":   true,
		})
		return
	}

	userId, err := primitive.ObjectIDFromHex(userIdStr)
	if err != nil {
		RenderHTML(c, http.StatusOK, "pages/admin_user.go.html", gin.H{
			"title": "Edit user",
			"error": "Incorrect user ID!",
		})
	}
	var user model.User
	if err := ah.AdminStorer.GetUserFromID(userId, &user); err != nil {
		RenderHTML(c, http.StatusOK, "pages/admin_user.go.html", gin.H{
			"title": "Edit user",
			"error": "User does not exist!",
		})
		return
	}
	RenderHTML(c, http.StatusOK, "pages/admin_user.go.html", gin.H{
		"title":    "Edit user",
		"userEdit": user,
		"id":       user.ID.Hex(),
		"new":      false,
	})
}

// PostEditUser handles editing (and adding) a user from POST request
func (ah AdminHandler) PostEditUser(c *gin.Context) {
	userIdStr := c.PostForm("id")
	username := strings.Trim(c.PostForm("username"), " ")
	password1 := strings.Trim(c.PostForm("password1"), " ")
	password2 := strings.Trim(c.PostForm("password2"), " ")
	isAdmin := c.PostForm("isadmin") == "isadmin"

	if userIdStr == "" {
		if _, err := AddUser(username, password1, password2, isAdmin); err != nil {
			RenderHTML(c, http.StatusUnauthorized, "pages/admin_user.go.html", gin.H{
				"title":    "Add new user",
				"userEdit": model.User{},
				"new":      true,
				"error":    err.Error(),
			})
			return
		}
	} else {
		// TODO: editing a user
		var user model.User
		userID, _ := primitive.ObjectIDFromHex(userIdStr)
		ah.AdminStorer.GetUserFromID(userID, &user)
		RenderHTML(c, http.StatusUnauthorized, "pages/admin_user.go.html", gin.H{
			"title":    "Edit user",
			"userEdit": user,
			"id":       user.ID.Hex(),
			"new":      false,
			"error":    "This functionality is not available yet!",
		})
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin")
}

// PostDeleteVolume deletes a user from a POST request
func (ah AdminHandler) PostDeleteUser(c *gin.Context) {
	userID := c.PostForm("userId")

	err := ah.AdminStorer.DeleteUser(userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err})
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("User #%s deleted", userID)})
}

// PostReloadCache reloads the cache
func (ah AdminHandler) PostReloadCache(c *gin.Context) {
	films := ah.AdminStorer.GetFilms()
	for _, film := range films {
		cachePosterAndBackdrop(&film)
		for _, personID := range film.GetCastAndCrewIDs() {
			person, _ := ah.AdminStorer.GetPersonFromTMDBID(personID)
			cachePersonPhoto(&person)
		}
	}

	c.JSON(http.StatusOK, gin.H{})
}

// PostEditFilmOnline handle editing a film from online link
func (ah AdminHandler) PostEditFilmOnline(c *gin.Context) {
	inputUrl := c.PostForm("url")
	filmID := c.PostForm("filmID")

	tmdbID, err := GetTMDBIDFromLink(inputUrl)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Wrong URL format"})
		return
	}

	objID, err := primitive.ObjectIDFromHex(filmID)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Wrong film ID", "filmID": filmID})
		return
	}
	film, err := ah.AdminStorer.GetFilmFromID(objID)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Could not get film from database"})
		return
	}
	film.TMDBID = int(tmdbID)
	film.FetchDetails()
	err = tryAddFilmToDB(&film, true)
	if err != nil {
		log.Warning(err)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Could not update film in database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tmdbID": tmdbID})
}
