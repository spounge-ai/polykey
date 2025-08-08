package validator

import (
	"regexp"
	"github.com/go-playground/validator/v10"
)

func isARN(fl validator.FieldLevel) bool {
	arnRegex := `^arn:aws:[a-z0-9\-]+:[a-z0-9\-]*:[0-9]{12}:.*$`
	re := regexp.MustCompile(arnRegex)
	return re.MatchString(fl.Field().String())
}

func RegisterCustomValidators(validate *validator.Validate) error {
	if err := validate.RegisterValidation("arn", isARN); err != nil {
		return err
	}
	return nil
}