package server_test

import (
	"os"
	"testing"

	"github.com/Agurato/starfin/internal/media"
	"github.com/Agurato/starfin/internal/server"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	media.InitTMDB()
	result := m.Run()
	os.Exit(result)
}

func TestGetTMDBIDFromLink(t *testing.T) {
	url := "thisisnotaurl"
	_, err := server.GetTMDBIDFromLink(url)
	assert.Error(t, err)

	url = "https://www.themoviedb.org/movi"
	_, err = server.GetTMDBIDFromLink(url)
	assert.Error(t, err)

	url = "https://www.themoviedb.org/movie/1817-phone-booth"
	tmdbID, err := server.GetTMDBIDFromLink(url)
	assert.NoError(t, err)
	assert.Equal(t, tmdbID, "1817")

	url = "https://letterboxd.com/film/phone-booth/"
	tmdbID, err = server.GetTMDBIDFromLink(url)
	assert.NoError(t, err)
	assert.Equal(t, tmdbID, "1817")

	url = "https://www.imdb.com/title/tt0183649/"
	tmdbID, err = server.GetTMDBIDFromLink(url)
	assert.NoError(t, err)
	assert.Equal(t, tmdbID, "1817")
}
