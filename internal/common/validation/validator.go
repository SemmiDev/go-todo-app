// Package validation provides a centralized struct validator with custom rules.
package validation

import (
	"errors"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/semmidev/todo-app/internal/common/apperr"
	"time"
)

// Validator wraps the go-playground/validator instance to provide application-specific validation logic.
type Validator struct {
	validate *validator.Validate
}

var hexColorRegex = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3,4}){1,2}$`)

// New creates and configures a new application validator with custom tag functions and rules.
func New() *Validator {
	validate := validator.New()

	// 1. Register a TagNameFunc to extract the JSON tag for field names
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	// 2. Register custom validation tag: "iscolor" for HEX #FFFFFF or similar
	_ = validate.RegisterValidation("iscolor", func(fl validator.FieldLevel) bool {
		val := fl.Field().String()
		return hexColorRegex.MatchString(val)
	})

	// 3. Register custom validation tag: "duration" for time.ParseDuration valid strings
	_ = validate.RegisterValidation("duration", func(fl validator.FieldLevel) bool {
		val := fl.Field().String()
		_, err := time.ParseDuration(val)
		return err == nil
	})

	return &Validator{
		validate: validate,
	}
}

// Struct validates a struct and maps validator.ValidationErrors to apperr.ValidationError.
func (v *Validator) Struct(s interface{}) error {
	err := v.validate.Struct(s)
	if err == nil {
		return nil
	}

	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		appErr := &apperr.ValidationError{
			Errors: make([]apperr.FieldError, 0, len(validationErrors)),
		}
		for _, e := range validationErrors {
			// Extract custom messages by the tag failed
			msg := buildErrorMessage(e)
			appErr.Errors = append(appErr.Errors, apperr.FieldError{
				Field:   e.Field(), // Thanks to TagNameFunc, this will be the JSON tag!
				Message: msg,
			})
		}
		return appErr
	}

	// Unlikely to happen unless the passed argument wasn't a struct.
	return err
}

// buildErrorMessage converts a validation tag into a human-readable message.
func buildErrorMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return "must be at least " + e.Param()
	case "max":
		return "must be at most " + e.Param()
	case "iscolor":
		return "must be a valid HEX color code (e.g. #FFFFFF)"
	case "uuid":
		return "must be a valid UUID"
	case "duration":
		return "must be a valid duration (e.g. 1h, 15m)"
	default:
		return "violates constraint '" + e.Tag() + "'"
	}
}
