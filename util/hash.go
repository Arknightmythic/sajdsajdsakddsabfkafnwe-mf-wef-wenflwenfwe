package util

import (
	"crypto/sha256"
	"encoding/base64"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func GenerateDeterministicHash(password string) (string, error) {
	salt := os.Getenv("BCRYPT_SALT")
	if salt == "" {
		salt = "default-salt-please-change-in-production"
	}

	saltedPassword := password + salt

	hasher := sha256.New()
	hasher.Write([]byte(saltedPassword))
	hashedInput := base64.StdEncoding.EncodeToString(hasher.Sum(nil))

	hash, err := bcrypt.GenerateFromPassword([]byte(hashedInput), bcrypt.MinCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}


func VerifyPassword(hashedPassword, password string) error {
	salt := os.Getenv("BCRYPT_SALT")
	if salt == "" {
		salt = "default-salt-please-change-in-production"
	}

	saltedPassword := password + salt

	hasher := sha256.New()
	hasher.Write([]byte(saltedPassword))
	hashedInput := base64.StdEncoding.EncodeToString(hasher.Sum(nil))

	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(hashedInput))
}