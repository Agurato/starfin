package server_test

import (
	"testing"

	"github.com/Agurato/starfin/internal/server"
	"github.com/stretchr/testify/assert"
)

func TestParseParamsFilters(t *testing.T) {
	params := "/year/2019/genre/comedy/country/france/page/2/"
	yearFilter, years, genre, country, page, err := server.ParseParamsFilters(params)
	assert.Equal(t, yearFilter, "2019")
	assert.Equal(t, years, []int{2019})
	assert.Equal(t, genre, "comedy")
	assert.Equal(t, country, "france")
	assert.Equal(t, page, 2)
	assert.NoError(t, err)

	params = "/year/2010s/"
	yearFilter, years, genre, country, page, err = server.ParseParamsFilters(params)
	assert.Equal(t, yearFilter, "2010s")
	assert.Equal(t, years, []int{2010, 2011, 2012, 2013, 2014, 2015, 2016, 2017, 2018, 2019})
	assert.Equal(t, genre, "")
	assert.Equal(t, country, "")
	assert.Equal(t, page, 1)
	assert.NoError(t, err)
}
