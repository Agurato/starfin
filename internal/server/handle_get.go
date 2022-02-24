package server

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/Agurato/starfin/internal/media"
	"github.com/Agurato/starfin/internal/utilities"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

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
	RenderHTML(c, http.StatusOK, "pages/index.html", gin.H{
		"title": "starfin",
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

// HandleGETMovie displays information about a movie
func HandleGETMovie(c *gin.Context) {
	tmdbID, err := strconv.Atoi(c.Param("tmdbId"))
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}
	movie, err := GetMovieFromID(tmdbID)
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}

	var volumes []string
	for _, path := range movie.Paths {
		var volume media.Volume
		GetVolumeFromID(path.FromVolume, &volume)
		volumes = append(volumes, volume.Name)
	}

	RenderHTML(c, http.StatusOK, "pages/movie.html", gin.H{
		"title":   fmt.Sprintf("%s (%d)", movie.Title, movie.ReleaseYear),
		"movie":   movie,
		"volumes": volumes,
	})
}

func HandleGETDownloadMovie(c *gin.Context) {
	tmdbID, err := strconv.Atoi(c.Param("tmdbId"))
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}
	fileIndex, err := strconv.Atoi(c.Param("idx"))
	if err != nil {
		fileIndex = 0
	}

	movie, err := GetMovieFromID(tmdbID)
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}
	if fileIndex >= len(movie.Paths) {
		fileIndex = len(movie.Paths) - 1
	}
	http.ServeFile(c.Writer, c.Request, movie.Paths[fileIndex].Path)
}

// HandleGETMovies displays the list of movies
func HandleGETMovies(c *gin.Context) {
	movies := GetMovies()
	var (
		search string
		ok     bool
	)

	// Filter movies from search
	if search, ok = c.GetQuery("search"); ok {
		var filteredMovies []media.Movie
		for _, movie := range movies {
			if movie.ContainsSearch(search) {
				filteredMovies = append(filteredMovies, movie)
			}
		}
		movies = filteredMovies
	}

	sort.Slice(movies, func(i, j int) bool {
		titleI := utilities.RemoveArticle(movies[i].Title)
		titleJ := utilities.RemoveArticle(movies[j].Title)
		return titleI < titleJ
	})

	RenderHTML(c, http.StatusOK, "pages/movies.html", gin.H{
		"title":  "Movies",
		"movies": movies,
		"search": search,
	})
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
	users := GetUsers()
	var usersWithStringID []gin.H
	for _, user := range users {
		usersWithStringID = append(usersWithStringID, gin.H{
			"id":  user.ID.Hex(),
			"obj": user,
		})
	}
	RenderHTML(c, http.StatusOK, "pages/admin.html", gin.H{
		"title":   "Admin",
		"volumes": volumesWithStringID,
		"users":   usersWithStringID,
	})
}

func HandleGETAdminVolume(c *gin.Context) {
	volumeIdStr := c.Param("volumeId")

	// If we're adding a new volume
	if volumeIdStr == "new" {
		RenderHTML(c, http.StatusOK, "pages/admin_volume.html", gin.H{
			"title":  "Add new volume",
			"volume": media.Volume{},
			"new":    true,
		})
		return
	}

	volumeId, err := primitive.ObjectIDFromHex(volumeIdStr)
	if err != nil {
		RenderHTML(c, http.StatusOK, "pages/admin_volume.html", gin.H{
			"title": "Edit volume",
			"error": "Incorrect volume ID!",
		})
	}
	var volume media.Volume
	if err := GetVolumeFromID(volumeId, &volume); err != nil {
		RenderHTML(c, http.StatusOK, "pages/admin_volume.html", gin.H{
			"title": "Edit volume",
			"error": "Volume does not exist!",
		})
		return
	}
	RenderHTML(c, http.StatusOK, "pages/admin_volume.html", gin.H{
		"title":  "Edit volume",
		"volume": volume,
		"id":     volume.ID.Hex(),
		"new":    false,
	})
}

func HandleGETAdminUser(c *gin.Context) {
	userIdStr := c.Param("userId")

	// If we're adding a new user
	if userIdStr == "new" {
		RenderHTML(c, http.StatusOK, "pages/admin_user.html", gin.H{
			"title": "Add new user",
			"user":  User{},
			"new":   true,
		})
		return
	}

	userId, err := primitive.ObjectIDFromHex(userIdStr)
	if err != nil {
		RenderHTML(c, http.StatusOK, "pages/admin_user.html", gin.H{
			"title": "Edit user",
			"error": "Incorrect user ID!",
		})
	}
	var user User
	if err := GetUserFromID(userId, &user); err != nil {
		RenderHTML(c, http.StatusOK, "pages/admin_user.html", gin.H{
			"title": "Edit user",
			"error": "User does not exist!",
		})
		return
	}
	RenderHTML(c, http.StatusOK, "pages/admin_user.html", gin.H{
		"title":    "Edit user",
		"userEdit": user,
		"id":       user.ID.Hex(),
		"new":      false,
	})
}
