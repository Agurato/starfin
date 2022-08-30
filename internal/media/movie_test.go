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

type movieTest struct {
	fileName  string
	movieName string
	year      int
}

func TestNewMovie(t *testing.T) {
	expected := []movieTest{
		{
			fileName:  "1917.2019.mkv",
			movieName: "1917",
			year:      2019,
		},
	}

	for _, exp := range expected {
		movie := NewMovie(exp.fileName, primitive.NewObjectID(), nil)
		assert.Equal(t, exp.movieName, movie.Name)
		assert.Equal(t, exp.year, movie.ReleaseYear)

		assert.NoError(t, movie.FetchTMDBID())
		movie.FetchDetails()
		assert.Contains(t, movie.Cast, Cast{Character: "Lance Corporal Schofield", ActorID: 146750})
	}
}
