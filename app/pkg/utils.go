package pkg

import (
	"fmt"
	"math/rand"
	"time"

	petname "github.com/dustinkirkland/golang-petname"
)

func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seed := rand.NewSource(time.Now().UnixNano())
	random := rand.New(seed)

	result := make([]byte, length)
	for i := range result {
		result[i] = charset[random.Intn(len(charset))]
	}
	return string(result)
}

func GenerateRandomNumericString(length int) string {
	const charset = "0123456789"
	seed := rand.NewSource(time.Now().UnixNano())
	random := rand.New(seed)

	result := make([]byte, length)
	for i := range result {
		result[i] = charset[random.Intn(len(charset))]
	}
	return string(result)
}

func GenerateRandomBool() bool {
	seed := rand.NewSource(time.Now().UnixNano())
	random := rand.New(seed)
	return random.Intn(2) == 0
}

func GenerateRandomUserName() string {
	return fmt.Sprintf("%s-%s", petname.Generate(2, "-"), GenerateRandomNumericString(4))
}
