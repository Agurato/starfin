package server

import (
	"bytes"
	"context"
	"net/http"
	"strconv"
	"text/template"

	"github.com/Agurato/starfin/internal/model"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type TorrentStorer interface {
	SearchTorrents(ctx context.Context, search, category string, page uint) ([]model.RarbgTorrent, error)
	GetTorrents(ctx context.Context, imdbID string, offset, limit int64) ([]model.RarbgTorrent, error)
	GetAllTVTorrents(ctx context.Context, offset, limit int64) (torrents []model.RarbgTorrent, err error)
	GetTVTorrents(ctx context.Context, imdbID, season, episode string, offset, limit int64) (torrents []model.RarbgTorrent, err error)
}

type RarbgHandler struct {
	TorrentStorer
	torznabAPIKey string
	template      *template.Template
}

func NewRarbgHandler(ts TorrentStorer, torznabAPIKey string) *RarbgHandler {
	funcMap := template.FuncMap{
		"torznabID": func(category string) int {
			return model.GetTorznabID(model.RarbgCategory(category))
		},
	}
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob("web/templates/torznab/*.go.xml"))
	return &RarbgHandler{
		TorrentStorer: ts,
		torznabAPIKey: torznabAPIKey,
		template:      tmpl,
	}
}

// GETTorrents displays the list of torrents
func (rh RarbgHandler) GETTorrents(c *gin.Context) {
	pageInput := c.Query("page")
	var page uint64 = 1
	var err error
	if pageInput != "" {
		page, err = strconv.ParseUint(pageInput, 10, 32)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
	}

	category := c.Query("cat")
	search := c.Query("search")

	torrents, err := rh.TorrentStorer.SearchTorrents(c, search, category, uint(page))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	RenderHTML(c, http.StatusOK, "pages/torrents.go.html", gin.H{
		"title":      "Torrents",
		"torrents":   torrents,
		"search":     search,
		"cat":        category,
		"categories": []string{"ebooks", "games", "movies", "music", "software", "tv"},
	})
}

func (rh RarbgHandler) GETTorznab(c *gin.Context) {
	var buf bytes.Buffer

	if rh.torznabAPIKey != c.Query("apikey") {
		err := rh.template.ExecuteTemplate(&buf, "torznab/error.go.xml", gin.H{
			"code":  100,
			"error": "Invalid API Key",
		})
		if err != nil {
			log.Errorln(err)
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.Data(http.StatusOK, "application/xml", buf.Bytes())
		return
	}

	topic := c.Query("t")
	var err error
	switch topic {
	case "caps":
		err = rh.template.ExecuteTemplate(&buf, "torznab/caps.go.xml", nil)
	case "search":
		err = rh.template.ExecuteTemplate(&buf, "torznab/search.go.xml", nil)
	case "movie":
		imdbID := c.Query("imdbid")
		offset, _ := strconv.ParseInt(c.Query("offset"), 10, 64)
		limit, _ := strconv.ParseInt(c.Query("limit"), 10, 64)

		var torrents []model.RarbgTorrent
		torrents, err = rh.TorrentStorer.GetTorrents(c, imdbID, offset, limit)

		if err != nil {
			log.Errorln(err)
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		err = rh.template.ExecuteTemplate(&buf, "torznab/movie.go.xml", torrents)
	case "tvsearch":
		imdbID := c.Query("imdbid")
		query := c.Query("q")
		season := c.Query("season")
		episode := c.Query("ep")
		offset, _ := strconv.ParseInt(c.Query("offset"), 10, 64)
		limit, _ := strconv.ParseInt(c.Query("limit"), 10, 64)

		var torrents []model.RarbgTorrent
		if imdbID == "" && query == "" {
			torrents, err = rh.TorrentStorer.GetAllTVTorrents(c, offset, limit)
		} else {
			torrents, err = rh.TorrentStorer.GetTVTorrents(c, imdbID, season, episode, offset, limit)
		}

		if err != nil {
			log.Errorln(err)
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		err = rh.template.ExecuteTemplate(&buf, "torznab/tvsearch.go.xml", gin.H{
			"torrents": torrents,
			"season":   season,
		})
	default:
		err = rh.template.ExecuteTemplate(&buf, "torznab/error.go.xml", gin.H{
			"code":  203,
			"error": "Function not available",
		})
		c.Data(http.StatusBadRequest, "application/xml", buf.Bytes())
		return
	}

	if err != nil {
		log.Errorln(err)
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.Data(http.StatusOK, "application/xml", buf.Bytes())

	return
}
