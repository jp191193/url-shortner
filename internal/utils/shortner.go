package utils

import (
	"crypto/rand"
	"math/big"
)

const codeLength = 6

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func GenerateShortCode() string {
	code := make([]rune, codeLength)
	for i := range code {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		code[i] = letters[num.Int64()]
	}
	return string(code)
}
