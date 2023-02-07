package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Agurato/starfin/internal2/model"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type AdminFilmManager interface {
	CacheFilms()

	GetFilm(filmHexID string) (*model.Film, error)
	EditFilmWithLink(filmID, inputUrl string) error
}

type AdminUserManager interface {
	GetUsers() ([]model.User, error)
	GetUser(userHexID string) (*model.User, error)

	CreateUser(username, password1, password2 string, isAdmin, isOwner bool) (*model.User, error)

	DeleteUser(userHexID string) error
}

type AdminVolumeManager interface {
	GetVolumes() ([]model.Volume, error)
	GetVolume(volumeHexID string) (*model.Volume, error)

	CreateVolume(name, path string, isRecursive bool, mediaType string) error

	DeleteVolume(volumeHexID string) error
}

type AdminHandler struct {
	AdminFilmManager
	AdminUserManager
	AdminVolumeManager
}

func NewAdminHandler(fm AdminFilmManager, um AdminUserManager, vm AdminVolumeManager) *AdminHandler {
	return &AdminHandler{
		AdminFilmManager:   fm,
		AdminUserManager:   um,
		AdminVolumeManager: vm,
	}
}

// GETAdmin displays the admin page
func (ah AdminHandler) GETAdmin(c *gin.Context) {
	var allErr error
	volumes, err := ah.AdminVolumeManager.GetVolumes()
	if err != nil {
		log.Errorln(fmt.Errorf("error while fetching volumes: %w", err))
		allErr = errors.Join(allErr, err)
	}

	users, err := ah.AdminUserManager.GetUsers()
	if err != nil {
		log.Errorln(fmt.Errorf("error while fetching users: %w", err))
		allErr = errors.Join(allErr, err)
	}

	if allErr != nil {
		RenderHTML(c, http.StatusOK, "pages/admin.go.html", gin.H{
			"title":   "Admin",
			"volumes": volumes,
			"users":   users,
			"error":   allErr.Error(),
		})
		return
	}

	RenderHTML(c, http.StatusOK, "pages/admin.go.html", gin.H{
		"title":   "Admin",
		"volumes": volumes,
		"users":   users,
	})
}

// GETAdminVolume displays the volume edit page
func (ah AdminHandler) GETAdminVolume(c *gin.Context) {
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

	volume, err := ah.AdminVolumeManager.GetVolume(volumeIdStr)
	if err != nil {
		RenderHTML(c, http.StatusOK, "pages/admin_volume.go.html", gin.H{
			"title": "Edit volume",
			"error": err.Error(),
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

// POSTEditVolume handles editing (and adding) a volume from POST request
func (ah AdminHandler) POSTEditVolume(c *gin.Context) {
	volumeIdStr := c.PostForm("id")
	volumeName := strings.Trim(c.PostForm("name"), " ")
	volumePath := strings.Trim(c.PostForm("path"), " ")
	volumeIsRecursive := c.PostForm("recursive") == "recursive"
	volumeMediaType := c.PostForm("mediatype") // "Film" or "TV"

	if volumeIdStr == "" {
		err := ah.AdminVolumeManager.CreateVolume(volumeName, volumePath, volumeIsRecursive, volumeMediaType)
		if err != nil {
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
		volume, _ := ah.AdminVolumeManager.GetVolume(volumeIdStr)
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

// POSTDeleteVolume deletes a volume from a POST request
func (ah AdminHandler) POSTDeleteVolume(c *gin.Context) {
	volumeID := c.PostForm("volumeId")

	err := ah.AdminVolumeManager.DeleteVolume(volumeID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err})
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Volume #%s deleted", volumeID)})
}

// GETAdminUser displays the user edit page
func (ah AdminHandler) GETAdminUser(c *gin.Context) {
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

	user, err := ah.AdminUserManager.GetUser(userIdStr)
	if err != nil {
		RenderHTML(c, http.StatusOK, "pages/admin_user.go.html", gin.H{
			"title": "Edit user",
			"error": err.Error(),
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

// POSTEditUser handles editing (and adding) a user from POST request
func (ah AdminHandler) POSTEditUser(c *gin.Context) {
	userIdStr := c.PostForm("id")
	username := strings.Trim(c.PostForm("username"), " ")
	password1 := strings.Trim(c.PostForm("password1"), " ")
	password2 := strings.Trim(c.PostForm("password2"), " ")
	isAdmin := c.PostForm("isadmin") == "isadmin"

	if userIdStr == "" {
		_, err := ah.AdminUserManager.CreateUser(username, password1, password2, isAdmin, false)
		if err != nil {
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
		user, _ := ah.AdminUserManager.GetUser(userIdStr)
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

// POSTDeleteUser deletes a user from a POST request
func (ah AdminHandler) POSTDeleteUser(c *gin.Context) {
	userID := c.PostForm("userId")

	err := ah.AdminUserManager.DeleteUser(userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err})
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("User #%s deleted", userID)})
}

// POSTReloadCache reloads the cache
func (ah AdminHandler) POSTReloadCache(c *gin.Context) {
	ah.AdminFilmManager.CacheFilms()

	c.JSON(http.StatusOK, gin.H{})
}

// POSTEditFilmOnline handle editing a film from online link
func (ah AdminHandler) POSTEditFilmOnline(c *gin.Context) {
	inputUrl := c.PostForm("url")
	filmID := c.PostForm("filmID")

	err := ah.AdminFilmManager.EditFilmWithLink(filmID, inputUrl)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}
