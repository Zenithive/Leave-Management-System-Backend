package security

import (
	"golang.org/x/crypto/bcrypt"
)

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
