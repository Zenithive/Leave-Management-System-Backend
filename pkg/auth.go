package utils

import (
	"crypto/rand"
	"errors"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

type CustomClaims struct {
	UserID   string `json:"user_id"`
	UserRole string `json:"user_role"`
	jwt.RegisteredClaims
}

// -------------------------
// 1️ Bcrypt functions
// -------------------------

// HashPassword hashes a plain password using bcrypt
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword compares a plain password with hashed password
func CheckPassword(password, hashed string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password))

	return err == nil
}

// GenerateSecurePassword generates a secure random password
// Format: 3 uppercase + 3 lowercase + 2 digits + 2 special chars = 12 characters
// Example: ABCDefgh12@#
func GenerateSecurePassword() (string, error) {
	const (
		uppercaseLetters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		lowercaseLetters = "abcdefghijklmnopqrstuvwxyz"
		digits           = "0123456789"
		specialChars     = "@#$%&*!?"
	)

	// Generate 3 uppercase letters
	uppercase, err := generateRandomString(uppercaseLetters, 3)
	if err != nil {
		return "", err
	}

	// Generate 3 lowercase letters
	lowercase, err := generateRandomString(lowercaseLetters, 3)
	if err != nil {
		return "", err
	}

	// Generate 2 digits
	digit, err := generateRandomString(digits, 2)
	if err != nil {
		return "", err
	}

	// Generate 2 special characters
	special, err := generateRandomString(specialChars, 2)
	if err != nil {
		return "", err
	}

	// Combine all parts: Uppercase + Lowercase + Digits + Special
	password := uppercase + lowercase + digit + special

	return password, nil
}

// generateRandomString generates a random string of specified length from given charset
func generateRandomString(charset string, length int) (string, error) {
	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := 0; i < length; i++ {
		randomIndex, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}
		result[i] = charset[randomIndex.Int64()]
	}

	return string(result), nil
}

// -------------------------
// 2️ JWT functions
// -------------------------

// GenerateToken generates a JWT token with userID payload
func GenerateToken(userID string, userRole string, jwtKey string) (string, error) {
	claims := CustomClaims{
		UserID:   userID,
		UserRole: userRole,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtKey))
}

// ValidateToken validates a JWT token and returns userID
func ValidateToken(tokenString string, jwtKey string) (*CustomClaims, error) {
	claims := &CustomClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(jwtKey), nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

func GenerateExpiredToken(userID string, userRole string, jwtKey string) (string, error) {
	claims := CustomClaims{
		UserID:   userID,
		UserRole: userRole,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)), // already expired
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Minute)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtKey))
}
