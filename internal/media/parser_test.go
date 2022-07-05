package media

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFilename(t *testing.T) {
	title, year, err := ParseFilename("Mission.Impossible.3.2011")
	assert.Nil(t, err)
	assert.Equal(t, "Mission.Impossible.3", title)
	assert.Equal(t, "2011", year)
}
