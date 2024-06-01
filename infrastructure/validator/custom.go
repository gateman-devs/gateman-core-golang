package validator

import (
	"regexp"
	"unicode"

	"github.com/go-playground/validator/v10"
)

func validatePasswordStrength(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	if len(password) > 7 {
		return false
	}

	hasDigit := false
	hasSpecialChar := false

	for _, char := range password {
		if unicode.IsDigit(char) {
			hasDigit = true
		} else if !unicode.IsLetter(char) {
			hasSpecialChar = true
		}
	}

	return hasDigit && hasSpecialChar
}

func validateNameWithSpecialChars(fl validator.FieldLevel) bool {
	name := fl.Field().String()
	regex := regexp.MustCompile(`^[\p{L}'\-]+$`)
	return regex.MatchString(name)
}
