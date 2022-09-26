package media

import (
	"os"
	"strconv"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestMain(m *testing.M) {
	godotenv.Load("../../.env")
	InitTMDB()
	result := m.Run()
	os.Exit(result)
}

func TestGetIMDbRating(t *testing.T) {
	value, err := strconv.ParseFloat(GetIMDbRating("tt0183649"), 32)
	assert.Nil(t, err)
	assert.Greater(t, value, float64(0))
	assert.LessOrEqual(t, value, float64(10))
}

func TestGetLetterboxdRating(t *testing.T) {
	value, err := strconv.ParseFloat(GetLetterboxdRating("tt0183649"), 32)
	assert.Nil(t, err)
	assert.Greater(t, value, float64(0))
	assert.LessOrEqual(t, value, float64(10))
}

type filmTest struct {
	fileName string
	filmName string
	year     int
}

func TestNewFilm(t *testing.T) {
	expected := []filmTest{
		{
			fileName: "1917.2019.mkv",
			filmName: "1917",
			year:     2019,
		},
	}

	for _, exp := range expected {
		film := NewFilm(exp.fileName, primitive.NewObjectID(), nil)
		assert.Equal(t, exp.filmName, film.Name)
		assert.Equal(t, exp.year, film.ReleaseYear)

		assert.NoError(t, film.FetchTMDBID())
		film.FetchDetails()
		assert.Contains(t, film.Cast, Cast{Character: "Lance Corporal Schofield", ActorID: 146750})
	}
}
