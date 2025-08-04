package validator

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

// isARN checks if a string is a valid AWS ARN.
func isARN(fl validator.FieldLevel) bool {
	arnRegex := `^arn:aws:[a-z0-9\-]+:[a-z0-9\-]*:[0-9]{12}:.*$`
	re := regexp.MustCompile(arnRegex)
	return re.MatchString(fl.Field().String())
}

// RegisterCustomValidators registers custom validation functions with the validator.
func RegisterCustomValidators(validate *validator.Validate) {
	validate.RegisterValidation("arn", isARN)
}