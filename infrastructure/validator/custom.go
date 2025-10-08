package validator

import (
	"regexp"
	"unicode"

	"github.com/go-playground/validator/v10"
)

func validatePinStrength(fl validator.FieldLevel) bool {
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

func validatePasswordStrength(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	if len(password) < 7 {
		return false
	}

	hasUppercase := false
	hasLowercase := false
	hasDigit := false
	hasSpecialChar := false

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUppercase = true
		case unicode.IsLower(char):
			hasLowercase = true
		case unicode.IsDigit(char):
			hasDigit = true
		case !unicode.IsLetter(char) && !unicode.IsDigit(char):
			hasSpecialChar = true
		}
	}

	return hasUppercase && hasLowercase && hasDigit && hasSpecialChar
}

func validateNameWithSpecialChars(fl validator.FieldLevel) bool {
	name := fl.Field().String()
	regex := regexp.MustCompile(`^[\p{L}'\-]+$`)
	return regex.MatchString(name)
}
