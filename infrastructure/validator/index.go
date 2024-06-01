package validator

func init() {
	validate.RegisterValidation("password", validatePasswordStrength)
	validate.RegisterValidation("name_spacial_char", validateNameWithSpecialChars)
}

type Validator struct{}

func (v *Validator) ValidateStruct(payload interface{}) *[]error {
	return validateStruct(payload)
}

var ValidatorInstance = Validator{}
