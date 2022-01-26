package server

import (
	"fmt"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Handle404 displays the 404 page
func Handle404(c *gin.Context) {
	RenderHTML(c, http.StatusNotFound, "pages/404.html", gin.H{
		"title": "404 - Not Found",
	})
}

// HandleGETStart allows regsitration of first user (admin)
func HandleGETStart(c *gin.Context) {
	if GetUserNb() > 0 {
		// TODO: log
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	}
	RenderHTML(c, http.StatusOK, "pages/start.html", gin.H{
		"title": "Create admin account",
	})
}

// HandleGETIndex displays the index page
func HandleGETIndex(c *gin.Context) {
	// TODO: display recently added movies in index page
	RenderHTML(c, http.StatusOK, "pages/index.html", gin.H{
		"title": "down-low-d",
	})
}

// HandleGETLogin displays the registration page
func HandleGETLogin(c *gin.Context) {
	RenderHTML(c, http.StatusOK, "pages/login.html", gin.H{
		"title": "Login",
	})
}

// HandleGETLogout logs out the user and redirects to index
func HandleGETLogout(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(UserKey)

	if user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The session could not be found"})
		return
	}

	session.Delete(UserKey)
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Server had problem to log you out"})
	}

	c.Redirect(http.StatusTemporaryRedirect, "/")
}

// HandleGETSearch displays the search & results page
func HandleGETSearch(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(UserKey).(User)

	searchQuery := c.Query("q")

	if len(searchQuery) == 0 {
		RenderHTML(c, http.StatusOK, "pages/search.html", gin.H{
			"title":       "Search",
			"searchQuery": searchQuery,
		})
		return
	}

	searchResults, err := MediaSearchMulti(searchQuery, user)
	if err != nil {
		RenderHTML(c, http.StatusServiceUnavailable, "pages/search.html", gin.H{
			"title": "Search",
			"error": "An error occured during the search",
		})
	}

	RenderHTML(c, http.StatusOK, "pages/search.html", gin.H{
		"title":         "Search",
		"searchQuery":   searchQuery,
		"searchResults": searchResults,
	})
}

// HandleGETMovie displays information about a movie
func HandleGETMovie(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(UserKey).(User)

	// Fetch info from TMDB
	// tmdbIDInt, err := strconv.Atoi(c.Param("tmdbId"))
	// if err != nil {
	// 	RenderHTML(c, http.StatusOK, "pages/media.html", gin.H{
	// 		"title": "Movie",
	// 		"error": "This ID could not be found",
	// 	})
	// 	return
	// }

	var userDB User
	if err := mongoUsers.FindOne(MongoCtx, bson.M{"_id": user.ID}).Decode(&userDB); err != nil {
		// TODO
	}

	RenderHTML(c, http.StatusOK, "pages/media.html", gin.H{})
}

// HandleGETSettings displays the user settings page
func HandleGETSettings(c *gin.Context) {
	success := ""
	setPassword := c.Query("setpassword")
	if setPassword == "success" {
		success = "Password changed successfully"
	}
	RenderHTML(c, http.StatusOK, "pages/settings.html", gin.H{
		"title":   "Settings",
		"success": success,
	})
}

// HandleGETUser displays the page of a user
func HandleGETUser(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(UserKey).(User)

	var userDB User

	// Check for user existence and fetch all queries
	if err := GetUserFromID(user.ID, &userDB); err != nil {
		RenderHTML(c, http.StatusOK, "pages/user.html", gin.H{
			"title": fmt.Sprintf("%s's profile", c.Param("userId")),
			"error": "User does not exist!",
		})
		return
	}

	RenderHTML(c, http.StatusOK, "pages/user.html", gin.H{
		"title": fmt.Sprintf("%s's profile", c.Param("userId")),
		"user":  c.Param("userId"),
	})
}

// HandleGETAdmin displays the admin page
func HandleGETAdmin(c *gin.Context) {
	volumes := GetVolumes()
	var volumesWithStringID []gin.H
	for _, vol := range volumes {
		volumesWithStringID = append(volumesWithStringID, gin.H{
			"id":  vol.ID.Hex(),
			"obj": vol,
		})
	}
	RenderHTML(c, http.StatusOK, "pages/admin.html", gin.H{
		"title":   "Admin",
		"volumes": volumesWithStringID,
	})
}

func HandleGETVolume(c *gin.Context) {
	volumeIdStr := c.Param("volumeId")

	// If we're adding a new volume
	if volumeIdStr == "new" {
		RenderHTML(c, http.StatusOK, "pages/volume.html", gin.H{
			"title":  "Add new volume",
			"volume": Volume{},
			"new":    true,
		})
		return
	}

	volumeId, err := primitive.ObjectIDFromHex(volumeIdStr)
	if err != nil {
		RenderHTML(c, http.StatusOK, "pages/volume.html", gin.H{
			"title": "Edit volume",
			"error": "Incorrect volume ID!",
		})
	}
	var volume Volume
	if err := GetVolumeFromID(volumeId, &volume); err != nil {
		RenderHTML(c, http.StatusOK, "pages/volume.html", gin.H{
			"title": "Edit volume",
			"error": "Volume does not exist!",
		})
		return
	}
	RenderHTML(c, http.StatusOK, "pages/volume.html", gin.H{
		"title":  "Edit volume",
		"volume": volume,
		"new":    false,
	})
}
