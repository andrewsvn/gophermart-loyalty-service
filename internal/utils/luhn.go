package utils

import (
	"math/rand"
	"regexp"
	"strconv"
)

func GenerateLuhnNumber(rnd *rand.Rand) string {
	intVal := (10000000000 + rnd.Int63()%90000000000) * 10
	strVal := strconv.FormatInt(intVal, 10)
	intVal += int64(10-luhnSum(strVal)) % 10
	return strconv.FormatInt(intVal, 10)
}

func IsValidLuhnNumber(number string) bool {
	re := regexp.MustCompile("^[0-9]+$")
	if !re.Match([]byte(number)) {
		return false
	}
	return luhnSum(number)%10 == 0
}

func luhnSum(number string) int {
	sum := 0
	inv := false
	for i := len(number) - 1; i >= 0; i-- {
		digit := int(number[i] - '0')
		if inv {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		inv = !inv
	}
	return sum % 10
}
