package infrastructure

import (
	"os"
	"strconv"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

var mw *MetadataWrapper

func TestMain(m *testing.M) {
	godotenv.Load("../../.env")
	mw, _ = NewMetadataWrapper(os.Getenv("TMDB_API_KEY"))
	result := m.Run()
	os.Exit(result)
}

func TestGetIMDbRating(t *testing.T) {
	value, err := strconv.ParseFloat(mw.getIMDbRating("tt0183649"), 32)
	assert.Nil(t, err)
	assert.Greater(t, value, float64(0))
	assert.LessOrEqual(t, value, float64(10))
}

func TestGetLetterboxdRating(t *testing.T) {
	value, err := strconv.ParseFloat(mw.getLetterboxdRating("tt0183649"), 32)
	assert.Nil(t, err)
	assert.Greater(t, value, float64(0))
	assert.LessOrEqual(t, value, float64(10))
}
