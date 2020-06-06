package fetcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getPage(t *testing.T) {
	s := &session{}

	p, err := getPage(s)
	assert.NoError(t, err)
	assert.NotEmpty(t, p)
}

func Test_deguffHTML(t *testing.T) {
	s := &session{}

	p, _ := getPage(s)
	parcels, _ := deguffHTML(p)
	assert.NotEmpty(t, parcels)
}

func Test_authenticate(t *testing.T) {

}
