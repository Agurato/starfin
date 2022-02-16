package media

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
