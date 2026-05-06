package security

import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func VerifyPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func HashCVV(cvv string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(cvv), bcrypt.DefaultCost)
	return string(b), err
}
