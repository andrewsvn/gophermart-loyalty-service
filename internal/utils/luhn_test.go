package utils

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLuhnNumbers(t *testing.T) {
	const luhnNumberLen = 12

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 500; i++ {
		ln := GenerateLuhnNumber(rnd)
		assert.Equal(t, luhnNumberLen, len(ln))

		ok := IsValidLuhnNumber(ln)
		assert.True(t, ok, "Luhn number should be valid: %s", ln)
	}
}
