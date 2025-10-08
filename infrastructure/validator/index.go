package validator

func init() {
	validate.RegisterValidation("pin", validatePinStrength)
	validate.RegisterValidation("password", validatePasswordStrength)
	validate.RegisterValidation("name_spacial_char", validateNameWithSpecialChars)
}

type Validator struct{}

func (v *Validator) ValidateStruct(payload interface{}) *[]error {
	return validateStruct(payload)
}

func (v *Validator) ValidateValue(value any, rules string) error {
	return validateField(value, rules)
}

var ValidatorInstance = Validator{}
