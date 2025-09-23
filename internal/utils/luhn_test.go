package utils

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLuhnNumbers(t *testing.T) {
	const luhnNumberLen = 12

	const (
		correctLn1 string = "504459260643"
		correctLn2 string = "961684022091"

		incorrectLn1 string = "504456160643"
		incorrectLn2 string = "961684022096"
	)

	assert.True(t, IsValidLuhnNumber(correctLn1))
	assert.True(t, IsValidLuhnNumber(correctLn2))

	assert.False(t, IsValidLuhnNumber(incorrectLn1))
	assert.False(t, IsValidLuhnNumber(incorrectLn2))

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 500; i++ {
		ln := GenerateLuhnNumber(rnd)
		assert.Equal(t, luhnNumberLen, len(ln))

		ok := IsValidLuhnNumber(ln)
		assert.True(t, ok, "Luhn number should be valid: %s", ln)
	}
}
