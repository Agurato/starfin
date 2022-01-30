package media

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type movieTest struct {
	fileName  string
	movieName string
	year      int
}

func TestCreateMediaFromFilename(t *testing.T) {
	expected := []movieTest{
		{
			fileName:  "1917.2019.mkv",
			movieName: "1917",
			year:      2019,
		},
	}

	for _, exp := range expected {
		movie := CreateMediaFromFilename(exp.fileName, primitive.NewObjectID()).(*Movie)
		assert.Equal(t, exp.movieName, movie.Name)
		assert.Equal(t, exp.year, movie.ReleaseYear)
	}
}
