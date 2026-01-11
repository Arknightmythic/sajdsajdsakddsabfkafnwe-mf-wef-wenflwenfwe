package util

import (
	"crypto/sha256"
	"dokuprime-be/config"
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
)

func GenerateDeterministicHash(password string) (string, error) {
	salt := config.AppConfig.BcryptSalt
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
	salt := config.AppConfig.BcryptSalt
	if salt == "" {
		salt = "default-salt-please-change-in-production"
	}

	saltedPassword := password + salt

	hasher := sha256.New()
	hasher.Write([]byte(saltedPassword))
	hashedInput := base64.StdEncoding.EncodeToString(hasher.Sum(nil))

	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(hashedInput))
}
