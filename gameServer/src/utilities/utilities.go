package utilities

import (
	"math/rand"
	"time"

	"github.com/google/uuid"
)

func AutoId() string {
	return uuid.Must(uuid.NewRandom()).String()
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandSequenceOfLength(n int) string {
	rand.Seed(time.Now().UnixNano())

	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func GenerateGameId() string {
	t := time.Now()
	return t.Format("2006-01-02") + ":" + RandSequenceOfLength(6)
}
