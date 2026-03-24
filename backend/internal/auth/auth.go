package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret []byte

// SetJWTSecret configures the HMAC key for JWT signing. Call once at process startup.
func SetJWTSecret(secret string) error {
	if strings.TrimSpace(secret) == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	jwtSecret = []byte(secret)
	return nil
}

func signingKey() ([]byte, error) {
	if len(jwtSecret) == 0 {
		return nil, fmt.Errorf("auth: SetJWTSecret was not called")
	}
	return jwtSecret, nil
}

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GenerateToken(userID, email string) (string, error) {
	key, err := signingKey()
	if err != nil {
		return "", err
	}
	claims := &Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(key)
}

func ValidateToken(tokenString string) (*Claims, error) {
	key, err := signingKey()
	if err != nil {
		return nil, err
	}
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return key, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
