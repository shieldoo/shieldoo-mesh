package main

import (
	"crypto/rand"
	"encoding/base64"
	"os/user"
	"strings"
)

func GenerateRandomBytes(n int) []byte {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil
	}

	return b
}

func GenerateRandomString(s int) string {
	b := GenerateRandomBytes(s * 3)
	r := base64.URLEncoding.EncodeToString(b)
	r = strings.Replace(r, "=", "0", -1)
	r = strings.Replace(r, "+", "1", -1)
	r = strings.Replace(r, "/", "2", -1)
	return r[:s]
}

func GetHomeDir() string {
	user, err := user.Current()
	if err != nil {
		return ""
	}
	return user.HomeDir
}
