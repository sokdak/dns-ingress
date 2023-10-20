package common

import (
	"crypto/md5"
	"encoding/hex"
	"math/rand"
	"time"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")

func init() {
	rand.NewSource(time.Now().UnixNano())
}

func GenerateRandomRunes(n int) string {
	runes := make([]rune, n)
	for i := 0; i < n; i++ {
		runes[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(runes)
}

func GenerateMD5Hash(in string) string {
	hash := md5.Sum([]byte(in))
	return hex.EncodeToString(hash[:])
}
