package auth

import "golang.org/x/crypto/bcrypt"

const bcryptCost = 12

var hashPasswordFn = bcrypt.GenerateFromPassword

func HashPassword(password string) (string, error) {
	b, err := hashPasswordFn([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
